package server

import (
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

type Client struct {
	ws   *websocket.Conn
	IP   net.IP
	port int
}

type Mailbox struct {
	Sender   Client
	Receiver Client
	File     models.File
}
