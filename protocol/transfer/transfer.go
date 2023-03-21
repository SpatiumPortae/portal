// transfer.go specifies the necessary messaging needed for the transfer protocol.
package transfer

import (
	"fmt"
	"net"
	"strings"
)

// MsgType specifies the message type for the messages in the transfer protocol.
type MsgType int

const (
	TransferError     MsgType = iota // An error has occurred in transferProtocol
	ReceiverHandshake                // Receiver exchange its IP via the rendezvous server to the sender
	SenderHandshake                  // Sender exchanges IP, port and payload size to the receiver via the rendezvous server
	ReceiverDirectCommunication
	SenderDirectAck            // Sender ACKs the request for direct communication
	ReceiverRelayCommunication // Receiver has tried to probe the sender but cannot find it on the subnet, relay communication will be used
	SenderRelayAck             // Sender ACKs the request for relay communication
	ReceiverRequestPayload     // Receiver request the payload from the sender
	SenderPayloadSent          // Sender announces that the entire file has been transferred
	ReceiverPayloadAck         // Receiver ACKs that is has received the payload
	SenderClosing              // Sender announces that it is closing the connection
	ReceiverClosingAck         // Receiver ACKs the closing of the connection
)

type Type int

const (
	Unknown Type = iota
	Direct
	Relay
)

// Msg specifies a message in the transfer protocol.
type Msg struct {
	Type    MsgType `json:"type"`
	Payload Payload `json:"payload,omitempty"`
}

type Payload struct {
	IP          net.IP `json:"ip,omitempty"`
	Port        int    `json:"port,omitempty"`
	PayloadSize int64  `json:"payload_size,omitempty"`
}

func (t Msg) Bytes() []byte {
	return []byte(fmt.Sprintf("%v", t))
}

type Error struct {
	Expected []MsgType
	Got      MsgType
}

func (e Error) Error() string {
	var expectedMessageTypes []string
	for _, expectedType := range e.Expected {
		expectedMessageTypes = append(expectedMessageTypes, expectedType.Name())
	}
	oneOfExpected := strings.Join(expectedMessageTypes, ", ")
	return fmt.Sprintf("wrong message type, expected one of: (%s), got: (%s)", oneOfExpected, e.Got.Name())
}

func (t MsgType) Name() string {
	switch t {
	case TransferError:
		return "TransferError"
	case ReceiverHandshake:
		return "ReceiverHandshake"
	case SenderHandshake:
		return "SenderHandshake"
	case ReceiverRelayCommunication:
		return "ReceiverRelayCommunication"
	case SenderRelayAck:
		return "SenderRelayAck"
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
