package history

import "github.com/frizinak/ym/search"

type entry struct {
	t string
	r []search.Result
}

// History is not thread safe
type History struct {
	h []*entry
	i int
}

func New(size int) *History {
	return &History{make([]*entry, size), 0}
}

func (h *History) Current() (string, []search.Result) {
	if e := h.h[h.i]; e != nil {
		return e.t, e.r
	}

	return "", nil
}

func (h *History) Write(title string, r []search.Result) {
	if h.i < len(h.h)-1 && h.h[h.i] != nil {
		h.i++
	}

	if h.h[h.i] != nil {
		for i := 0; i < h.i; i++ {
			h.h[i] = h.h[i+1]
		}
	}

	h.h[h.i] = &entry{title, r}
}

func (h *History) Forward() {
	if h.i < len(h.h)-1 && h.h[h.i+1] != nil {
		h.i++
	}
}

func (h *History) Back() {
	if h.i > 0 {
		h.i--
	}
}
