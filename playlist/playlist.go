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
	d    chan struct{}
	i    int
}

func New(size int) *Playlist {
	return &Playlist{
		list: make([]*command.Command, 0, size),
		d:    make(chan struct{}, 0),
	}
}

func (p *Playlist) Add(cmd *command.Command) {
	p.sem.Lock()
	select {
	case p.d <- struct{}{}:
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
	p.i = 0
	p.sem.Unlock()
}

func (p *Playlist) Read() *command.Command {
	p.sem.Lock()
	if p.i >= len(p.list) {
		p.sem.Unlock()
		<-p.d
		return p.Read()
	}

	r := p.list[p.i]
	p.i++
	p.sem.Unlock()
	return r
}

func (p *Playlist) Prev() {
	p.sem.Lock()

	select {
	case p.d <- struct{}{}:
		p.i = len(p.list) - 1
	default:
		if p.i <= len(p.list) {
			p.i--
		}

		p.i -= 1
	}

	if p.i < 0 {
		p.i = 0
	}

	p.sem.Unlock()
}

func (p *Playlist) Index() int {
	p.sem.Lock()
	i := p.i
	p.sem.Unlock()
	return i
}
