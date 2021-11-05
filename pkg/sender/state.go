// state.go defines the necessary state types and variables for the sender.
package sender

import "fmt"

type TransferState int

const (
	Initial TransferState = iota
	WaitForFileRequest
	SendingData
	WaitForFileAck
	WaitForCloseMessage
	WaitForCloseAck
)

// UIUpdate is a struct that is continously communicated to the UI (if sender has attached a UI)
type UIUpdate struct {
	State    TransferState
	Progress float32
}

// WrongStateError is a custom error for the Transfer sequence
type WrongStateError struct {
	expected TransferState
	got      TransferState
}

// WrongStateError constructor
func NewWrongStateError(expected, got TransferState) *WrongStateError {
	return &WrongStateError{
		expected: expected,
		got:      got,
	}
}

func (e *WrongStateError) Error() string {
	return fmt.Sprintf("wrong message type, expected type: %d(%s), got: %d(%s)", e.expected, e.expected.Name(), e.got, e.got.Name())
}

// Name returns the associated to the state enum.
func (s TransferState) Name() string {
	switch s {
	case Initial:
		return "Initial"
	case WaitForFileRequest:
		return "WaitForFileRequest"
	case SendingData:
		return "SendingData"
	case WaitForFileAck:
		return "WaitForFileAck"
	case WaitForCloseMessage:
		return "WaitForCloseMessage"
	case WaitForCloseAck:
		return "WaitForCloseAck"
	default:
		return ""
	}
}
