package search

import (
	"net/url"
	"time"
)

type Format struct {
	Resolution    string
	VideoEncoding string
	AudioEncoding string
	AudioBitrate  int
}

type Info interface {
	ID() string
	PageURL() *url.URL
	Title() string
	Created() time.Time
	Formats() []*Format
	Author() string
	Duration() time.Duration
}

type Result interface {
	ID() string

	IsPlayList() bool
	PlaylistResults() ([]Result, error)

	DownloadURL() (*url.URL, error)
	PageURL() *url.URL

	Title() string
	Info() (Info, error)
}

type Engine interface {
	Search(q string) ([]Result, error)
}
