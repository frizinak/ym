package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

type Transcoder interface {
	Transcode(video io.Reader, audio io.Writer) error
	Ext() string
}

type Entry struct {
	id  string
	ext string
	url *url.URL
}

func (e *Entry) ID() string    { return e.id }
func (e *Entry) Ext() string   { return e.ext }
func (e *Entry) URL() *url.URL { return e.url }

func NewEntry(id, ext string, url *url.URL) *Entry {
	return &Entry{id, ext, url}
}

type Cached struct {
	id   string
	path string
}

func (c *Cached) ID() string {
	return c.id
}

func (c *Cached) Path() string {
	return c.path
}

type Cache struct {
	t       Transcoder
	dir     string
	tempdir string
}

func New(t Transcoder, dir, tempdir string) (*Cache, error) {
	if err := os.MkdirAll(tempdir, 0755); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &Cache{t, dir, tempdir}, nil
}

func (c *Cache) Get(id string) *Cached {
	if c == nil {
		return nil
	}

	r, err := filepath.Glob(path.Join(c.dir, hashFn(id, true)+"*"))
	if err != nil || len(r) == 0 {
		return nil
	}

	return &Cached{id, r[0]}
}

func (c *Cache) Set(e *Entry) error {
	return c.SetProgress(e, nil)
}

func (c *Cache) SetProgress(e *Entry, progress func(written, total int64)) error {
	if c == nil {
		return errors.New("No cache initialized")
	}

	u := e.URL()
	if u == nil {
		return errors.New("url cannot be nil")
	}
	id := e.ID()
	if id == "" {
		return errors.New("id cannot be empty")
	}

	tmp := path.Join(
		c.tempdir,
		hashFn(id, false)+"."+strconv.FormatInt(time.Now().UnixNano(), 36),
	)

	if err := download(c.t, u.String(), tmp, progress); err != nil {
		return err
	}

	ext := e.Ext()
	if c.t != nil {
		ext = c.t.Ext()
	}

	dest := path.Join(c.dir, c.Base(id)+"."+ext)
	dir := path.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.Rename(tmp, dest); err != nil {
		defer os.Remove(tmp)
		return copy(dest, tmp)
	}

	return nil
}

func (c *Cache) Base(id string) string {
	return hashFn(id, true)
}

func hashFn(fn string, dirs bool) string {
	h := sha256.Sum256([]byte(fn))
	p := hex.EncodeToString(h[:])
	if !dirs {
		return p
	}

	parts := make([]string, 9)
	for i := 0; i < 8; i++ {
		parts[i] = p[:2]
		p = p[2:]
	}

	parts[8] = p

	return path.Join(parts...)
}

func copy(dest, src string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	destF, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destF.Close()

	_, err = io.Copy(destF, srcF)
	return err
}

type progressWriter struct {
	w       io.WriteCloser
	size    int64
	written int64
	cb      func(written, total int64)
}

func (p *progressWriter) Write(d []byte) (n int, err error) {
	n, err = p.w.Write(d)
	p.written += int64(n)

	if p.cb != nil {
		if p.written > p.size {
			p.size = p.written
		}
		p.cb(p.written, p.size)
	}

	return n, err
}

func (p *progressWriter) Close() error {
	return p.w.Close()
}

func download(t Transcoder, u, dest string, progress func(int64, int64)) error {
	_f, err := os.Create(dest)
	f := &progressWriter{_f, 0, 0, progress}
	defer f.Close()
	if err != nil {
		return err
	}

	res, err := http.Get(u)
	if res != nil && res.Body != nil {
		defer res.Body.Close()
	}

	f.size = res.ContentLength

	if err != nil {
		return err
	}

	if t != nil {
		return t.Transcode(res.Body, f)
	}

	_, err = io.Copy(f, res.Body)
	return err
}
