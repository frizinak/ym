package playlist

import (
	"sync"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/search"
)

// Playlist is thread safe
type Playlist struct {
	list []*command.Command
	sem  sync.Mutex
	d    chan *command.Command
}

func New(size int) *Playlist {
	return &Playlist{
		list: make([]*command.Command, 0, size),
		d:    make(chan *command.Command, 0),
	}
}

func (p *Playlist) Add(cmd *command.Command) {
	p.sem.Lock()
	select {
	case p.d <- cmd:
		p.sem.Unlock()
		return
	default:
	}

	p.list = append(p.list, cmd)
	p.sem.Unlock()
}

func (p *Playlist) List() []*command.Command {
	p.sem.Lock()
	r := make([]*command.Command, len(p.list))
	copy(r, p.list)
	p.sem.Unlock()
	return r
}

func (p *Playlist) Length() int {
	p.sem.Lock()
	l := len(p.list)
	p.sem.Unlock()
	return l
}

func (p *Playlist) ResultList() []search.Result {
	p.sem.Lock()
	r := make([]search.Result, 0, len(p.list))
	for i := range p.list {
		res := p.list[i].Result()
		if res != nil {
			r = append(r, res)
		}
	}
	p.sem.Unlock()
	return r
}

func (p *Playlist) Truncate() {
	p.sem.Lock()
	p.list = make([]*command.Command, 0, cap(p.list))
	p.sem.Unlock()
}

func (p *Playlist) Pop() *command.Command {
	p.sem.Lock()
	if len(p.list) == 0 {
		p.sem.Unlock()
		return <-p.d
	}

	r := p.list[0]
	p.list = p.list[1:]
	p.sem.Unlock()
	return r
}
