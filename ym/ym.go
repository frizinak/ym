package ym

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/player"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
)

type YM struct {
	playlist *playlist.Playlist
	search   search.Engine
	player   player.Player
	cache    *cache.Cache

	state   string
	current search.Result
	addr    *net.TCPAddr
}

func New(
	playlist *playlist.Playlist,
	search search.Engine,
	player player.Player,
	cache *cache.Cache,
	sock *net.TCPAddr,
) *YM {
	return &YM{
		playlist,
		search,
		player,
		cache,
		"stop",
		nil,
		sock,
	}
}

func (ym *YM) ExecSearch(q string) ([]search.Result, error) {
	results, err := ym.search.Search(q)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (ym *YM) Listen() error {
	l, err := net.ListenTCP("tcp", ym.addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func(conn net.Conn) {
			r := bufio.NewReader(conn)
			for {
				line, _, err := r.ReadLine()
				if err != nil {
					break
				}

				var msg []string
				switch string(line) {
				case "close":
					conn.Close()
					return
				case "status":
					msg = []string{
						"volume: -1",
						"repeat: 0",
						"random: 0",
						"single: 0",
						"consume: 0",
						"playlist: 1",
						fmt.Sprintf("playlistlength: %d", ym.playlist.Length()),
						"mixrampdb: 0.000000",
						fmt.Sprintf("state: %s", ym.state),
					}

				case "currentsong":
					cur := ym.current
					if cur == nil {
						break
					}
					info, err := cur.Info()
					if err != nil {
						break
					}

					msg = []string{
						fmt.Sprintf("file: %s", cur.PageURL().String()),
						fmt.Sprintf("Last-Modified: %s", info.Created().String()),
						"Artist: -",
						fmt.Sprintf("Title: %s", info.Title()),
						"Track: 1",
						fmt.Sprintf("Date: %d", info.Created().Year()),
						"Genre: -",
						"Composer: -",
						fmt.Sprintf("Time: %d", int(info.Duration().Seconds())),
						fmt.Sprintf("duration: %0.3f", info.Duration().Seconds()),
						"Pos: 1",
						"Id: -",
					}
				}

				if len(msg) != 0 {
					fmt.Fprint(conn, strings.Join(msg, "\n"))
				}
			}

		}(conn)
	}
}

func (ym *YM) Play(
	queue <-chan *command.Command,
	current chan<- search.Result,
	status chan<- string,
	errs chan<- error,
) error {
	var commands chan player.Command
	var sem sync.Mutex

	go func() {
		for {
			c := ym.playlist.Read()
			result := c.Result()
			if result == nil {
				continue
			}

			var file string
			if c.Cmd() != '!' {
				cached := ym.cache.Get(result.ID())
				if cached != nil {
					file = cached.Path()
				}
			}

			if file == "" {
				u, err := result.DownloadURL()
				if err != nil {
					errs <- err
					continue
				}
				file = u.String()
			}

			params := []player.Param{player.PARAM_SILENT}

			if c.Cmd() != '!' {
				params = append(params, player.PARAM_NO_VIDEO)
			}

			var wait func()
			var err error

			sem.Lock()
			commands, wait, err = ym.player.Spawn(file, params)
			sem.Unlock()
			ym.state = "play"
			status <- "Playing"
			current <- result
			ym.current = result
			if err != nil {
				errs <- err
				continue
			}

			if wait != nil {
				wait()
			}

			ym.state = "stop"
			status <- "Stopped"
			current <- nil
			ym.current = nil
		}
	}()

	for cmd := range queue {
		choice := cmd.Choice()
		if choice > 0 {
			ym.playlist.SetIndex(choice - 1)
			cmd = command.New(":next")
		}

		if cmd.Cmd() == ':' {
			arg := cmd.Arg(0)
			if arg == "" {
				continue
			}
			var c player.Command = player.CMD_NIL
			switch {
			case strings.HasPrefix("next", arg):
				c = player.CMD_STOP
			case strings.HasPrefix("previous", arg):
				c = player.CMD_STOP
				ym.playlist.Prev()
			case strings.HasPrefix("clear", arg):
				c = player.CMD_STOP
				ym.playlist.Truncate()
			case strings.HasPrefix("pause", arg):
				c = player.CMD_PAUSE
				switch ym.state {
				case "pause":
					status <- "Playing"
					ym.state = "play"
				case "play":
					status <- "Paused"
					ym.state = "pause"
				}
			}

			if c != player.CMD_NIL {
				sem.Lock()
				if commands != nil {
					commands <- c
					if c == player.CMD_STOP {
						commands = nil
					}
				}
				sem.Unlock()
			}
		}
	}

	return nil
}
