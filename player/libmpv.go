// +build !nolibmpv

package player

import (
	"time"

	"github.com/YouROK/go-mpv/mpv"
)

type LibMPV struct {
	cmdPause   map[bool]string
	volume     float64
	volumeChan chan<- int
	seekChan   chan<- *Pos
	pos        pos
}

type pos struct {
	timePos float64
	timeEnd float64
}

func (p pos) Pos() *Pos {
	return &Pos{
		time.Duration(p.timePos * float64(time.Second)),
		time.Duration(p.timeEnd * float64(time.Second)),
	}
}

func NewLibMPV(volume chan<- int, seek chan<- *Pos) Player {
	return &LibMPV{
		map[bool]string{
			true:  "yes",
			false: "no",
		},
		100,
		volume,
		seek,
		pos{0, 0},
	}
}

func (m *LibMPV) Name() string {
	return "libmpv"
}

func (m *LibMPV) Spawn(file string, params []Param) (chan Command, func(), error) {
	commands := make(chan Command)
	done := make(chan struct{})

	p := mpv.Create()
	m.adjustVolume(p, 0)

	c := make(chan *mpv.Event)
	go func() {
		for {
			e := p.WaitEvent(.05)
			if e.Event_Id == mpv.EVENT_END_FILE ||
				e.Event_Id == mpv.EVENT_SHUTDOWN {
				done <- struct{}{}
				p.TerminateDestroy()
				close(c)
				return
			}
			c <- e
		}
	}()

	for _, par := range params {
		switch par {
		case ParamNoVideo:
			p.SetOption("vid", mpv.FORMAT_FLAG, false)
		case ParamSilent:
			p.SetOption("really-quiet", mpv.FORMAT_FLAG, true)
		}
	}

	if err := p.Initialize(); err != nil {
		return nil, nil, err
	}

	go func() {
		paused := false
		closed := false
		quickTO := time.Millisecond * 200
		slowTO := time.Second * 2
		quick := time.After(slowTO)
		slow := time.After(quickTO)
		for {
			select {
			case <-done:
				closed = true
			case command, ok := <-commands:
				if !ok && closed {
					return
				}

				if closed {
					continue
				}

				switch command {
				case CmdPause:
					paused = !paused
					p.SetPropertyString("pause", m.cmdPause[paused])

				case CmdStop:
					p.Command([]string{"quit"})

				case CmdVolDown:
					m.adjustVolume(p, -5)

				case CmdVolUp:
					m.adjustVolume(p, 5)

				case CmdSeekBackward:
					m.seek(p, -time.Second*10)

				case CmdSeekForward:
					m.seek(p, time.Second*10)
				}
			case <-quick:
				quick = time.After(quickTO)
				if closed {
					continue
				}

				m.positionQuick(p)
			case <-slow:
				slow = time.After(slowTO)
				if closed {
					continue
				}

				m.position(p)
			}
		}
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

func (m *LibMPV) seek(p *mpv.Mpv, adjustment time.Duration) error {
	_cur, err := p.GetProperty("time-pos", mpv.FORMAT_INT64)
	if err != nil {
		return err
	}

	if adjustment != 0 {
		cur := _cur.(int64) + int64(adjustment.Seconds())
		if cur < 0 {
			cur = 0
		}

		p.SetProperty("time-pos", mpv.FORMAT_INT64, cur)
	}

	m.positionQuick(p)

	return nil
}

func (m *LibMPV) positionQuick(p *mpv.Mpv) error {
	_cur, err := p.GetProperty("time-pos", mpv.FORMAT_DOUBLE)
	if err != nil {
		return err
	}

	m.pos.timePos = _cur.(float64)

	select {
	case m.seekChan <- m.pos.Pos():
	default:
	}

	return nil
}

func (m *LibMPV) position(p *mpv.Mpv) error {
	_byteCur, err := p.GetProperty("stream-pos", mpv.FORMAT_DOUBLE)
	if err != nil {
		return err
	}
	_byteTotal, err := p.GetProperty("stream-end", mpv.FORMAT_DOUBLE)
	if err != nil {
		return err
	}

	byteTotal := _byteTotal.(float64)
	if byteTotal < 1 {
		byteTotal = 1
	}

	bytePos := _byteCur.(float64) / byteTotal
	if bytePos > 1.0 {
		bytePos = 1.0
	}

	m.pos.timeEnd = float64(m.pos.timePos) / bytePos
	return m.positionQuick(p)
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
