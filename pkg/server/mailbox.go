package server

import (
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

type Mailboxes struct{ *sync.Map }

type Mailbox struct {
	Sender   *Client
	Receiver *Client
	File     models.File
}

func (mailboxes *Mailboxes) AddMailbox(p models.Password, m *Mailbox) bool {
	// _, didNotStore := server.mailboxes.LoadOrStore(p, m)
	return true
}

func NewClient(wsConn *websocket.Conn) *Client {
	return &Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.TCPAddr).IP,
	}
}
