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

		fmt.Printf("\033[2;0f\033[K") //, title)
		for i := 0; i < amount; i++ {
			fmt.Printf(
				"\033[%d;0f\033[K\033[30;42m %02d) \033[0m %s\n",
				i+2,
				i+1,
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
