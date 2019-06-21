// +build nolibmpv

package player

import "errors"

type LibMPV struct {
}

func NewLibMPV(volume chan<- int, seek chan<- float64) Player {
	return &LibMPV{}
}

func (m *LibMPV) Spawn(file string, params []Param) (chan Command, func(), error) {
	return nil, nil, errors.New("Not supported")
}

func (m *LibMPV) Name() string {
	return "?"
}

func (m *LibMPV) Supported() bool {
	return false
}
