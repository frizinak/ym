// +build !nolibmpv

package player

import (
	"github.com/YouROK/go-mpv/mpv"
)

type LibMPV struct {
	cmdPause   map[bool]string
	volume     float64
	volumeChan chan<- int
}

func NewLibMPV(volume chan<- int) Player {
	return &LibMPV{
		map[bool]string{
			true:  "yes",
			false: "no",
		},
		100,
		volume,
	}
}

func (m *LibMPV) Spawn(file string, params []Param) (chan Command, func(), error) {
	commands := make(chan Command, 0)

	p := mpv.Create()
	m.adjustVolume(p, 0)

	c := make(chan *mpv.Event)
	go func() {
		for {
			e := p.WaitEvent(.05)
			if e.Event_Id == mpv.EVENT_END_FILE ||
				e.Event_Id == mpv.EVENT_SHUTDOWN {
				p.TerminateDestroy()
				close(c)
				return
			}
			c <- e
		}
	}()

	for _, par := range params {
		switch par {
		case PARAM_NO_VIDEO:
			p.SetOption("vid", mpv.FORMAT_FLAG, false)
		case PARAM_SILENT:
			p.SetOption("really-quiet", mpv.FORMAT_FLAG, true)
		}
	}

	if err := p.Initialize(); err != nil {
		return nil, nil, err
	}

	go func() {
		paused := false
	outer:
		for command := range commands {
			switch command {
			case CMD_PAUSE:
				paused = !paused
				p.SetPropertyString("pause", m.cmdPause[paused])

			case CMD_STOP:
				break outer

			case CMD_VOL_DOWN:
				m.adjustVolume(p, -5)

			case CMD_VOL_UP:
				m.adjustVolume(p, 5)
			}
		}

		p.Command([]string{"quit"})
	}()

	wait := func() {
		for range c {
		}
	}

	p.Command([]string{"loadfile", file})
	return commands, wait, nil
}

func (m *LibMPV) Supported() bool {
	return true
}

func (m *LibMPV) adjustVolume(p *mpv.Mpv, adjustment float64) error {
	m.volume += adjustment
	if m.volume < 0 {
		m.volume = 0
	} else if m.volume > 100 {
		m.volume = 100
	}

	p.SetProperty("volume", mpv.FORMAT_DOUBLE, m.volume)

	select {
	case m.volumeChan <- int(m.volume):
	default:
	}

	return nil
}
