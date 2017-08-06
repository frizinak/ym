package player

import (
	"errors"
	"os/exec"
)

const (
	CMD_NIL Command = iota
	CMD_PAUSE
	CMD_STOP
	CMD_NEXT
	CMD_PREV

	PARAM_NO_VIDEO Param = "no-video"
	PARAM_SILENT   Param = "silent"
)

type Command int

type Param string

type Player interface {
	Spawn(file string, params []Param) (chan Command, func(), error)
	Supported() bool
}

func FindSupportedPlayer(players ...Player) (Player, error) {
	for _, p := range players {
		if p.Supported() {
			return p, nil
		}
	}

	return nil, errors.New("No supported player found")
}

type GenericPlayer struct {
	cmd        string
	args       []string
	paramMap   map[Param][]string
	commandMap map[Command][]byte
}

func (m *GenericPlayer) Supported() bool {
	return binaryInPath(m.cmd)
}

func (m *GenericPlayer) Spawn(file string, params []Param) (
	chan Command,
	func(),
	error,
) {
	args := m.args
	if args == nil {
		args = make([]string, 0, len(params)+1)
	}
	for _, p := range params {
		if a, ok := m.paramMap[p]; ok {
			args = append(args, a...)
			continue
		}
	}

	args = append(args, file)
	cmd := exec.Command(m.cmd, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	commands := make(chan Command, 1)
	wait := func() {
		cmd.Wait()
	}

	go func() {
	outer:
		for command := range commands {
			if c, ok := m.commandMap[command]; ok {
				stdin.Write(c)
			}

			switch command {
			case CMD_STOP:
				break outer
			}
		}

		cmd.Process.Kill()
		cmd.Wait()
	}()

	return commands, wait, nil
}

func binaryInPath(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
