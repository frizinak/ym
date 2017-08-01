package search

import (
	"net/url"
)

type Result interface {
	IsPlayList() bool
	PlaylistResults() ([]Result, error)
	DownloadURL() (*url.URL, error)
	URL() *url.URL
	Title() string
}

type Engine interface {
	Search(q string) ([]Result, error)
}
