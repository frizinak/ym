package search

import (
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

type Result interface {
	ID() string

	IsPlayList() bool
	PlaylistResults(timeout time.Duration) ([]Result, error)

	DownloadURL() (*url.URL, error)
	PageURL() *url.URL

	Title() string
	Info() (Info, error)
	Marshal() []byte
	Unmarshal(b []byte) error
}

type Engine interface {
	Search(q string) ([]Result, error)
}
