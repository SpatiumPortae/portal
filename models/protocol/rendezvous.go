package protocol

import (
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

type RendezvousMessageType int

const (
	ReceiverToRendezvousEstablish RendezvousMessageType = iota
	RendezvousToReceiverApprove
	SenderToRendezvousEstablish
	SenderToRendezvousReceiverRequest
	RendezvousToSenderApproveReceiver
	RendezvousToSenderGeneratedPassword
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

/* [Receiver -> Rendezvous] messages */

type ReceiverToRendezvousEstablishPayload struct {
	Password models.Password `json:"password"`
}

/* [Rendezvous -> Receiver] messages */

type RendezvousToReceiverApprovePayload struct {
	SenderIP   net.IP      `json:"senderIP"`
	SenderPort int         `json:"senderPort"`
	File       models.File `json:"File"`
}

/* [Sender -> Rendezvous] messages */

type SenderToRendezvousEstablishPayload struct {
	DesiredPort int         `json:"desiredPort"`
	File        models.File `json:"file"`
}

type SenderToRendezvousReceiverRequestPayload struct {
	ReceiverIP net.IP `json:"receiverIP"`
}

/* [Rendezvous -> Sender] messages */

type RendezvousToSenderApproveReceiverPayload struct {
	Approve    bool   `json:"approve"`
	ReceiverIP net.IP `json:"receiverIP"`
}

type RendezvousToSenderGeneratedPasswordPayload struct {
	Password models.Password `json:"password"`
}
