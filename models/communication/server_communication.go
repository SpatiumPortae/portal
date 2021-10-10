package communication

import (
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

type EstablishMessageType int

const (
	ReceiverToServerEstablish EstablishMessageType = iota
	ServerToReceiverApprove
	SenderToServerEstablish
	SenderToServerReceiverRequest
	ServerToSenderApproveReceiver
	ServerToSenderGeneratedPassword
)

type EstablishMessage struct {
	Type    EstablishMessageType `json:"type"`
	Payload interface{}          `json:"payload"`
}

type Client struct {
	Conn *websocket.Conn
	IP   net.IP
}

type Sender struct {
	Client
	Port int
}

type Receiver = Client

/* [Receiver -> Server] messages */

type ReceiverToServerEstablishPayload struct {
	Password models.Password `json:"password"`
}

/* [Server -> Receiver] messages */

type ServerToReceiverApprovePayload struct {
	SenderIP   net.IP      `json:"senderIP"`
	SenderPort int         `json:"senderPort"`
	File       models.File `json:"File"`
}

/* [Sender -> Server] messages */

type SenderToServerEstablishPayload struct {
	DesiredPort int         `json:"desiredPort"`
	File        models.File `json:"file"`
}

type SenderToServerReceiverRequestPayload struct {
	ReceiverIP net.IP `json:"receiverIP"`
}

/* [Server -> Sender] messages */

type ServerToSenderApproveReceiverPayload struct {
	Approve    bool   `json:"approve"`
	ReceiverIP net.IP `json:"receiverIP"`
}

type ServerToSenderGeneratedPasswordPayload struct {
	Password models.Password `json:"password"`
}
