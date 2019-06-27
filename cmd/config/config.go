package config

import (
	"os/user"
	"path"
	"path/filepath"
)

var (
	CacheDir   string
	Playlist   string
	Downloads  string
	Preflights int = 10
)

func init() {
	if user, err := user.Current(); err == nil {
		CacheDir = path.Join(user.HomeDir, ".cache", "ym")
	}
	Playlist = filepath.Join(CacheDir, "playlist")
	Downloads = filepath.Join(CacheDir, "downloads")
}
