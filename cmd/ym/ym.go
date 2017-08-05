package main

import (
	"net"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/history"
	"github.com/frizinak/ym/player"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
	"github.com/frizinak/ym/terminal"
	"github.com/frizinak/ym/ym"
)

func getPlaylist(cacheDir string) (*playlist.Playlist, error) {
	var f string
	var e bool
	if cacheDir != "" {
		f = path.Join(cacheDir, "playlist.gob")
		if _, err := os.Stat(f); err == nil || !os.IsNotExist(err) {
			e = true
		}
	}

	pl := playlist.New(f, 100)
	if e {
		if err := pl.Load(); err != nil {
			return nil, err
		}
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			if err := pl.Save(true); err != nil {
				panic(err)
			}
		}
	}()

	return pl, nil
}

func getCache(cacheDir string) *cache.Cache {
	if cacheDir != "" {
		dls, _ := cache.New(cacheDir, path.Join(os.TempDir(), "ym"))
		return dls
	}

	return nil
}

func main() {
	yt, err := search.NewYoutube()
	if err != nil {
		panic(err)
	}

	p, err := player.FindSupportedPlayer(
		player.NewMPlayer(),
		player.NewMPV(), // TODO fix pause
		player.NewFFPlay(),
	)

	if err != nil {
		panic(err)
	}

	var cacheDir string
	if user, err := user.Current(); err == nil {
		cacheDir = path.Join(user.HomeDir, ".cache", "ym")
	}

	dls := getCache(cacheDir)
	pl, err := getPlaylist(cacheDir)
	if err != nil {
		panic(err)
	}

	ym := ym.New(
		pl,
		yt,
		p,
		dls,
		&net.TCPAddr{IP: net.IP{127, 0, 0, 1}, Port: 6600},
	)

	// ignore error
	go ym.Listen()

	var cmd *command.Command
	var ocmd *command.Command
	cmds := make([]*command.Command, 0, 1)
	if len(os.Args) > 1 {
		cmds = append(cmds, command.New(strings.Join(os.Args[1:], " ")))
	}

	history := history.New(20)

	playChan := make(chan *command.Command, 2000)
	currentChan := make(chan search.Result, 0)

	statusChan := make(chan string, 0)
	go func() {
		if err := ym.Play(playChan, currentChan, statusChan); err != nil {
			panic(err)
		}
	}()

	terminal.Clear()

	titleChan := make(chan string, 0)
	statusCurrentChan := make(chan search.Result, 0)
	go printStatus(titleChan, statusCurrentChan, statusChan)

	resultsChan := make(chan []search.Result, 0)
	go printResults(resultsChan)

	infoChan := make(chan search.Info, 0)
	go printInfo(infoChan)

	playlistChan := make(chan struct{}, 0)
	go printPlaylist(pl, playlistChan)

	view := "results"

	go func() {
		for r := range currentChan {
			statusCurrentChan <- r
			if view == "playlist" {
				playlistChan <- struct{}{}
			}
		}
	}()

	cacheChan := make(chan search.Result, 2000)
	go func() {
		for entry := range cacheChan {
			if dls.Get(entry.ID()) != nil {
				continue
			}

			u, err := entry.DownloadURL()
			if err != nil {
				continue
			}

			dls.Set(cache.NewEntry(entry.ID(), "mp4", u))
		}
	}()

	for {
		if view == "results" {
			title, r := history.Current()
			titleChan <- title
			resultsChan <- r
		}

		if len(cmds) == 0 {
			if cmds, err = prompt(); err != nil {
				panic(err)
			}
		}

		if len(cmds) == 0 {
			continue
		}

		cmd = cmds[0]
		cmds = cmds[1:]

		if view != "results" && view != "playlist" {
			view = "results"
			continue
		}

		if cmd.Cmd() == ':' {
			switch cmd.Arg() {
			case "list", "queue", "playlist":
				view = "playlist"
				playlistChan <- struct{}{}
				continue
			}
		}

		switch cmd.Arg() {
		case "<", "back":
			if view == "playlist" {
				view = "results"
				continue
			}
			history.Back()
			continue
		case ">", "forward":
			history.Forward()
			continue
		}

		if !cmd.Equal(ocmd) && !cmd.IsChoice() && !cmd.IsCmd() {
			view = "results"
			r, err := ym.ExecSearch(cmd.Arg())
			if err != nil {
				panic(err)
			}
			history.Write(cmd.Arg(), r)
			ocmd = cmd
			continue
		}

		choice := cmd.Choice()
		_, cur := history.Current()
		if choice > 0 && choice <= len(cur) {
			if view != "results" {
				view = "results"
				continue
			}

			r := cur[choice-1]
			if r.IsPlayList() {
				if cur, err = r.PlaylistResults(); err != nil {
					panic(err)
				}
				history.Write("Playlist: "+r.Title(), cur)
				continue
			}

			if cmd.Cmd() == ':' {
				i, err := r.Info()
				if err != nil {
					panic(err)
				}
				infoChan <- i
				view = "info"
				continue
			}

			cacheChan <- r
			cmd.SetResult(r)
		}

		playChan <- cmd
	}
}
