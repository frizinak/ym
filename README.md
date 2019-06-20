# Simple youtube music player

`go get github.com/frizinak/ym/cmd/ym`

## Requirements

- Playing audio: one of: libmpv, mplayer or ffplay
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

- `:list`: view queue
- `:clear`: clear queue
- `:mov <i> <j>`: move item at index i to index j
- `:del <i>`: remove item at index i from the queue
- `:<i>`: show info about item at index i
- `C-d`: scroll down
- `C-u`: scroll up
- `.` or space: pause
- `<` or left arrow key: previous song
- `>` or right arrow key: next song
- `[` seek forward
- `]` seek backward

- `C-q`, `C-c`, `:q`, `:exit`: quit


# Thx

- https://github.com/rylio/ytdl
- https://github.com/PuerkitoBio/goquery
- https://github.com/mattn/go-runewidth
- https://github.com/nsf/termbox-go

