package protocol

import (
	"net"

	"github.com/gorilla/websocket"
)

type RendezvousMessageType int

const (
	RendezvousToSenderBind        RendezvousMessageType = iota // An ID for this connection is bound and communicated.
	SenderToRendezvousEstablish                                // Sender has generated and hashed password.
	ReceiverToRendezvousEstablish                              // Passsword has been communicated to receiver who has hashed it.
	RendezvousToSenderReady                                    // Rendezvous announces to sender that receiver is connected.
	SenderToRendezvousPAKE                                     // Sender sends PAKE information to rendezvous.
	RendezvousToReceiverPAKE                                   // Rendezvous forwards PAKE information to receiver.
	ReceiverToRendezvousPAKE                                   // Receiver sends PAKE information to rendezvous.
	RendezvousToSenderPAKE                                     // Rendezvous forwards PAKE information to receiver.
	SenderToRendezvousSalt                                     // Sender sends cryptographic salt to rendezvous.
	RendezvousToReceiverSalt                                   // Rendevoux forwards cryptographic salt to receiver
	// From this point there is a safe channel established.
	ReceiverTorendezvousClose // Receiver can connect directly to sender, close receiver connection -> close sender connection.
	SenderToRendezousClose    // Transit sequence is completed, close sender connection -> close receiver connection.
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
type PAKEPayload struct {
	PAKEBytes []byte `json:"pake_bytes"`
}

type SaltPayload struct {
	Salt []byte `json:"salt"`
}

/* [Rendezvous -> Sender] messages */

type RendezvousToSenderBindPayload struct {
	ID int `json:"id"`
}
