// mailbox.go defines the central datastructure that keeps track of the different connections.
package rendezvous

import (
	"fmt"
	"sync"
)

// Mailbox is a data structure that links together a sender and a receiver client.
type Mailbox struct {
	hasReceiver bool

	Receiver chan []byte // messages to Receiver
	Sender   chan []byte // messages to Sender
}

type Mailboxes struct{ *sync.Map }

// StoreMailbox allocates a mailbox.
func (mailboxes *Mailboxes) StoreMailbox(p string, m *Mailbox) {
	mailboxes.Store(p, m)
}

// GetMailbox returns the desired mailbox.
func (mailboxes *Mailboxes) GetMailbox(p string) (*Mailbox, error) {
	mailbox, ok := mailboxes.Load(p)
	if !ok {
		return nil, fmt.Errorf("no mailbox with password '%s'", p)
	}
	return mailbox.(*Mailbox), nil
}

// DeleteMailbox deallocates a mailbox.
func (mailboxes *Mailboxes) DeleteMailbox(p string) {
	mailboxes.Delete(p)
}
