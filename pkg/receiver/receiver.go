package receiver

import (
	"fmt"

	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Receiver struct {
	crypt *crypt.Crypt
	state TransferState
	ui    chan<- UIUpdate
}

type WrongStateError struct {
	expected TransferState
	got      TransferState
}

func NewWrongStateError(expected, got TransferState) *WrongStateError {
	return &WrongStateError{
		expected: expected,
		got:      got,
	}
}

func (e *WrongStateError) Error() string {
	return fmt.Sprintf("wrong message type, expected type: %d(%s), got: %d(%s)", e.expected, e.expected.Name(), e.got, e.got.Name())
}
