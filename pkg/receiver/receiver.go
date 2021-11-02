package receiver

import (
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt       *crypt.Crypt
	payloadSize int64
	ui          chan<- UIUpdate
}

func NewReceiver() *Receiver {
	return &Receiver{}
}

func WithUI(r *Receiver, uiCh chan<- UIUpdate) *Receiver {
	r.ui = uiCh
	return r
}

func (r *Receiver) updateUI(progress float32) {
	if r.ui == nil {
		return
	}
	r.ui <- UIUpdate{Progress: progress}
}
