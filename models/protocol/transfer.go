// transfer.go specifies the necessary messaging needed for the transfer protocol.
package protocol

import (
	"fmt"
	"net"
)

// TransferMessageType specifies the message type for the messages in the transfer protocol.
type TransferMessageType int

const (
	TransferError          TransferMessageType = iota // An error has occured in transferProtocol.
	ReceiverHandshake                                 // Receiver exchange its IP via the rendezvous server to the sender.
	SenderHandshake                                   // Sender exchanges IP, port and payload size to the receiver via the rendezvous server.
	ReceiverTransit                                   // Receiver has tried to probe the sender but cannot find it on the subnet, transit communication will be used.
	SenderTransitAck                                  // Sender ACKs the request for transit communication.
	ReceiverRequestPayload                            // Receiver request the payload from the sender.
	SenderPayloadSent                                 // Sender announces that the entire file has been transfered.
	ReceiverPayloadAck                                // Receiver ACKs that is has received the payload.
	SenderClosing                                     // Sender announces that it is closing the connection.
	ReceiverClosingAck                                // Receiver ACKs the closing of the connection.
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

func NewWrongMessageTypeError(expected, got TransferMessageType) *WrongMessageTypeError {
	return &WrongMessageTypeError{
		expected: expected,
		got:      got,
	}
}

func (e *WrongMessageTypeError) Error() string {
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
	case ReceiverTransit:
		return "ReceiverTransit"
	case SenderTransitAck:
		return "SenderTransitAck"
	case ReceiverRequestPayload:
		return "ReceiverRequestPayload"
	case SenderPayloadSent:
		return "SenderPayloadSent"
	case ReceiverPayloadAck:
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
