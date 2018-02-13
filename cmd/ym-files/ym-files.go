package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/frizinak/ym/audio"
	"github.com/frizinak/ym/cache"
	"github.com/frizinak/ym/cmd/config"
	"github.com/frizinak/ym/playlist"
)

func main() {
	download := true
	fn := regexp.MustCompile("[\\/:*?\"<>|\\0]+")
	e, _ := audio.FindSupportedExtractor(
		audio.NewFFMPEG(),
		audio.NewMEncoder(),
	)

	dls, _ := cache.New(e, config.Downloads, filepath.Join(os.TempDir(), "ym"))

	dest := "ym-files"
	os.MkdirAll(dest, 0755)

	pl := playlist.New(config.Playlist, 100, nil)
	if err := pl.Load(); err != nil {
		panic(err)
	}

	clean := func(p string) string {
		return strings.Trim(fn.ReplaceAllString(p, "-"), "-")
	}

	for _, e := range pl.List() {
		r := e.Result()
		c := dls.Get(r.ID())
		if c == nil {
			if !download {
				continue
			}

			u, err := r.DownloadURL()
			if err != nil {
				log.Println("No download url for", r.Title())
				continue
			}

			if err := dls.Set(cache.NewEntry(r.ID(), "mp4", u)); err != nil {
				log.Println(err)
				continue
			}

		}

		c = dls.Get(r.ID())
		hardlink := c.Path()
		ext := filepath.Ext(hardlink)

		symlink := filepath.Join(dest, clean(r.Title())+ext)
		if err := os.Link(hardlink, symlink); err != nil && !os.IsExist(err) {
			panic(err)
		}
	}

}
