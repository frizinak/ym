package playlist

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/frizinak/ym/command"
	"github.com/frizinak/ym/search"
)

type ints []int

func (in ints) Len() int           { return len(in) }
func (in ints) Swap(i, j int)      { in[i], in[j] = in[j], in[i] }
func (in ints) Less(i, j int) bool { return in[i] < in[j] }

type storable struct {
	ResultType string
	Result     string
}

// Playlist is thread safe
type Playlist struct {
	file     string
	list     []*command.Command
	sem      sync.RWMutex
	d        chan struct{}
	i        int
	last     int
	changed  bool
	update   chan<- struct{}
	scroll   int
	scrolled bool
	rand     bool
}

func New(file string, size int, updates chan<- struct{}) *Playlist {
	return &Playlist{
		file:   file,
		list:   make([]*command.Command, 0, size),
		d:      make(chan struct{}),
		update: updates,
	}
}

func (p *Playlist) updated(superficial bool) {
	if !superficial {
		p.changed = true
	}
	go func() { p.update <- struct{}{} }()
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

	p.sem.RLock()
	defer func() {
		if err != nil {
			os.Remove(tmp)
		}
		err = f.Close()
		p.sem.RUnlock()
	}()

	enc := json.NewEncoder(f)
	i := make([]byte, 5)

	binary.LittleEndian.PutUint32(i, uint32(p.i))
	i[4] = 10

	if _, err = f.Write(i); err != nil {
		return err
	}

	for _, c := range p.list {
		r := c.Result()
		if r == nil {
			continue
		}

		to := search.ResultTypeName(r)
		d, err := r.Marshal()
		if err != nil {
			return err
		}

		item := &storable{to, d}

		if err = enc.Encode(item); err != nil {
			return err
		}
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

	r := bufio.NewReader(f)
	i := make([]byte, 5)
	if _, err = r.Read(i); err != nil {
		return err
	}

	index := int(binary.LittleEndian.Uint32(i[:4]))

	raw := make([]*storable, 0)

	var nonCritErr error
	for {
		b, _, err := bufio.NewReaderSize(r, 4096).ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		item := &storable{}
		if err := json.Unmarshal(b, item); err != nil {
			nonCritErr = err
			continue
		}
		raw = append(raw, item)
	}

	list := make([]*command.Command, 0, len(raw))
	for _, s := range raw {
		c := command.New(nil)
		if s.ResultType != "" {
			r := search.ResultType(s.ResultType)
			if err := r.Unmarshal(s.Result); err != nil {
				nonCritErr = err
				continue
			}
			c.SetResult(r)
		}

		list = append(list, c)
	}

	p.list = list
	p.i = index - 1
	if p.i < 0 {
		p.i = 0
	}

	return nonCritErr
}

func (p *Playlist) ToggleRandom() {
	p.sem.Lock()
	p.rand = !p.rand
	p.sem.Unlock()
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

func (p *Playlist) Del(indexes []int) {
	ixs := make(ints, len(indexes))
	copy(ixs, indexes)
	sort.Sort(ixs)

	p.sem.Lock()
	done := make(map[int]struct{}, len(ixs))
	amount := 0
	for _, ix := range ixs {
		if _, ok := done[ix]; ok {
			continue
		}

		ix -= amount
		if ix >= 0 && ix < len(p.list) {
			if p.i > ix && p.i > 0 {
				p.i--
			}
			p.list = append(p.list[:ix], p.list[ix+1:]...)
		}

		done[ix] = struct{}{}
		amount++
	}

	if amount != 0 {
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

func (p *Playlist) ScrollTo(index int) {
	p.sem.RLock()
	amount := index - p.scroll
	p.sem.RUnlock()
	p.Scroll(amount)
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
	activeIndex = p.Index()
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
	p.last = p.i
	if p.rand {
		p.i = rand.Intn(len(p.list))
	}
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
	defer p.sem.RUnlock()
	if p.rand {
		return p.last - 1
	}
	return p.i - 1
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

func (p *Playlist) Search(qry string, offset *int) bool {
	p.sem.RLock()
	o := *offset
	if o > len(p.list) {
		o = 0
	}

	qry = strings.ToLower(qry)
	m := 0
	to := -1

	for i, c := range p.list {
		r := c.Result()
		if r == nil {
			continue
		}

		if strings.Contains(strings.ToLower(r.Title()), qry) {
			if m == o {
				o++
				to = i
				break
			}
			m++
		}
	}

	p.sem.RUnlock()

	if m > 0 && to == -1 {
		o = 0
		*offset = o
		return p.Search(qry, offset)
	}

	*offset = o
	if to > -1 {
		p.ScrollTo(to)
		return true
	}

	return false
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
