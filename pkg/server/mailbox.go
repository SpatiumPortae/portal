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
}

type SenderClient struct {
	Client
	Port int
}

type Mailboxes struct{ *sync.Map }

type Mailbox struct {
	Sender   *SenderClient
	Receiver *Client
	File     models.File
}

func (mailboxes *Mailboxes) AddMailbox(p models.Password, m *Mailbox) {
	server.mailboxes.Store(p, m)
}

func NewClient(wsConn *websocket.Conn) *Client {
	return &Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.TCPAddr).IP,
	}
}
