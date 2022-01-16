package receiver

import (
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type ReceiverOptions func(*Receiver)

// Receiver struct that encapsulates all necessary information to connect to Rendezvous,
// exchange cryptographic information and receive files.
type Receiver struct {
	crypt             *crypt.Crypt    // Cryptographic information.
	payloadSize       int64           // Size of payload in bytes.
	rendezvousAddress string          // Address of the rendezvous server.
	rendezvousPort    int             // Port that the rendezvous server is running on.
	ui                chan<- UIUpdate // Channel that can be used to communicate with the UI.
	usedRelay         bool            // Bool that signifies if the relay is being used for file transfer.
}

// New creates a new receiver with the provided options.
func New(programOptions models.ProgramOptions, opts ...ReceiverOptions) *Receiver {

	r := &Receiver{
		rendezvousAddress: programOptions.RendezvousAddress,
		rendezvousPort:    programOptions.RendezvousPort,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// OPTIONS

// WithUI returns an option to set the channel for UI communication.
func WithUI(ui chan<- UIUpdate) ReceiverOptions {
	return func(r *Receiver) {
		r.ui = ui
	}
}

// GETTERS

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

// HELPERS

func (r *Receiver) updateUI(progress float32) {
	if r.ui == nil {
		return
	}
	r.ui <- UIUpdate{Progress: progress}
}
