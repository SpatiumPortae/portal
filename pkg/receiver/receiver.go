package receiver

import (
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt             *crypt.Crypt
	payloadSize       int64
	rendezvousAddress string
	rendezvousPort    int
	ui                chan<- UIUpdate
	usedRelay         bool
}

func NewReceiver(programOptions models.ProgramOptions) *Receiver {
	return &Receiver{
		rendezvousAddress: programOptions.RendezvousAddress,
		rendezvousPort:    programOptions.RendezvousPort,
	}
}

func WithUI(r *Receiver, ui chan<- UIUpdate) *Receiver {
	r.ui = ui
	return r
}

func (r *Receiver) UsedRelay() bool {
	return r.usedRelay
}

func (r *Receiver) PayloadSize() int64 {
	return r.payloadSize
}

func (r *Receiver) RendezvousAddress() string {
	return r.rendezvousAddress
}

func (r *Receiver) RendezvousPort() int {
	return r.rendezvousPort
}

func (r *Receiver) updateUI(progress float32) {
	if r.ui == nil {
		return
	}
	r.ui <- UIUpdate{Progress: progress}
}
