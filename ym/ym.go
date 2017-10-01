package ym

import (
	"bufio"
	"fmt"
	"net"
	"strings"

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
	quit <-chan struct{},
) error {
	var commands chan player.Command

	iq := make(chan *command.Command, 0)
	wait := make(chan func(), 0)
	go func() {
		for {
			iq <- ym.playlist.Read()
			if w := <-wait; w != nil {
				w()
			}
			ym.state = "stop"
			status <- "Stopped"
			current <- nil
			ym.current = nil
		}
	}()

	for {
		select {
		case <-quit:
			if commands != nil {
				commands <- player.CMD_STOP
			}
			return nil
		case c := <-iq:
			result := c.Result()
			if result == nil {
				continue
			}

			var file string
			// TODO
			// if c.Cmd() != '!' {
			cached := ym.cache.Get(result.ID())
			if cached != nil {
				file = cached.Path()
			}
			// }

			if file == "" {
				u, err := result.DownloadURL()
				if err != nil {
					errs <- err
					wait <- nil
					continue
				}
				file = u.String()
			}

			params := []player.Param{player.PARAM_SILENT}

			// TODO
			// if c.Cmd() != '!' {
			params = append(params, player.PARAM_NO_VIDEO)
			// }

			var err error
			var w func()

			commands, w, err = ym.player.Spawn(file, params)
			ym.state = "play"
			status <- "Playing"
			current <- result
			wait <- w
			ym.current = result
			if err != nil {
				errs <- err
				continue
			}

		case cmd := <-queue:
			var c player.Command = player.CMD_NIL
			if choice := cmd.Choice(); choice > 0 {
				ym.playlist.SetIndex(choice - 1)
				cmd = command.New([]rune{'>'})
			}

			if cmd.Next() {
				ym.playlist.Next(1)
				c = player.CMD_STOP

			} else if cmd.Prev() {
				ym.playlist.Prev(1)
				c = player.CMD_STOP

			} else if from, to := cmd.Move(); from != 0 && to != 0 {
				ym.playlist.Move(from-1, to-1)

			} else if ints := cmd.Delete(); len(ints) != 0 {
				ix := ym.playlist.Index()
				for i := range ints {
					ints[i]--
					if ix == ints[i] {
						c = player.CMD_STOP
					}
				}

				ym.playlist.Del(ints)

			} else if cmd.Clear() {
				ym.playlist.Truncate()
				c = player.CMD_STOP

			} else if cmd.Pause() {
				switch ym.state {
				case "pause":
					status <- "Playing"
					ym.state = "play"
				case "play":
					status <- "Paused"
					ym.state = "pause"
				}

				c = player.CMD_PAUSE

			} else if y := cmd.Scroll(); y != 0 {
				ym.playlist.Scroll(y)

			} else if ud := cmd.Volume(); ud != 0 {
				c = player.CMD_VOL_UP
				if ud < 0 {
					c = player.CMD_VOL_DOWN
				}

			} else if cmd.SeekBack() {
				c = player.CMD_SEEK_BACKWARD
			} else if cmd.SeekForward() {
				c = player.CMD_SEEK_FORWARD

			} else if cmd.Rand() {
				ym.playlist.ToggleRandom()
			}

			if c != player.CMD_NIL {
				if commands != nil {
					commands <- c
					if c == player.CMD_STOP {
						close(commands)
						commands = nil
					}
				}
			}
		}
	}

	return nil
}
