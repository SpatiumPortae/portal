package models

import (
	"net"

	"github.com/gorilla/websocket"
)

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

type ReceiverToServerEstablishMessage struct {
	Password Password `json:"password"`
}

/* [Server -> Receiver] messages */

type ServerToReceiverApproveMessage struct {
	SenderIP   net.IP `json:"senderIP"`
	SenderPort int    `json:"senderPort"`
	File       File   `json:"File"`
}

/* [Sender -> Server] messages */

type SenderToServerEstablishMessage struct {
	DesiredPort int  `json:"desiredPort"`
	File        File `json:"file"`
}

type SenderToServerReceiverRequestMessage struct {
	ReceiverIP net.IP `json:"receiverIP"`
}

/* [Server -> Sender] messages */

type ServerToSenderApproveReceiverMessage struct {
	Approve    bool   `json:"approve"`
	ReceiverIP net.IP `json:"receiverIP"`
}

type ServerToSenderGeneratedPasswordMessage struct {
	Password Password `json:"password"`
}
