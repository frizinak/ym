// +build nolibmpv

package player

import "errors"

type LibMPV struct {
}

func NewLibMPV() Player {
	return &LibMPV{}
}

func (m *LibMPV) Spawn(file string, params []Param) (chan Command, func(), error) {
	return nil, nil, errors.New("Not supported")
}

func (m *LibMPV) Supported() bool {
	return false
}
