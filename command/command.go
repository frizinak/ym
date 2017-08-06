package command

import (
	"strconv"
	"strings"

	"github.com/frizinak/ym/search"
)

type Command struct {
	raw    string
	cmd    byte
	arg    []string
	choice int
	result search.Result
}

func (c *Command) parse() {
	if c.cmd != 0 || c.arg != nil || c.choice != 0 {
		return
	}

	c.arg = strings.Split(c.raw, " ")
	if len(c.raw) == 0 {
		return
	}

	if c.raw[0] == '!' ||
		c.raw[0] == '@' ||
		c.raw[0] == ':' {
		c.cmd = c.raw[0]
		c.arg = strings.Split(c.raw[1:], " ")
	}

	var err error
	c.choice, err = strconv.Atoi(c.arg[0])
	if err == nil {
		c.arg = c.arg[1:]
	}
}

func (c *Command) Raw() string               { return c.raw }
func (c *Command) IsCmd() bool               { return c.cmd != 0 }
func (c *Command) IsChoice() bool            { return c.choice != 0 }
func (c *Command) Cmd() byte                 { return c.cmd }
func (c *Command) Choice() int               { return c.choice }
func (c *Command) Result() search.Result     { return c.result }
func (c *Command) SetResult(r search.Result) { c.result = r }
func (c *Command) ArgStr() string            { return strings.Join(c.arg, " ") }
func (c *Command) Arg(i int) string {
	if i < 0 || i >= len(c.arg) {
		return ""
	}

	return c.arg[i]
}

func (c *Command) Clone() *Command {
	return &Command{
		c.raw,
		c.cmd,
		c.arg,
		c.choice,
		c.result,
	}
}

func (c *Command) Equal(cmd *Command) bool {
	if cmd == nil {
		return false
	}

	return c.raw == cmd.raw
}

func New(in string) *Command {
	c := &Command{raw: strings.TrimSpace(in)}
	c.parse()
	return c
}
