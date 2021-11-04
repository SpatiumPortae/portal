package receiver

import (
	"log"

	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt       *crypt.Crypt
	payloadSize int64
	logger      *log.Logger
	ui          chan<- UIUpdate
	usedRelay   bool
}

func NewReceiver(logger *log.Logger) *Receiver {
	return &Receiver{logger: logger}
}

func WithUI(r *Receiver, ui chan<- UIUpdate) *Receiver {
	r.ui = ui
	return r
}

func (r *Receiver) DidUseRelay() bool {
	return r.usedRelay
}

func (r *Receiver) GetPayloadSize() int64 {
	return r.payloadSize
}

func (r *Receiver) updateUI(progress float32) {
	if r.ui == nil {
		return
	}
	r.ui <- UIUpdate{Progress: progress}
}
