package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/cmd/config"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Specify a directory to hardlink cached items to.")
		os.Exit(1)
	}

	path := os.Args[1]
	if path == "-h" || path == "--help" {
		fmt.Println("Hardlink all cached items to the provided directory")
		os.Exit(0)
	}

	fn := regexp.MustCompile("[\\/:*?\"<>|\\0]+")
	e, _ := config.Extractor()
	dls, err := cache.New(e, config.Downloads, filepath.Join(os.TempDir(), "ym"))
	if err != nil {
		panic(err)
	}

	os.MkdirAll(path, 0755)
	pl := playlist.New(config.Playlist, 100, nil)
	if err := pl.Load(); err != nil {
		panic(err)
	}

	clean := func(p string) string {
		return strings.Trim(fn.ReplaceAllString(p, "-"), "-")
	}

	workers := 1
	work := make(chan search.Result, workers)
	var wg sync.WaitGroup

	have := 0
	done := make(chan struct{}, workers)
	fin := make(chan struct{})
	go func() {
		for range done {
			have++
			fmt.Printf("\033[20D\033[K%d/%d", have, pl.Length())
		}
		fin <- struct{}{}
	}()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for r := range work {
				c := dls.Get(r.ID())
				if c == nil {
					continue
				}
				hardlink := c.Path()
				symlink := filepath.Join(path, clean(r.Title())+filepath.Ext(hardlink))
				if err := os.Link(hardlink, symlink); err != nil && !os.IsExist(err) {
					panic(err)
				}
				done <- struct{}{}
			}
			wg.Done()
		}()
	}

	for _, e := range pl.List() {
		work <- e.Result()
	}
	close(work)
	wg.Wait()
	close(done)
	<-fin
	fmt.Println("\ndone")

}
