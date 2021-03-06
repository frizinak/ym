package main

import (
	"errors"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/frizinak/ym/audio"
	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/cmd/config"
	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/history"
	"github.com/frizinak/ym/player"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
	"github.com/frizinak/ym/ym"
)

const (
	ViewPlaylist = iota
	ViewSearch
	ViewInfo
	ViewHelp
)

var version = "unknown"

func getPlaylist(cacheDir string, ch chan struct{}) (*playlist.Playlist, error) {
	var e bool
	if cacheDir != "" {
		_, err := os.Stat(config.Playlist)
		if err == nil || !os.IsNotExist(err) {
			e = true
		}
	}

	pl := playlist.New(config.Playlist, 100, ch)
	if e {
		if err := pl.Load(); err != nil {
			return pl, err
		}
	}

	return pl, nil
}

func getCache(cacheDir string, e audio.Extractor) *cache.Cache {
	dls, _ := cache.New(e, cacheDir, filepath.Join(os.TempDir(), "ym"))
	return dls
}

func main() {
	rand.Seed(time.Now().UnixNano())
	quit := make(chan struct{})
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	errChan := make(chan error)

	yt, err := search.NewYoutube(time.Second * 5)
	if err != nil {
		panic(err)
	}

	volumeChan := make(chan int)
	seekChan := make(chan *player.Pos)
	p, err := config.Player(volumeChan, seekChan)
	if err != nil {
		panic(err)
	}

	e, _ := config.Extractor()
	dls := getCache(config.Downloads, e)
	playlistChan := make(chan struct{}, 1)
	pl, err := getPlaylist(config.CacheDir, playlistChan)
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

			u, err := entry.DownloadURLs()
			if err != nil {
				continue
			}
			du, err := u.Find(config.Preflights)
			if err != nil {
				continue
			}

			err = dls.Set(cache.NewEntry(entry.ID(), "mp4", du))
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
		config.Preflights,
	)

	// ignore error
	go ym.Listen()
	titleChan := make(chan *status, 100)

	if err := initTerm(); err != nil {
		panic(err)
	}

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
	currentChan := make(chan search.Result)

	statusChan := make(chan string)
	go func() {
		err := ym.Play(
			playChan,
			currentChan,
			statusChan,
			errChan,
			quit,
		)
		if err != nil {
			closeTerm()
			panic(err)
		}
	}()

	go func() {
		for err := range errChan {
			titleChan <- &status{"Error: " + err.Error(), time.Second * 5}
		}
	}()

	go printStatus(titleChan, currentChan, volumeChan)
	go printSeeker(seekChan, statusChan)

	resultsChan := make(chan []search.Result)
	go printResults(resultsChan)

	infoChan := make(chan search.Info)
	go printInfo(infoChan)

	playlistTriggerChan := make(chan struct{})
	go printPlaylist(pl, playlistTriggerChan)

	helpTriggerChan := make(chan struct{})
	go printHelp(version, p.Name(), e.Name(), helpTriggerChan)

	view := ViewPlaylist
	go func() {
		for range playlistChan {
			if view == ViewPlaylist {
				playlistTriggerChan <- struct{}{}
			}
		}
	}()

	var info search.Result

	var lastSearch string
	var offsetSearch int

	for {
		switch view {
		case ViewPlaylist:
			titleChan <- &status{msg: "Playlist"}
			playlistTriggerChan <- struct{}{}
		case ViewSearch:
			title, r := history.Current()
			titleChan <- &status{msg: title}
			resultsChan <- r
		case ViewInfo:
			if info == nil {
				view = ViewSearch
				continue
			}

			i, err := info.Info()
			if err != nil {
				errChan <- err
				view = ViewSearch
				continue
			}

			infoChan <- i
			titleChan <- &status{msg: "Info:" + info.Title()}
		case ViewHelp:
			helpTriggerChan <- struct{}{}
			titleChan <- &status{msg: "Help"}
		}

		//if len(cmds) == 0 {
		//if cmd == nil {
		if cmd, err = prompt(cmd); err != nil {
			closeTerm()
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

		if view != ViewSearch && view != ViewPlaylist {
			view = ViewSearch
			continue
		}

		switch {
		case cmd.Playlist():
			view = ViewPlaylist
			pl.ResetScroll()
			continue
		case cmd.Back():
			if view == ViewPlaylist {
				view = ViewSearch
				continue
			}
			history.Back()
			continue
		case cmd.Forward():
			history.Forward()
			continue
		case cmd.Help():
			view = ViewHelp
			continue
		}

		if cmd.IsText() {
			qry := cmd.String()
			if qry == "" && view == ViewPlaylist {
				view = ViewSearch
				continue
			}

			view = ViewSearch
			titleChan <- &status{msg: "Searching: " + qry}
			r, err := ym.ExecSearch(qry, 60)
			if err != nil {
				errChan <- err
				continue
			}
			history.Write(qry, r)
			continue
		} else if u := cmd.URL(); u != "" {
			view = ViewSearch
			titleChan <- &status{msg: "Page: " + u}
			r, err := ym.ExecPage(u)
			if err != nil {
				errChan <- err
				continue
			}
			history.Write("Page: "+u, r)
			continue
		}

		_, cur := history.Current()

		switch view {
		case ViewSearch:
			if i := cmd.Info(); i != 0 && i <= len(cur) {
				r := cur[i-1]
				info = r
				view = ViewInfo
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
		case ViewPlaylist:
			if i := cmd.Info(); i > 0 && i <= pl.Length() {
				r := pl.At(i - 1)
				if r == nil {
					continue
				}
				info = r.Result()
				view = ViewInfo
				continue
			}

			if qry := cmd.Search(); qry != "" {
				if lastSearch != qry {
					lastSearch = qry
					offsetSearch = 0
				}

				pl.Search(qry, &offsetSearch)
				cmd = cmd.Clone()
				continue
			}

		default:
			view = ViewSearch
			continue
		}

		playChan <- cmd
	}
}
