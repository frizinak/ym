package search

import (
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/rylio/ytdl"
)

type YoutubeResult struct {
	re    *regexp.Regexp
	url   *url.URL
	title string
}

func (y *YoutubeResult) DownloadURL() (*url.URL, error) {
	vid, err := ytdl.GetVideoInfo(y.url)
	y.title = vid.Title
	if err != nil {
		return nil, err
	}

	bestAudio := vid.Formats.Best(ytdl.FormatAudioBitrateKey)
	if len(bestAudio) == 0 {
		return nil, fmt.Errorf("No downloadable formats available")
	}

	return vid.GetDownloadURL(bestAudio[0])
}

func (y *YoutubeResult) Title() string {
	return y.title
}

func (y *YoutubeResult) URL() *url.URL {
	return y.url
}

func (y *YoutubeResult) IsPlayList() bool {
	return y.url.Query().Get("list") != ""
}

func (y *YoutubeResult) PlaylistResults() ([]Result, error) {
	return match(
		&url.URL{
			Scheme:   y.url.Scheme,
			Host:     y.url.Host,
			Path:     "playlist",
			RawQuery: "list=" + y.url.Query().Get("list"),
		},
		y.re,
		true,
	)
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
				re,
				p,
				strings.TrimSpace(html.UnescapeString(string(matches[i][3]))),
			},
		)
	}

	return results, nil
}
