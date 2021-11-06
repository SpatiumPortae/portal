package receiver

import (
	"io/ioutil"
	"log"
	"os"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt             *crypt.Crypt
	payloadSize       int64
	rendezvousAddress string
	rendezvousPort    int
	logger            *log.Logger
	ui                chan<- UIUpdate
	usedRelay         bool
}

func NewReceiver(programOptions models.ProgramOptions) *Receiver {
	logger := log.New(ioutil.Discard, "", 0)
	if programOptions.Verbose {
		logger = log.New(os.Stderr, "VERBOSE: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
	return &Receiver{
		rendezvousAddress: programOptions.RendezvousAddress,
		rendezvousPort:    programOptions.RendezvousPort,
		logger:            logger,
	}
}

func WithUI(r *Receiver, ui chan<- UIUpdate) *Receiver {
	r.ui = ui
	return r
}

func (r *Receiver) Logger() *log.Logger {
	return r.logger
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
