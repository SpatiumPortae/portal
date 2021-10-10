package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/communication"
)

type Mailbox struct {
	Sender   *communication.Sender
	Receiver *communication.Receiver
	File     models.File
}

type Mailboxes struct{ *sync.Map }

func (mailboxes *Mailboxes) StoreMailbox(p models.Password, m *Mailbox) {
	mailboxes.Store(p, m)
}

func (mailboxes *Mailboxes) GetMailbox(p models.Password) (*Mailbox, error) {
	mailbox, ok := mailboxes.Load(p)
	if !ok {
		return nil, fmt.Errorf("no mailbox with password '%s'", p)
	}
	return mailbox.(*Mailbox), nil
}

func (mailboxes *Mailboxes) DeleteMailbox(p models.Password) {
	mailboxes.Delete(p)
}

func NewClient(wsConn *websocket.Conn) *communication.Client {
	return &communication.Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.TCPAddr).IP,
	}
}
