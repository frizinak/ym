package command

import (
	"strconv"
	"strings"

	"github.com/frizinak/ym/search"
)

type Command struct {
	raw    string
	cmd    byte
	arg    string
	choice int
	result search.Result
}

func (c *Command) parse() {
	if c.cmd != 0 || c.arg != "" || c.choice != 0 {
		return
	}

	c.arg = c.raw
	if len(c.raw) == 0 {
		return
	}

	if c.raw[0] == '!' ||
		c.raw[0] == '@' ||
		c.raw[0] == ':' {
		c.cmd = c.raw[0]
		c.arg = c.raw[1:]
	}

	var err error
	c.choice, err = strconv.Atoi(c.arg)
	if err == nil {
		c.arg = ""
	}
}

func (c *Command) IsCmd() bool               { return c.cmd != 0 }
func (c *Command) IsChoice() bool            { return c.choice != 0 }
func (c *Command) Cmd() byte                 { return c.cmd }
func (c *Command) Arg() string               { return c.arg }
func (c *Command) Choice() int               { return c.choice }
func (c *Command) Result() search.Result     { return c.result }
func (c *Command) SetResult(r search.Result) { c.result = r }

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
