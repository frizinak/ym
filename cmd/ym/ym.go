package main

import (
	"errors"
	"net"
	"os"
	"os/signal"
	"os/user"
	"path"
	"time"

	"github.com/frizinak/ym/audio"
	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/history"
	"github.com/frizinak/ym/player"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
	"github.com/frizinak/ym/ym"
)

const (
	VIEW_PLAYLIST = iota
	VIEW_SEARCH
	VIEW_INFO
)

func getPlaylist(cacheDir string, ch chan struct{}) (*playlist.Playlist, error) {
	var f string
	var e bool
	if cacheDir != "" {
		f = path.Join(cacheDir, "playlist")
		if _, err := os.Stat(f); err == nil || !os.IsNotExist(err) {
			e = true
		}
	}

	pl := playlist.New(f, 100, ch)
	if e {
		if err := pl.Load(); err != nil {
			return pl, err
		}
	}

	return pl, nil
}

func getCache(cacheDir string, e audio.Extractor) *cache.Cache {
	dls, _ := cache.New(e, cacheDir, path.Join(os.TempDir(), "ym"))
	return dls
}

func main() {
	if err := initTerm(); err != nil {
		panic(err)
	}

	quit := make(chan struct{}, 0)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

	errChan := make(chan error, 0)

	yt, err := search.NewYoutube(time.Second * 5)
	if err != nil {
		panic(err)
	}

	p, err := player.FindSupportedPlayer(
		player.NewLibMPV(),
		player.NewMPlayer(),
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
		audio.NewMEncoder(),
	)

	dls := getCache(path.Join(cacheDir, "downloads"), e)
	playlistChan := make(chan struct{}, 1)
	pl, err := getPlaylist(cacheDir, playlistChan)
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

	cacheChan := make(chan search.Result, 2000)
	go func() {
		for entry := range cacheChan {
			if entry == nil || dls.Get(entry.ID()) != nil {
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

	go func() {
		for _, cmd := range pl.List() {
			cacheChan <- cmd.Result()
		}
	}()

	if err != nil {
		go func() {
			errChan <- errors.New("Could not load playlist " + err.Error())
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
	titleChan := make(chan *status, 100)
	go func() {
		for range signals {
			titleChan <- &status{msg: "Saving playlist and quitting"}
			quit <- struct{}{}
			pl.Save(true)
			closeTerm()
			os.Exit(0)
		}
	}()

	var cmd *command.Command
	history := history.New(20)

	playChan := make(chan *command.Command, 100)
	currentChan := make(chan search.Result, 0)

	statusChan := make(chan string, 0)
	go func() {
		if err := ym.Play(playChan, currentChan, statusChan, errChan, quit); err != nil {
			panic(err)
		}
	}()

	go func() {
		for err := range errChan {
			titleChan <- &status{err.Error(), time.Second * 5}
		}
	}()

	go printStatus(titleChan, currentChan, statusChan)

	resultsChan := make(chan []search.Result, 0)
	go printResults(resultsChan)

	infoChan := make(chan search.Info, 0)
	go printInfo(infoChan)

	playlistTriggerChan := make(chan struct{}, 0)
	go printPlaylist(pl, playlistTriggerChan)

	view := VIEW_PLAYLIST
	go func() {
		for {
			select {
			case <-playlistChan:
				if view == VIEW_PLAYLIST {
					playlistTriggerChan <- struct{}{}
				}
			}
		}
	}()

	var info search.Result

	for {
		switch view {
		case VIEW_PLAYLIST:
			titleChan <- &status{msg: "Playlist"}
			playlistTriggerChan <- struct{}{}
		case VIEW_SEARCH:
			title, r := history.Current()
			titleChan <- &status{msg: title}
			resultsChan <- r
		case VIEW_INFO:
			if info == nil {
				view = VIEW_SEARCH
				continue
			}

			i, err := info.Info()
			if err != nil {
				errChan <- err
				view = VIEW_SEARCH
				continue
			}

			infoChan <- i
			titleChan <- &status{msg: "Info:" + info.Title()}
		}

		//if len(cmds) == 0 {
		//if cmd == nil {
		if cmd, err = prompt(cmd); err != nil {
			panic(err)
		}
		//}

		if cmd == nil || !cmd.Done() {
			continue
		}

		if cmd.Exit() {
			signals <- os.Interrupt
			continue
		}

		//cmd = cmds[0]
		//cmds = cmds[1:]

		if view != VIEW_SEARCH && view != VIEW_PLAYLIST {
			view = VIEW_SEARCH
			continue
		}

		switch {
		case cmd.Playlist():
			view = VIEW_PLAYLIST
			pl.ResetScroll()
			continue
		case cmd.Back():
			if view == VIEW_PLAYLIST {
				view = VIEW_SEARCH
				continue
			}
			history.Back()
			continue
		case cmd.Forward():
			history.Forward()
			continue
		}

		if cmd.IsText() {
			qry := cmd.String()
			if qry == "" && view == VIEW_PLAYLIST {
				view = VIEW_SEARCH
				continue
			}

			view = VIEW_SEARCH
			titleChan <- &status{msg: "Searching: " + qry}
			r, err := ym.ExecSearch(qry)
			if err != nil {
				errChan <- err
				continue
			}
			history.Write(qry, r)
			continue
		}

		_, cur := history.Current()

		switch view {
		case VIEW_SEARCH:
			if i := cmd.Info(); i != 0 && i <= len(cur) {
				r := cur[i-1]
				info = r
				view = VIEW_INFO
				continue
			}

			choices := cmd.Choices()
			if len(choices) == 0 {
				break
			}

			for _, choice := range choices {
				r := cur[choice-1]
				if r.IsPlayList() {
					if len(choices) == 1 {
						if cur, err = r.PlaylistResults(time.Second * 5); err != nil {
							errChan <- err
							continue
						}
						history.Write("Playlist: "+r.Title(), cur)
					}
					continue
				}

				cacheChan <- r
				pl.Add(cmd.Clone().SetResult(r))
			}
			continue
		case VIEW_PLAYLIST:
			if i := cmd.Info(); i > 0 && i <= pl.Length() {
				r := pl.At(i - 1)
				if r == nil {
					continue
				}
				info = r.Result()
				view = VIEW_INFO
				continue
			}
		default:
			view = VIEW_SEARCH
			continue
		}

		playChan <- cmd
	}
}
