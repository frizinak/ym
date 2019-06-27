package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/frizinak/ym/audio"
	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/cmd/config"
	"github.com/frizinak/ym/playlist"
	"github.com/frizinak/ym/search"
)

func handle(workerIndex int, r search.Result, dls *cache.Cache) error {
	c := dls.Get(r.ID())
	if c != nil {
		return nil
	}

	us, err := r.DownloadURLs()
	if err != nil {
		return err
	}
	u, err := us.Find(config.Preflights)
	if err != nil {
		return err
	}

	last := time.Time{}
	progress := func(written, total int64) {
		if written == total {
			fmt.Printf("\033[%d;0f\033[K\n", workerIndex+2)
			return
		}

		if time.Since(last).Seconds() < 3 {
			return
		}
		last = time.Now()

		if total == 0 {
			return
		}

		w, t := float64(written), float64(total)

		fmt.Printf(
			"\033[%d;0f\033[K %6.2fMB / %6.2fMB %5.1f%% %s\n",
			workerIndex+2,
			w/(1024*1024),
			t/(1024*1024),
			w*100/t,
			r.Title(),
		)
	}

	return dls.SetProgress(cache.NewEntry(r.ID(), "mp4", u), progress)
}

func main() {
	e, _ := audio.FindSupportedExtractor(
		audio.NewFFMPEG(),
		audio.NewMEncoder(),
	)

	dls, _ := cache.New(e, config.Downloads, filepath.Join(os.TempDir(), "ym"))
	pl := playlist.New(config.Playlist, 100, nil)
	if err := pl.Load(); err != nil {
		panic(err)
	}

	workers := runtime.NumCPU()
	if len(os.Args) == 2 {
		w, _ := strconv.Atoi(os.Args[1])
		if w > 0 {
			workers = w
		}
	}

	work := make(chan search.Result, workers)
	var wg sync.WaitGroup

	list := pl.List()
	have := 0
	total := len(list)
	done := make(chan struct{}, 1)
	go func() {
		for range done {
			have++
			fmt.Printf(
				"\033[%d;0f\033[K %04d/%04d\n",
				workers+3,
				have,
				total,
			)
		}
	}()

	fmt.Printf("\033[2;J")
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			for r := range work {
				if err := handle(i, r, dls); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"\033[30;41m ERR: %s \n %s \n %s \033[0m\n",
						err,
						r.Title(),
						dls.Base(r.ID()),
					)
				}

				done <- struct{}{}
			}
			wg.Done()
		}(i)
	}

	for _, e := range list {
		work <- e.Result()
	}
	close(work)
	wg.Wait()
	close(done)
	fmt.Fprintln(os.Stdout, "done")
}
