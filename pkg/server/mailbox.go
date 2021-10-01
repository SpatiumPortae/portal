package server

import (
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)
type Client struct {
	Conn *websocket.Conn
	IP   net.IP
	Port int
}

type Mailboxes struct { *sync.Map }

type Mailbox struct {
	Sender   *Client
	Receiver *Client
	File     models.File
}

func (mailboxes *Mailboxes) Bla() error {
	return nil
}

func (mailboxes *Mailboxes) AddMailbox(p models.Password, m *Mailbox) error {
	_, didLoad :=  server.mailboxes.LoadOrStore(p, m)
	if !didLoad {
		return errors.New("a mailbox is already present for this password")
	}
	
	return nil
}

func NewClient(wsConn *websocket.Conn) *Client {
	return &Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.IPAddr).IP,
	}
}