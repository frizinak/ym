package config

import (
	"os/user"
	"path"
	"path/filepath"

	"github.com/frizinak/ym/audio"
	"github.com/frizinak/ym/player"
)

var (
	CacheDir   string
	Playlist   string
	Downloads  string
	Preflights = 10
)

func init() {
	if user, err := user.Current(); err == nil {
		CacheDir = path.Join(user.HomeDir, ".cache", "ym")
	}
	Playlist = filepath.Join(CacheDir, "playlist")
	Downloads = filepath.Join(CacheDir, "downloads")
}

func Extractor() (audio.Extractor, error) {
	return audio.FindSupportedExtractor(
		audio.NewFFMPEG(),
		audio.NewMEncoder(),
	)
}

func Player(volumeChan chan int, seekChan chan float64) (player.Player, error) {
	return player.FindSupportedPlayer(
		player.NewLibMPV(volumeChan, seekChan),
		player.NewMPlayer(),
		player.NewFFPlay(),
	)
}
