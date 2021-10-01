package models

import (
	"net"

	"github.com/gorilla/websocket"
)

type Client struct {
	ws   *websocket.Conn
	IP   net.IP
	port int
}

type Mailbox struct {
	Sender   Client
	Receiver Client
	File     File
}
