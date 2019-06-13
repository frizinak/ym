package search

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

var resultTypes = make(map[string]reflect.Type)

func init() {
	RegisterResultType(&YoutubeResult{})
}

func RegisterResultType(r Result) {
	resultTypes[ResultTypeName(r)] = reflect.TypeOf(r).Elem()
}

func ResultType(name string) Result {
	return reflect.New(resultTypes[name]).Interface().(Result)
}

func ResultTypeName(r Result) string {
	t := reflect.TypeOf(r).Elem()
	return t.PkgPath() + "/" + t.Name()
}

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

type Engine interface {
	Search(q string, page int) ([]Result, error)
	Page(url string) ([]Result, error)
}

type Result interface {
	ID() string

	IsPlayList() bool
	PlaylistResults(timeout time.Duration) ([]Result, error)

	DownloadURLs() (URLs, error)
	PageURL() *url.URL

	Title() string
	Info() (Info, error)
	Marshal() (string, error)
	Unmarshal(b string) error
}

type URLs []*url.URL

func (urls URLs) Find(maxURLsToTry int) (*url.URL, error) {
	for i, u := range urls {
		if i == maxURLsToTry {
			break
		}

		res, err := http.Head(u.String())
		if res != nil && res.Body != nil {
			res.Body.Close()
		}

		if err == nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			return u, nil
		}
	}

	return nil, errors.New("No suitable url found")
}
