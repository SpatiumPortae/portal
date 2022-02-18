// sender.go specifies the sender client structs and options.
package sender

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type SenderOption func(*Sender)

// Sender represents the sender client, handles rendezvous communication and file transfer.
type Sender struct {
	payload           io.Reader
	payloadSize       int64
	senderServer      *Server
	closeServer       chan os.Signal
	receiverIP        net.IP
	rendezvousAddress string
	rendezvousPort    int
	ui                chan<- UIUpdate
	crypt             *crypt.Crypt
	state             TransferState
}

// New returns a bare bones Sender.
func New(rendezvousAddress string, rendezvousPort int, opts ...SenderOption) *Sender {
	closeServerCh := make(chan os.Signal, 1)
	signal.Notify(closeServerCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	s := &Sender{
		closeServer:       closeServerCh,
		rendezvousAddress: rendezvousAddress,
		rendezvousPort:    rendezvousPort,
		state:             Initial,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithPayload specifies the payload that will be transfered.
func WithPayload(payload io.Reader, payloadSize int64) SenderOption {
	return func(s *Sender) {
		s.payload = payload
		s.payloadSize = payloadSize
	}
}

// WithServer specifies the option to run the sender by hosting a server which the receiver establishes a connection to.
func WithServer(options ServerOptions) SenderOption {
	return func(s *Sender) {
		s.receiverIP = options.receiverIP
		router := &http.ServeMux{}
		s.senderServer = &Server{
			router: router,
			server: &http.Server{
				Addr:         fmt.Sprintf(":%d", options.port),
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				Handler:      router,
			},
			upgrader: websocket.Upgrader{},
		}

		// setup routes
		router.HandleFunc("/portal", s.handleTransfer())
	}
}

// WithUI specifies the option to run the sender with an UI channel that reports the state of the transfer.
func WithUI(ui chan<- UIUpdate) SenderOption {
	return func(s *Sender) {
		s.ui = ui
	}
}

func (s *Sender) RendezvousAddress() string {
	return s.rendezvousAddress
}

func (s *Sender) RendezvousPort() int {
	return s.rendezvousPort
}

// updateUI is a helper function that checks if we have a UI channel and reports the state.
func (s *Sender) updateUI(progress ...float32) {
	if s.ui == nil {
		return
	}
	var p float32
	if len(progress) > 0 {
		p = progress[0]
	}
	s.ui <- UIUpdate{State: s.state, Progress: p}
}
