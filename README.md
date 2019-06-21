# Simple youtube music player

`go get github.com/frizinak/ym/cmd/ym`

or

`go get -tags nolibmpv github.com/frizinak/ym/cmd/ym`

## Requirements

- Playing audio: libmpv. (or with `-tags nolibmpv`: mplayer or ffplay binaries)
- Extracting audio (optional, to save diskspace): ffmpeg or mencoder

## Search

`> kendrick`

```
 01)  Kendrick Lamar - HUMBLE.
 02)  Kendrick Lamar - LOYALTY. ft. Rihanna
 03)  Kendrick Lamar - ELEMENT.
 ...
```

## Pick some

`> 1,2`


## Other commands:

Use `:help`

# Thx

- https://github.com/rylio/ytdl
- https://github.com/PuerkitoBio/goquery
- https://github.com/mattn/go-runewidth
- https://github.com/nsf/termbox-go

