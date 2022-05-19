// mailbox.go defines the central datastructure that keeps track of the different connections.
package rendezvous

import (
	"fmt"
	"sync"

	"github.com/SpatiumPortae/portal/protocol/rendezvous"
)

// Mailbox is a data structure that links together a sender and a receiver client.
type Mailbox struct {
	Sender               *rendezvous.Client
	Receiver             *rendezvous.Client
	CommunicationChannel chan []byte
	Quit                 chan bool
}

type Mailboxes struct{ *sync.Map }

// StoreMailbox allocates a mailbox.
func (mailboxes *Mailboxes) StoreMailbox(p string, m *Mailbox) {
	mailboxes.Store(p, m)
}

// GetMailbox returns the decired mailbox.
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
