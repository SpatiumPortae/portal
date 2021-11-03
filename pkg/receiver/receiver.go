package receiver

import (
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt       *crypt.Crypt
	payloadSize int64
	ui          chan<- UIUpdate
	usedRelay   bool
}

func NewReceiver() *Receiver {
	return &Receiver{}
}

func WithUI(r *Receiver, ui chan<- UIUpdate) *Receiver {
	r.ui = ui
	return r
}

func (r *Receiver) DidUseRelay() bool {
	return r.usedRelay
}

func (r *Receiver) updateUI(progress float32) {
	if r.ui == nil {
		return
	}
	r.ui <- UIUpdate{Progress: progress}
}
