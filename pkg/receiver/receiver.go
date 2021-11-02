package receiver

import (
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt       *crypt.Crypt
	state       TransferState
	payloadSize int64
	ui          chan<- UIUpdate
}
