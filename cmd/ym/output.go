package main

import (
	"fmt"
	"strings"

	runewidth "github.com/mattn/go-runewidth"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/search"
	"github.com/frizinak/ym/terminal"
)

func printStatus(q <-chan string, r <-chan search.Result) {
	query := "-"
	var result search.Result

	print := func() {
		w, _ := terminal.Dimensions()
		title := "-"
		if result != nil {
			title = result.Title()
		}

		if query == "" {
			query = "-"
		}

		query = fmt.Sprintf(" %s ", query)
		fmt.Printf(
			fmt.Sprintf(
				"\033[s\033[0;0f\033[K\033[1;41m%s%%%ds\033[0m\n\033[u",
				query,
				w-2-runewidth.StringWidth(query),
			),
			fmt.Sprintf(" Playing: %s ", title),
		)
	}

	for {
		select {
		case query = <-q:
			print()
		case result = <-r:
			print()
		}
	}
}

func printResults(c <-chan []search.Result) {
	for results := range c {
		_, h := terminal.Dimensions()
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

			fmt.Printf(
				"\033[%d;0f\033[K\033[1;41m %02d) \033[0m %s%s\n",
				i+2,
				i+1,
				strings.Join(labels, " "),
				results[i].Title(),
			)
		}

		for i := amount; i < h+1; i++ {
			fmt.Printf(
				"\033[s\033[%d;0f\033[K\033[u",
				i+2,
			)
		}
		fmt.Printf("> ")
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

		_, h := terminal.Dimensions()
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
				j+2,
				f,
			)
		}

		for i := len(items); i < h+1; i++ {
			fmt.Printf(
				"\033[s\033[%d;0f\033[K\033[u",
				i+2,
			)
		}
		fmt.Printf("> ")
	}
}

func prompt() ([]*command.Command, error) {
	in, err := terminal.Prompt()
	if err != nil {
		return nil, err
	}

	sp := strings.Split(in, ",")
	commands := make([]*command.Command, 0, 2*len(sp))
	for _, cmd := range sp {
		commands = append(commands, command.New(cmd))
	}

	return commands, nil
}
