package server

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

type Mailbox struct {
	Sender   *models.Sender
	Receiver *models.Receiver
	File     models.File
}

type Mailboxes struct{ *sync.Map }

func (mailboxes *Mailboxes) StoreMailbox(p models.Password, m *Mailbox) {
	s.mailboxes.Store(p, m)
}

func (mailboxes *Mailboxes) GetMailbox(p models.Password) (*Mailbox, error) {
	mailbox, ok := s.mailboxes.Load(p)
	if !ok {
		return nil, fmt.Errorf("no mailbox with password '%s'", p)
	}
	return mailbox.(*Mailbox), nil
}

func (mailboxes *Mailboxes) DeleteMailbox(p models.Password) {
	s.mailboxes.Delete(p)
}

func NewClient(wsConn *websocket.Conn) *models.Client {
	return &models.Client{
		Conn: wsConn,
		IP:   wsConn.RemoteAddr().(*net.TCPAddr).IP,
	}
}
