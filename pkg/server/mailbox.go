package server

import (
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

var mailboxes sync.Map


type Client struct {
	Conn *websocket.Conn
	IP   net.IP
	Port int
}
type Mailbox struct {
	Sender   *Client
	Receiver *Client
	File     models.File
}


func AddMailbox(p models.Password, m *Mailbox) error {
	_, didLoad :=  mailboxes.LoadOrStore(p, m)
	if !didLoad {
		return errors.New("a mailbox is already present for this password")
	}
	
	return nil
}

func NewClient(c *websocket.Conn) *Client {
	return &Client{
		Conn: c,
		IP:   c.RemoteAddr().(*net.IPAddr).IP,
	}
}

// Find availbale ports
// Send messages
// recive messages
