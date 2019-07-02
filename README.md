# Simple youtube music player

Download a [release](https://github.com/frizinak/ym/releases)

or

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

## Tools:

**Makes sure all items are cached in ~/.cache/ym/downloads**

`go get github.com/frizinak/ym/cmd/ym-cache`

**Hardlinks copies in ~/.cache/ym/downloads to whatever dir you specify, with clean filenames.**

`go get github.com/frizinak/ym/cmd/ym-files`


# Thx

- https://github.com/rylio/ytdl
- https://github.com/PuerkitoBio/goquery
- https://github.com/mattn/go-runewidth
- https://github.com/nsf/termbox-go

