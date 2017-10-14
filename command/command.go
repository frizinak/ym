package command

import (
	"strconv"
	"strings"

	"github.com/frizinak/ym/search"
)

type Command struct {
	buf    []rune
	result search.Result
	done   bool
}

func (c *Command) Append(b rune) *Command {
	c.buf = append(c.buf, b)

	return c
}

func (c *Command) Pop() *Command {
	if len(c.buf) != 0 {
		c.buf = c.buf[:len(c.buf)-1]
	}

	return c
}

func (c *Command) Done() bool {
	return c.done ||
		c.Next() || c.Prev() || c.Pause() ||
		c.SeekForward() || c.SeekBack()
}

func (c *Command) SetDone() *Command {
	c.done = true

	return c
}

func (c *Command) IsText() bool {
	return !(c.Next() || c.Prev() || c.Pause() ||
		c.SeekForward() || c.SeekBack()) &&
		(len(c.buf) == 0 || (c.buf[0] != ':' && c.buf[0] != '/')) &&
		len(c.Choices()) == 0
}

func (c *Command) String() string        { return string(c.buf) }
func (c *Command) Result() search.Result { return c.result }

func (c *Command) SetResult(r search.Result) *Command {
	c.result = r

	return c
}

func (c *Command) Prev() bool { return len(c.buf) == 1 && c.buf[0] == '<' }
func (c *Command) Next() bool { return len(c.buf) == 1 && c.buf[0] == '>' }

func (c *Command) SeekBack() bool    { return len(c.buf) == 1 && c.buf[0] == '[' }
func (c *Command) SeekForward() bool { return len(c.buf) == 1 && c.buf[0] == ']' }

func (c *Command) Pause() bool {
	return len(c.buf) == 1 && (c.buf[0] == '.' || c.buf[0] == ' ')
}

func (c *Command) Search() string {
	if len(c.buf) != 0 && c.buf[0] == '/' {
		return string(c.buf[1:])
	}

	return ""
}

func (c *Command) Info() int {
	if len(c.buf) > 1 && c.buf[0] == ':' {
		i, err := strconv.Atoi(string(c.buf[1:]))
		if err != nil {
			return 0
		}

		return i
	}

	return 0
}

func (c *Command) Move() (from int, to int) {
	s := c.fields("move", 2)
	if s == nil {
		return
	}

	f, err := strconv.Atoi(s[0])
	if err != nil || f <= 0 {
		return
	}
	t, err := strconv.Atoi(s[1])
	if err != nil || f <= 0 {
		return
	}

	from = f
	to = t
	return
}

func (c *Command) Delete() []int {
	s := c.fields("delete", 1)
	if s == nil {
		return nil
	}

	return getInts(s[0])
}

func (c *Command) Scroll() int {
	s := c.fields("scroll", 1)
	if s == nil {
		return 0
	}

	f, err := strconv.Atoi(s[0])
	if err != nil {
		return 0
	}

	return f
}

func (c *Command) Volume() int {
	s := c.fields("volume", 1)
	if s == nil {
		return 0
	}

	f, err := strconv.Atoi(s[0])
	if err != nil {
		return 0
	}

	return f
}

func (c *Command) fields(start string, amount int) []string {
	if len(c.buf) == 0 || c.buf[0] != ':' {
		return nil
	}

	s := strings.Fields(string(c.buf[1:]))
	if len(s) != amount+1 || len(s[0]) == 0 || !strings.HasPrefix(start, s[0]) {
		return nil
	}

	return s[1:]
}

func (c *Command) Clear() bool {
	return c.fields("clear", 0) != nil
}

func (c *Command) Rand() bool {
	str := c.String()
	return str == ":rand" || str == ":random" || str == ":shuffle"
}

func (c *Command) Playlist() bool {
	str := c.String()
	return str == ":list" || str == ":queue" || str == ":playlist"
}

func (c *Command) Back() bool {
	str := c.String()
	return str == ":prev" || str == ":back"
}

func (c *Command) Forward() bool {
	str := c.String()
	return str == ":next" || str == ":forward"
}

func (c *Command) Exit() bool {
	str := c.String()
	return str == ":exit" || str == ":q" || str == ":quit"
}

func (c *Command) Choices() []int {
	return getInts(c.String())
}

func (c *Command) Choice() int {
	if len(c.buf) == 0 {
		return 0
	}
	i, err := strconv.Atoi(c.String())
	if err != nil {
		return 0
	}

	return i
}

func (c *Command) Equal(cmd *Command) bool {
	if cmd == nil || len(c.buf) != len(cmd.buf) {
		return false
	}

	for i := range c.buf {
		if c.buf[i] != cmd.buf[i] {
			return false
		}
	}

	return true
}

func (c *Command) Clone() *Command {
	b := make([]rune, len(c.buf))
	copy(b, c.buf)
	return New(b)
}

func (c *Command) Buffer() []rune {
	return c.buf
}

func (c *Command) Truncate() {
	c.buf = make([]rune, 0)
}

func New(buffer []rune) *Command {
	if buffer == nil {
		buffer = make([]rune, 0)
	}

	return &Command{buffer, nil, false}
}

func getInts(str string) []int {
	s := strings.FieldsFunc(str, func(r rune) bool { return r == ',' })

	ints := make([]int, 0, len(s))
	for i := range s {
		rng := strings.FieldsFunc(s[i], func(r rune) bool { return r == '-' })
		if len(rng) == 2 {
			rngInts := getRange(rng[0], rng[1])
			if rngInts != nil {
				ints = append(ints, rngInts...)
				continue
			}
		}

		in, err := strconv.Atoi(s[i])
		if err != nil || in <= 0 {
			return nil
		}

		ints = append(ints, in)
	}

	return ints
}

func getRange(from, to string) []int {
	f, err := strconv.Atoi(from)
	if err != nil {
		return nil
	}

	t, err := strconv.Atoi(to)
	if err != nil {
		return nil
	}

	if f > t {
		return nil
	}

	ints := make([]int, 0, t-f+1)
	for i := f; i <= t; i++ {
		ints = append(ints, i)
	}

	return ints
}
