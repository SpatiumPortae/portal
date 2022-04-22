package rendezvous

import (
	"fmt"
	"net"
	"strings"

	"github.com/gorilla/websocket"
)

type MsgType int

const (
	RendezvousToSenderBind        MsgType = iota // An ID for this connection is bound and communicated
	SenderToRendezvousEstablish                  // Sender has generated and hashed password
	ReceiverToRendezvousEstablish                // Passsword has been communicated to receiver who has hashed it
	RendezvousToSenderReady                      // Rendezvous announces to sender that receiver is connected
	SenderToRendezvousPAKE                       // Sender sends PAKE information to rendezvous
	RendezvousToReceiverPAKE                     // Rendezvous forwards PAKE information to receiver
	ReceiverToRendezvousPAKE                     // Receiver sends PAKE information to rendezvous
	RendezvousToSenderPAKE                       // Rendezvous forwards PAKE information to receiver
	SenderToRendezvousSalt                       // Sender sends cryptographic salt to rendezvous
	RendezvousToReceiverSalt                     // Rendevoux forwards cryptographic salt to receiver
	// From this point there is a safe channel established
	ReceiverToRendezvousClose // Receiver can connect directly to sender, close receiver connection -> close sender connection
	SenderToRendezvousClose   // Transit sequence is completed, close sender connection -> close receiver connection
)

type Msg struct {
	Type    MsgType `json:"type"`
	Payload Payload `json:"payload,omitempty"`
}

type Payload struct {
	ID       int    `json:"id,omitempty"`
	Password string `json:"password,omitempty"`
	Bytes    []byte `json:"pake_bytes,omitempty"`
	Salt     []byte `json:"salt,omitempty"`
}

type Client struct {
	Conn *websocket.Conn
	IP   net.IP
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

// NewClient returns a new client struct.
func NewClient(wsConn *websocket.Conn) *Client {
	return &Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.TCPAddr).IP,
	}
}
