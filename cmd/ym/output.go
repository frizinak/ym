package main

import (
	"fmt"
	"strings"
	"time"

	runewidth "github.com/mattn/go-runewidth"
	termbox "github.com/nsf/termbox-go"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
)

type status struct {
	msg     string
	timeout time.Duration
}

func printStatus(q <-chan *status, r <-chan search.Result, s <-chan string) {
	var lstatus string
	lstatusChan := make(chan string, 0)
	rstatus := "-"
	var result search.Result

	print := func() {
		w, _ := termSize()
		title := "-"
		if result != nil {
			title = result.Title()
		}

		if lstatus == "" {
			lstatus = "-"
		}

		left := fmt.Sprintf(" %s ", strings.TrimSpace(lstatus))
		right := fmt.Sprintf(" %s: %s ", rstatus, title)
		lw := runewidth.StringWidth(left)
		rw := runewidth.StringWidth(right)
		diff := lw + rw - w + 10
		if diff > 0 {
			title = runewidth.Truncate(
				title,
				runewidth.StringWidth(title)-diff-1,
				"…",
			)
			right = fmt.Sprintf(" %s: %s ", rstatus, title)
		}

		fmt.Printf(
			fmt.Sprintf(
				"\033[0;0f\033[K\033[1;41m%s%%%ds\033[0m\n\033[u",
				left,
				w-lw,
			),
			right,
		)
	}

	go func() {
		var lstatus *status
		for s := range q {
			lstatusChan <- s.msg
			if s.timeout != 0 {
				time.Sleep(s.timeout)
				if lstatus != nil {
					lstatusChan <- lstatus.msg
				}
				continue
			}
			lstatus = s
		}
	}()

	for {
		select {
		case lstatus = <-lstatusChan:
			print()
		case result = <-r:
			print()
		case rstatus = <-s:
			print()
		}
	}
}

func printResults(c <-chan []search.Result) {
	for results := range c {
		w, h := termSize()
		h -= 3
		amount := len(results)
		if h < amount {
			amount = h
		}

		fmt.Printf("\033[2;0f\033[K")
		for i := 0; i < amount; i++ {
			labels := []string{}
			if results[i].IsPlayList() {
				labels = append(labels, "\033[30;42m list \033[0m ")
			}

			lbls := strings.Join(labels, " ")
			fmt.Printf(
				"\033[%d;0f\033[K\033[1;41m %02d \033[0m %s%s\n",
				i+2,
				i+1,
				strings.Join(labels, " "),
				runewidth.Truncate(
					results[i].Title(),
					w-runewidth.StringWidth(lbls)-5,
					"…",
				),
			)
		}

		clearAndPrompt(amount, h+1)
	}
}

func printPlaylist(pl *playlist.Playlist, c <-chan struct{}) {
	for range c {
		w, h := termSize()
		h -= 3
		offset, ix, results := pl.Surrounding(h)

		fmt.Printf("\033[2;0f\033[K")
		for i := range results {
			title := runewidth.Truncate(
				results[i].Title(),
				w-1,
				"…",
			)
			if ix == i {
				title = fmt.Sprintf("\033[30;42m%s\033[0m", title)
			}

			fmt.Printf(
				"\033[%d;0f\033[K\033[1;41m %02d \033[0m %s\n",
				i+2,
				offset+i+1,
				title,
			)
		}

		clearAndPrompt(len(results), h+1)
	}
}

func printInfo(c <-chan search.Info) {
	for i := range c {
		formats := i.Formats()
		items := make([]string, 5+len(formats))

		dur := i.Duration()
		min := int(dur.Minutes())
		sec := int(dur.Seconds())
		if min > 0 {
			sec = sec % (60 * min)
		}

		items[0] = i.PageURL().String()
		items[2] = fmt.Sprintf("%s [%02d:%02d]", i.Title(), min, sec)

		items[3] = fmt.Sprintf(
			"By: %s At: %s",
			i.Author(),
			i.Created(),
		)

		_, h := termSize()
		h -= 3
		for j, f := range formats {
			items[j+5] = fmt.Sprintf(
				"%s: %s | %s: %d kbps",
				f.VideoEncoding,
				f.Resolution,
				f.AudioEncoding,
				f.AudioBitrate,
			)
		}

		fmt.Printf("\033[2;0f\033[K")
		for j, f := range items {
			fmt.Printf(
				"\033[%d;0f\033[K %s\n",
				j+3,
				f,
			)
		}

		clearAndPrompt(len(items), h+1)
	}
}

func clearAndPrompt(from, til int) {
	for i := from; i < til-1; i++ {
		fmt.Printf(
			"\033[%d;0f\033[K",
			i+2,
		)
	}
	fmt.Print("\033[u")
}

func initTerm() error {
	if err := termbox.Init(); err != nil {
		return err
	}

	_, y := termbox.Size()
	termbox.SetCursor(2, y-2)
	return termbox.Flush()
}

func termSize() (int, int) {
	w, h := termbox.Size()
	if w < 10 {
		w = 10
	}

	if h < 5 {
		h = 5
	}

	return w, h
}

func closeTerm() {
	termbox.Close()
}

func prompt(c *command.Command) (*command.Command, error) {
	if c == nil || c.Done() {
		c = command.New(make([]rune, 0, 1))
	}

	_, height := termbox.Size()
	print := func(str string) {
		termbox.SetCursor(2+runewidth.StringWidth(str), height-2)
		fmt.Printf("\033[%d;0f> \033[K%s\033[s", height-1, str)
		termbox.Flush()
	}

	print(c.String())
	for {
		e := termbox.PollEvent()
		switch e.Type {
		case termbox.EventInterrupt:
			return command.New([]rune(":exit")), nil
		case termbox.EventKey:
			switch e.Key {
			case termbox.KeyCtrlQ, termbox.KeyCtrlC:
				return command.New([]rune(":exit")).SetDone(), nil

			case termbox.KeyCtrlU:
				return command.New([]rune(fmt.Sprintf(":scroll %d", -height/2))).SetDone(), nil

			case termbox.KeyCtrlD:
				return command.New([]rune(fmt.Sprintf(":scroll %d", height/2))).SetDone(), nil

			case termbox.KeyBackspace, termbox.KeyBackspace2:
				c.Pop()

			case termbox.KeyEnter:
				e.Ch = 0
				c.SetDone()

			case termbox.KeySpace:
				e.Ch = ' '

			case termbox.KeyArrowLeft:
				if len(c.Buffer()) == 0 {
					e.Ch = '<'
				}

			case termbox.KeyArrowRight:
				if len(c.Buffer()) == 0 {
					e.Ch = '>'
				}

			case termbox.KeyArrowUp:
				return command.New([]rune(":vol +1")).SetDone(), nil

			case termbox.KeyArrowDown:
				return command.New([]rune(":vol -1")).SetDone(), nil
			}

			if e.Ch != 0 {
				c.Append(e.Ch)
			}

			if c.Done() {
				print("")
				return c, nil
			}
			print(c.String())
		case termbox.EventResize:
			_, height = termbox.Size()
			print(c.String())
			return c, nil
		}
	}
}
