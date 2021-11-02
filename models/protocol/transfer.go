// transfer.go specifies the necessary messaging needed for the transfer protocol.
package protocol

import (
	"fmt"
	"net"
)

// TransferMessageType specifies the message type for the messages in the transfer protocol.
type TransferMessageType int

const (
	TransferError TransferMessageType = iota // An Error has
	ReceiverHandshake
	SenderHandshake
	ReceiverRequestPayload
	SenderPayloadSent
	ReceiverAckPayload
	SenderClosing
	ReceiverClosingAck
)

// TransferMessage specifies a message in the transfer protocol.
type TransferMessage struct {
	Type    TransferMessageType `json:"type"`
	Payload interface{}         `json:"payload,omitempty"`
}

func (t TransferMessage) Bytes() []byte {
	return []byte(fmt.Sprintf("%v", t))
}

type ReceiverHandshakePayload struct {
	IP net.IP `json:"ip"`
}

// SenderHandshakePayload specifies a payload type for announcing the payload size.
type SenderHandshakePayload struct {
	IP          net.IP `json:"ip"`
	Port        int    `json:"port"`
	PayloadSize int64  `json:"payload_size"`
}

type WrongMessageTypeError struct {
	expected TransferMessageType
	got      TransferMessageType
}

func NewWrongMessageTypeError(expected, got TransferMessageType) WrongMessageTypeError {
	return WrongMessageTypeError{
		expected: expected,
		got:      got,
	}
}

func (e WrongMessageTypeError) Error() string {
	return fmt.Sprintf("wrong message type, expected type: %d(%s), got: %d(%s)", e.expected, e.expected.Name(), e.got, e.got.Name())
}

func (t TransferMessageType) Name() string {
	switch t {
	case TransferError:
		return "TransferError"
	case ReceiverHandshake:
		return "ReceiverHandshake"
	case SenderHandshake:
		return "SenderHandshake"
	case ReceiverRequestPayload:
		return "ReceiverRequestPayload"
	case SenderPayloadSent:
		return "SenderPayloadSent"
	case ReceiverAckPayload:
		return "ReceiverAckPayload"
	case SenderClosing:
		return "SenderClosing"
	case ReceiverClosingAck:
		return "ReceiverClosingAck"
	default:
		return ""
	}
}

//NOTE: should probably implment JSON object for regular string messages as well.
