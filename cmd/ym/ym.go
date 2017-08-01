package main

import (
	"os"
	"strings"
	"sync"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/history"
	"github.com/frizinak/ym/player"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
	"github.com/frizinak/ym/terminal"
)

type YM struct {
	search search.Engine
	player player.Player
}

func (ym *YM) execSearch(q string) ([]search.Result, error) {
	results, err := ym.search.Search(q)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (ym *YM) play(
	playlist *playlist.Playlist,
	queue <-chan *command.Command,
	current chan<- search.Result,
) error {
	var commands chan player.Command
	var sem sync.Mutex
	errs := make(chan error, 0)

	go func() {

		for {
			c := playlist.Pop()
			result := c.Result()
			if result == nil {
				continue
			}

			u, err := result.DownloadURL()
			if err != nil {
				errs <- err
				break
			}

			params := []player.Param{player.PARAM_SILENT}
			if c.Cmd() == '@' {
				params = append(params, player.PARAM_ATTACH)
			}

			if c.Cmd() != '!' {
				params = append(params, player.PARAM_NO_VIDEO)
			}

			var wait func()

			sem.Lock()
			commands, wait, err = ym.player.Spawn(u.String(), params)
			sem.Unlock()
			current <- result
			if err != nil {
				errs <- err
				break
			}

			wait()
		}
	}()

	for {
		select {
		case err := <-errs:
			return err
		case cmd, ok := <-queue:
			if !ok {
				return nil
			}

			if cmd.Cmd() == ':' {
				stop := false
				switch {
				case strings.HasPrefix("next", cmd.Arg()):
					stop = true
				case strings.HasPrefix("clear", cmd.Arg()):
					stop = true
					playlist.Truncate()
				case strings.HasPrefix("pause", cmd.Arg()):
					commands <- player.CMD_PAUSE
					continue
				}

				if stop {
					sem.Lock()
					if commands != nil {
						current <- nil
						commands <- player.CMD_STOP
					}
					sem.Unlock()
				}
				continue
			}

			playlist.Add(cmd)
		}
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

	ym := &YM{yt, p}

	var cmd *command.Command
	var ocmd *command.Command
	cmds := make([]*command.Command, 0, 1)
	if len(os.Args) > 1 {
		cmds = append(cmds, command.New(strings.Join(os.Args[1:], " ")))
	}

	results := history.New(20)
	playlist := playlist.New(100)

	playChan := make(chan *command.Command, 2000)
	currentChan := make(chan search.Result, 0)
	go func() {
		if err := ym.play(playlist, playChan, currentChan); err != nil {
			panic(err)
		}
	}()

	statusChan := make(chan string, 0)
	resultsChan := make(chan []search.Result, 0)
	terminal.Clear()
	go printStatus(statusChan, currentChan)
	go printResults(resultsChan)

	for {
		title, r := results.Current()
		statusChan <- title
		resultsChan <- r
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

		if cmd.Cmd() == ':' {
			switch cmd.Arg() {
			case "list", "queue":
				results.Write("Queued:", playlist.ResultList())
				continue
			}
		}

		switch cmd.Arg() {
		case "<", "back":
			results.Back()
			continue
		case ">", "forward":
			results.Forward()
			continue
		}

		if !cmd.Equal(ocmd) && !cmd.IsChoice() && !cmd.IsCmd() {
			r, err := ym.execSearch(cmd.Arg())
			if err != nil {
				panic(err)
			}
			results.Write(cmd.Arg(), r)
			ocmd = cmd
			continue
		}

		choice := cmd.Choice()
		_, cur := results.Current()
		if choice > 0 && choice <= len(cur) {
			r := cur[choice-1]
			if r.IsPlayList() {
				if cur, err = r.PlaylistResults(); err != nil {
					panic(err)
				}
				results.Write("Playlist: "+r.Title(), cur)
				continue
			}

			cmd.SetResult(r)
		}

		playChan <- cmd
	}
}
