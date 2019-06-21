package audio

import (
	"errors"
	"io"
	"os/exec"
)

type Extractor interface {
	Name() string
	Extract(video io.Reader, audio io.Writer) error
	Transcode(video io.Reader, audio io.Writer) error
	Supported() bool
	Ext() string
}

func FindSupportedExtractor(extractors ...Extractor) (Extractor, error) {
	for _, e := range extractors {
		if e.Supported() {
			return e, nil
		}
	}

	return nil, errors.New("No supported extractor found")
}

type GenericExtractor struct {
	cmd  string
	args []string
	ext  string
}

func (m *GenericExtractor) Name() string {
	return m.cmd
}

func (m *GenericExtractor) Supported() bool {
	_, err := exec.LookPath(m.cmd)
	return err == nil
}

func (m *GenericExtractor) Transcode(v io.Reader, a io.Writer) error {
	return m.Extract(v, a)
}

func (m *GenericExtractor) Extract(v io.Reader, a io.Writer) error {
	cmd := exec.Command(m.cmd, m.args...)
	cmd.Stdin = v
	cmd.Stdout = a
	return cmd.Run()
}

func (m *GenericExtractor) Ext() string {
	if m.ext == "" {
		return "aac"
	}

	return m.ext
}
