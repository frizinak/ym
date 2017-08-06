package main

import (
	"errors"
	"net"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/frizinak/ym/audio"
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
			return pl, err
		}
	}

	return pl, nil
}

func getCache(cacheDir string, e audio.Extractor) *cache.Cache {
	if cacheDir != "" {
		dls, _ := cache.New(e, cacheDir, path.Join(os.TempDir(), "ym"))
		return dls
	}

	return nil
}

func main() {
	errChan := make(chan error, 0)

	yt, err := search.NewYoutube(time.Second * 5)
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

	e, _ := audio.FindSupportedExtractor(
		audio.NewFFMPEG(),
	)

	dls := getCache(path.Join(cacheDir, "downloads"), e)
	pl, err := getPlaylist(cacheDir)
	if pl == nil {
		panic(err)
	}

	go func() {
		for {
			time.Sleep(time.Second * 5)
			if err := pl.Save(true); err != nil {
				panic(err)
			}
		}
	}()

	if err != nil {
		go func() {
			errChan <- errors.New("Could not load playlist")
		}()
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
		if err := ym.Play(playChan, currentChan, statusChan, errChan); err != nil {
			panic(err)
		}
	}()

	terminal.Clear()

	titleChan := make(chan *status, 0)
	go func() {
		for err := range errChan {
			titleChan <- &status{err.Error(), time.Second * 5}
		}
	}()

	statusCurrentChan := make(chan search.Result, 0)
	go printStatus(titleChan, statusCurrentChan, statusChan)

	resultsChan := make(chan []search.Result, 0)
	go printResults(resultsChan)

	infoChan := make(chan search.Info, 0)
	go printInfo(infoChan)

	playlistChan := make(chan struct{}, 0)
	go printPlaylist(pl, playlistChan)

	view := "playlist"
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

			err = dls.Set(cache.NewEntry(entry.ID(), "mp4", u))
			if err != nil {
				errChan <- err
			}
		}
	}()

	var info search.Result
	for {
		switch view {
		case "playlist":
			playlistChan <- struct{}{}
		case "results":
			title, r := history.Current()
			titleChan <- &status{msg: title}
			resultsChan <- r
		case "info":
			if info == nil {
				view = "results"
				continue
			}

			i, err := info.Info()
			if err != nil {
				errChan <- err
				view = "results"
				continue
			}

			infoChan <- i
			titleChan <- &status{msg: "Info:" + info.Title()}
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
			switch cmd.Arg(0) {
			case "list", "queue", "playlist":
				view = "playlist"
				titleChan <- &status{msg: "Playlist"}
				continue
			}
		}

		switch cmd.Arg(0) {
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
			titleChan <- &status{msg: "Searching: " + cmd.ArgStr()}
			r, err := ym.ExecSearch(cmd.ArgStr())
			if err != nil {
				errChan <- err
				continue
			}
			history.Write(cmd.ArgStr(), r)
			ocmd = cmd
			continue
		}

		_, cur := history.Current()
		switch view {
		case "results":
			if cmd.Choice() > 0 && cmd.Choice() <= len(cur) {
				r := cur[cmd.Choice()-1]
				if r.IsPlayList() {
					if cur, err = r.PlaylistResults(time.Second * 5); err != nil {
						errChan <- err
						continue
					}
					history.Write("Playlist: "+r.Title(), cur)
					continue
				}

				if cmd.Cmd() == ':' {
					info = r
					view = "info"
					continue
				}

				cacheChan <- r
				cmd.SetResult(r)
				pl.Add(cmd)
				continue
			}
		case "playlist":
			if cmd.Cmd() == ':' {
				switch {
				case cmd.Arg(0) != "" && strings.HasPrefix("delete", cmd.Arg(0)):
					which, err := strconv.Atoi(cmd.Arg(1))
					if err != nil {
						break
					}
					which--

					ix := pl.Index()
					pl.Del(which)
					if ix != which {
						continue
					}
					cmd = command.New(":next")
				case cmd.Choice() > 0 && cmd.Choice() <= pl.Length():
					r := pl.At(cmd.Choice() - 1)
					if r == nil {
						continue
					}
					info = r.Result()
					view = "info"
					continue
				}
			}
		default:
			view = "results"
			continue
		}

		playChan <- cmd
	}
}
