package protocol

import (
	"fmt"
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

type RendezvousMessageType int

const (
	RendezvousToSenderBind        RendezvousMessageType = iota // An ID for this connection is bound and communicated
	SenderToRendezvousEstablish                                // Sender has generated and hashed password
	ReceiverToRendezvousEstablish                              // Passsword has been communicated to receiver who has hashed it
	RendezvousToSenderReady                                    // Rendezvous announces to sender that receiver is connected
	SenderToRendezvousPAKE                                     // Sender sends PAKE information to rendezvous
	RendezvousToReceiverPAKE                                   // Rendezvous forwards PAKE information to receiver
	ReceiverToRendezvousPAKE                                   // Receiver sends PAKE information to rendezvous
	RendezvousToSenderPAKE                                     // Rendezvous forwards PAKE information to receiver
	SenderToRendezvousSalt                                     // Sender sends cryptographic salt to rendezvous
	RendezvousToReceiverSalt                                   // Rendevoux forwards cryptographic salt to receiver
	// From this point there is a safe channel established
	ReceiverToRendezvousClose // Receiver can connect directly to sender, close receiver connection -> close sender connection
	SenderToRendezvousClose   // Transit sequence is completed, close sender connection -> close receiver connection
)

type RendezvousMessage struct {
	Type    RendezvousMessageType `json:"type"`
	Payload interface{}           `json:"payload"`
}

type RendezvousClient struct {
	Conn *websocket.Conn
	IP   net.IP
}

type RendezvousSender struct {
	RendezvousClient
	Port int
}

type RendezvousReceiver = RendezvousClient

/* [Receiver <-> Sender] messages */

type PasswordPayload struct {
	Password string `json:"password"`
}
type PakePayload struct {
	Bytes []byte `json:"pake_bytes"`
}

type SaltPayload struct {
	Salt []byte `json:"salt"`
}

/* [Rendezvous -> Sender] messages */

type RendezvousToSenderBindPayload struct {
	ID int `json:"id"`
}

func DecodeRendezvousPayload(msg RendezvousMessage) (RendezvousMessage, error) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return RendezvousMessage{}, fmt.Errorf("unable to cast payload to map[string]interface")
	}
	switch msg.Type {
	case RendezvousToSenderBind:
		{
			id, ok := payload["id"].(int)
			if !ok {
				return RendezvousMessage{}, fmt.Errorf("unable to cast id to int")
			}
			return RendezvousMessage{Type: msg.Type, Payload: RendezvousToSenderBindPayload{ID: id}}, nil
		}
	case SenderToRendezvousEstablish, ReceiverToRendezvousEstablish:
		{
			password, ok := payload["password"].(string)
			if !ok {
				return RendezvousMessage{}, fmt.Errorf("unable to cast password to string")
			}
			return RendezvousMessage{Type: msg.Type, Payload: PasswordPayload{Password: password}}, nil
		}
	case SenderToRendezvousPAKE, RendezvousToReceiverPAKE, ReceiverToRendezvousPAKE, RendezvousToSenderPAKE:
		{
			bytes, ok := payload["pake_bytes"].([]byte)
			if !ok {
				return RendezvousMessage{}, fmt.Errorf("unable to cast pake bytes to []byte")
			}
			return RendezvousMessage{Type: msg.Type, Payload: PakePayload{Bytes: bytes}}, nil
		}
	case SenderToRendezvousSalt, RendezvousToReceiverSalt:
		{

			salt, ok := payload["pake_salt"].([]byte)
			if !ok {
				return RendezvousMessage{}, fmt.Errorf("unable to cast salt to []byte")
			}
			return RendezvousMessage{Type: msg.Type, Payload: SaltPayload{Salt: salt}}, nil
		}
	default:
		return msg, nil
	}
}

type WrongRendezvousMessageTypeError struct {
	expected []RendezvousMessageType
	got      RendezvousMessageType
}

func NewRendezvousError(expected []RendezvousMessageType, got RendezvousMessageType) *WrongRendezvousMessageTypeError {
	return &WrongRendezvousMessageTypeError{
		expected: expected,
		got:      got,
	}
}

func (e *WrongRendezvousMessageTypeError) Error() string {
	var expectedMessageTypes []string
	for _, expectedType := range e.expected {
		expectedMessageTypes = append(expectedMessageTypes, expectedType.Name())
	}
	oneOfExpected := strings.Join(expectedMessageTypes, ", ")
	return fmt.Sprintf("wrong message type, expected one of: (%s), got: (%s)", oneOfExpected, e.got.Name())
}

func (t RendezvousMessageType) Name() string {
	switch t {
	case RendezvousToSenderBind:
		return "RendezvousToSenderBind"
	case SenderToRendezvousEstablish:
		return "SenderToRendezvousEstablish"
	case ReceiverToRendezvousEstablish:
		return "ReceiverToRendezvousEstablish"
	case RendezvousToSenderReady:
		return "RendezvousToSenderReady"
	case SenderToRendezvousPAKE:
		return "SenderToRendezvousPAKE"
	case RendezvousToReceiverPAKE:
		return "RendezvousToReceiverPAKE"
	case ReceiverToRendezvousPAKE:
		return "ReceiverToRendezvousPAKE"
	case RendezvousToSenderPAKE:
		return "RendezvousToSenderPAKE"
	case SenderToRendezvousSalt:
		return "SenderToRendezvousSalt"
	case RendezvousToReceiverSalt:
		return "RendezvousToReceiverSalt"
	case ReceiverToRendezvousClose:
		return "ReceiverToRendezvousClose"
	case SenderToRendezvousClose:
		return "SenderToRendezvousClose"
	default:
		return ""
	}
}
