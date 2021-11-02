package receiver

import "www.github.com/ZinoKader/portal/pkg/crypt"

type Receiver struct {
	crypt *crypt.Crypt
	state TransferState
	ui    chan<- UIUpdate
}
