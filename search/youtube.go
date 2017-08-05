package search

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rylio/ytdl"
)

type YoutubeInfo struct {
	i       *ytdl.VideoInfo
	formats []*Format
	url     *url.URL
}

func (y *YoutubeInfo) ID() string              { return y.i.ID }
func (y *YoutubeInfo) PageURL() *url.URL       { return y.url }
func (y *YoutubeInfo) Title() string           { return y.i.Title }
func (y *YoutubeInfo) Created() time.Time      { return y.i.DatePublished }
func (y *YoutubeInfo) Formats() []*Format      { return y.formats }
func (y *YoutubeInfo) Author() string          { return y.i.Author }
func (y *YoutubeInfo) Duration() time.Duration { return y.i.Duration }

type YoutubeResult struct {
	id    string
	re    *regexp.Regexp
	url   *url.URL
	title string
	info  *YoutubeInfo
}

func (y *YoutubeResult) ID() string        { return y.id }
func (y *YoutubeResult) Title() string     { return y.title }
func (y *YoutubeResult) PageURL() *url.URL { return y.url }
func (y *YoutubeResult) IsPlayList() bool  { return y.url.Query().Get("list") != "" }

func (y *YoutubeResult) DownloadURL() (*url.URL, error) {
	if err := y.getInfo(); err != nil {
		return nil, err
	}

	bestAudio := y.info.i.Formats.Best(ytdl.FormatAudioBitrateKey)
	if len(bestAudio) == 0 {
		return nil, fmt.Errorf("No downloadable formats available")
	}

	u, err := y.info.i.GetDownloadURL(bestAudio[0])
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (y *YoutubeResult) PlaylistResults() ([]Result, error) {
	results, err := match(
		&url.URL{
			Scheme:   y.url.Scheme,
			Host:     y.url.Host,
			Path:     "playlist",
			RawQuery: "list=" + y.url.Query().Get("list"),
		},
		y.re,
		true,
	)

	// Remove 'Play all' button
	if len(results) != 0 {
		results = results[1:]
	}

	return results, err
}

func (y *YoutubeResult) Info() (Info, error) {
	if err := y.getInfo(); err != nil {
		return nil, err
	}

	return y.info, nil
}

func (y *YoutubeResult) getInfo() error {
	if y.info != nil {
		return nil
	}

	vid, err := ytdl.GetVideoInfo(y.url)
	if err != nil {
		return err
	}

	filtered := vid.Formats.Filter(
		ytdl.FormatAudioBitrateKey,
		[]interface{}{128, 192, 256, 400},
	)

	filtered.Sort(ytdl.FormatAudioBitrateKey, false)
	formats := make([]*Format, 0, len(filtered))
	for _, f := range filtered {
		formats = append(
			formats,
			&Format{
				f.Resolution,
				f.VideoEncoding,
				f.AudioEncoding,
				f.AudioBitrate,
			},
		)
	}

	y.info = &YoutubeInfo{vid, formats, y.url}

	return nil
}

func (y *YoutubeResult) Marshal() []byte {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	enc.Encode([]string{y.id, y.url.String(), y.title})

	return b.Bytes()
}

func (y *YoutubeResult) Unmarshal(b []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(b))
	var d []string
	if err := dec.Decode(&d); err != nil {
		return err
	}

	y.id = d[0]
	y.url, _ = url.Parse(d[1])
	y.title = d[2]
	y.info = nil

	return nil
}

type Youtube struct {
	re *regexp.Regexp
}

func NewYoutube() (*Youtube, error) {
	re, err := regexp.Compile(`<a.*href="(/watch[^"]+)"[^>]*?(?: title="([^"]+)"|[^>]*?>([^<>]+)</a)`)
	if err != nil {
		return nil, err
	}

	return &Youtube{re: re}, nil
}

func (y *Youtube) Search(q string) ([]Result, error) {
	u, err := url.Parse(
		fmt.Sprintf(
			"https://www.youtube.com/results?search_query=%s",
			url.QueryEscape(q),
		),
	)
	if err != nil {
		return nil, err
	}

	return match(u, y.re, false)
}

func match(u *url.URL, re *regexp.Regexp, trimPlaylist bool) ([]Result, error) {
	res, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	matches := re.FindAllSubmatch(body, -1)
	results := make([]Result, 0, len(matches))

	for i := range matches {
		p, err := url.Parse(html.UnescapeString(string(matches[i][1])))
		if err != nil {
			continue
		}

		p.Host = u.Host
		p.Scheme = u.Scheme
		if trimPlaylist {
			q := p.Query()
			q.Del("list")
			q.Del("index")
			p.RawQuery = q.Encode()
		}

		results = append(
			results,
			&YoutubeResult{
				p.Query().Get("v"),
				re,
				p,
				strings.TrimSpace(html.UnescapeString(string(matches[i][3]))),
				nil,
			},
		)
	}

	return results, nil
}
