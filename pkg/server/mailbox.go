package server

import (
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

type Client struct {
	Conn *websocket.Conn
	Addr net.Addr
	Port int
}

func NewClient(c *websocket.Conn) *Client {
	return &Client{
		Conn: c,
		Addr: c.RemoteAddr(),
	}
}

// Find availbale ports
// Send messages
// recive messages

type Mailbox struct {
	Sender   Client
	Receiver Client
	File     models.File
}
