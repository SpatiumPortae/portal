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

type UIUpdate struct {
	State    TransferState
	Progress float32
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
