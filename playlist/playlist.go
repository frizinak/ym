package playlist

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/gob"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/search"
)

type storable struct {
	Raw        []rune
	ResultType string
	Result     []byte
}

func init() {
	gob.Register([]*storable{})
}

// Playlist is thread safe
type Playlist struct {
	file     string
	list     []*command.Command
	sem      sync.RWMutex
	d        chan struct{}
	i        int
	changed  bool
	update   chan<- struct{}
	scroll   int
	scrolled bool
}

func New(file string, size int, updates chan<- struct{}) *Playlist {
	return &Playlist{
		file:   file,
		list:   make([]*command.Command, 0, size),
		d:      make(chan struct{}, 0),
		update: updates,
	}
}

func (p *Playlist) updated(superficial bool) {
	if !superficial {
		p.changed = true
	}
	p.update <- struct{}{}
}

func (p *Playlist) Save(onlyIfChanged bool) (err error) {
	if onlyIfChanged {
		p.sem.RLock()
		c := p.changed
		p.sem.RUnlock()
		if !c {
			return nil
		}
	}

	var f *os.File
	tmp := p.file + "." + strconv.FormatInt(time.Now().UnixNano(), 36)
	f, err = os.Create(tmp)
	if err != nil {
		return err
	}

	w := gzip.NewWriter(f)
	p.sem.RLock()
	defer func() {
		w.Close()
		if err != nil {
			os.Remove(tmp)
		}
		err = f.Close()
		p.sem.RUnlock()
	}()

	enc := gob.NewEncoder(w)
	i := make([]byte, 5)

	binary.LittleEndian.PutUint32(i, uint32(p.i))
	i[4] = 10

	if _, err = w.Write(i); err != nil {
		return err
	}

	list := make([]*storable, len(p.list))
	for i, c := range p.list {
		r := c.Result()
		var d []byte
		var to string
		if r != nil {
			to = search.ResultTypeName(r)
			d = r.Marshal()
		}

		list[i] = &storable{c.Buffer(), to, d}
	}

	if err = enc.Encode(list); err != nil {
		return err
	}

	err = os.Rename(tmp, p.file)
	p.changed = err != nil
	return err
}

func (p *Playlist) Load() error {
	p.sem.Lock()
	defer p.sem.Unlock()
	f, err := os.Open(p.file)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	r := bufio.NewReader(gr)
	i := make([]byte, 5)
	if _, err = r.Read(i); err != nil {
		return err
	}

	index := int(binary.LittleEndian.Uint32(i[:4]))

	raw := make([]*storable, 0)
	dec := gob.NewDecoder(r)
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	list := make([]*command.Command, len(raw))
	for i, s := range raw {
		c := command.New(s.Raw)
		if s.ResultType != "" {
			r := search.ResultType(s.ResultType)
			if err := r.Unmarshal(s.Result); err != nil {
				return err
			}
			c.SetResult(r)
		}

		list[i] = c
	}

	p.list = list
	p.i = index - 1
	if p.i < 0 {
		p.i = 0
	}

	return nil
}

func (p *Playlist) Add(cmd *command.Command) {
	if cmd.Result() == nil {
		return
	}

	p.sem.Lock()
	select {
	case p.d <- struct{}{}:
	default:
	}

	p.list = append(p.list, cmd)
	p.updated(false)
	p.sem.Unlock()
}

func (p *Playlist) Del(ix int) {
	p.sem.Lock()
	if ix >= 0 && ix < len(p.list) {
		if p.i > ix && p.i > 0 {
			p.i--
		}
		p.list = append(p.list[:ix], p.list[ix+1:]...)
		p.updated(false)
		select {
		case p.d <- struct{}{}:
		default:
		}
	}
	p.sem.Unlock()
}

func (p *Playlist) List() []*command.Command {
	p.sem.RLock()
	r := make([]*command.Command, len(p.list))
	copy(r, p.list)
	p.sem.RUnlock()
	return r
}

func (p *Playlist) Length() int {
	p.sem.RLock()
	l := len(p.list)
	p.sem.RUnlock()
	return l
}

func (p *Playlist) ResultList() []search.Result {
	p.sem.RLock()
	r := make([]search.Result, 0, len(p.list))
	for i := range p.list {
		res := p.list[i].Result()
		if res != nil {
			r = append(r, res)
		}
	}
	p.sem.RUnlock()
	return r
}

func (p *Playlist) Scroll(amount int) {
	if amount == 0 {
		return
	}

	p.sem.Lock()
	p.scrolled = true
	p.scroll += amount
	if p.scroll > len(p.list) {
		p.scroll = len(p.list)
	}

	if p.scroll < -len(p.list) {
		p.scroll = -len(p.list)
	}

	p.updated(true)
	p.sem.Unlock()
}

func (p *Playlist) ResetScroll() {
	p.sem.Lock()
	p.scroll = 0
	p.scrolled = false
	p.updated(true)
	p.sem.Unlock()
}

func (p *Playlist) Surrounding(amount int) (firstIndex int, activeIndex int, r []search.Result) {
	p.sem.RLock()
	r = make([]search.Result, 0, amount)
	activeIndex = p.i - 1
	if activeIndex < 0 {
		activeIndex = 0
	}

	offset := activeIndex - amount/2
	if p.scrolled {
		offset = p.scroll
	}

	if offset+amount/2 >= len(p.list)-amount/2 {
		offset = len(p.list) - amount
	}

	if offset < 0 {
		offset = 0
	}

	p.scroll = offset

	firstIndex = offset
	activeIndex -= offset

	for {
		if offset >= len(p.list) {
			break
		}
		res := p.list[offset].Result()
		if res != nil {
			r = append(r, res)
		}

		offset++
		if len(r) == amount {
			break
		}
	}

	p.sem.RUnlock()
	return
}

func (p *Playlist) Truncate() {
	p.sem.Lock()
	p.list = make([]*command.Command, 0, cap(p.list))
	p.i = 0
	p.updated(false)
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
	p.updated(false)
	p.sem.Unlock()
	return r
}

func (p *Playlist) At(ix int) *command.Command {
	p.sem.RLock()
	if ix < 0 || ix >= len(p.list) {
		return nil
	}

	r := p.list[ix]
	p.sem.RUnlock()
	return r
}

func (p *Playlist) Next(i int) {
	if i <= 1 {
		return
	}

	p.sem.Lock()

	p.i += i - 1
	if p.i > len(p.list)+1 {
		p.i = len(p.list) + 1
	}

	p.updated(false)
	p.sem.Unlock()
}

func (p *Playlist) Prev(i int) {
	p.sem.Lock()

	select {
	case p.d <- struct{}{}:
		p.i = len(p.list) - 1
		i--
	default:
	}

	if i > 0 {
		if p.i <= len(p.list) {
			p.i--
		}

		p.i -= i

	}

	if p.i < 0 {
		p.i = 0
	}

	p.updated(false)
	p.sem.Unlock()
}

func (p *Playlist) Index() int {
	p.sem.RLock()
	i := p.i - 1
	p.sem.RUnlock()
	return i
}

func (p *Playlist) SetIndex(i int) {
	p.sem.Lock()
	if i > len(p.list) {
		i = len(p.list)
	}

	if i < 0 {
		i = 0
	}
	p.i = i
	p.updated(false)
	select {
	case p.d <- struct{}{}:
	default:
	}

	p.sem.Unlock()
}

func (p *Playlist) Move(from, to int) {
	if from == to {
		return
	}

	d := -1
	min, max := to, from
	if from < to {
		min, max = from, to
		d = 1
	}

	p.sem.Lock()
	if min >= 0 && max < len(p.list) {
		s := p.list[from]
		for i := from; i >= min && i <= max; i += d {
			if i+d < 0 || i+d >= len(p.list) {
				continue
			}
			p.list[i] = p.list[i+d]
		}
		p.list[to] = s

		switch {
		case from == p.i-1:
			p.i = to + 1
		case from < p.i-1 && to >= p.i-1:
			p.i--
		case from > p.i-1 && to <= p.i-1:
			p.i++
		}

		p.updated(false)
	}
	p.sem.Unlock()
}
