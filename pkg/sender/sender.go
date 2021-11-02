package sender

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

type Sender struct {
	payload      io.Reader
	payloadSize  int64
	senderServer *Server
	closeServer  chan os.Signal
	receiverIP   net.IP
	logger       *log.Logger
	ui           chan<- UIUpdate
	crypt        *crypt.Crypt
	state        TransferState
}

func NewSender(logger *log.Logger) *Sender {
	closeServerCh := make(chan os.Signal, 1)
	signal.Notify(closeServerCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	return &Sender{
		closeServer: closeServerCh,
		logger:      logger,
		state:       Initial,
	}
}

func WithPayload(s *Sender, payload io.Reader, payloadSize int64) *Sender {
	s.payload = payload
	s.payloadSize = payloadSize
	return s
}

// WithServer specifies the option to run the sender by hosting a server which the receiver establishes a connection to
func WithServer(s *Sender, options ServerOptions) *Sender {
	s.receiverIP = options.receiverIP
	router := &http.ServeMux{}
	s.senderServer = &Server{
		router: router,
		server: &http.Server{
			Addr:         fmt.Sprintf(":%d", options.receiverIP),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			Handler:      router,
		},
		upgrader: websocket.Upgrader{},
	}

	// setup routes
	router.HandleFunc("/portal", s.handleTransfer())
	return s
}

// WithUI specifies the option to run the sender with an UI channel that reports the state of the transfer
func WithUI(s *Sender, ui chan<- UIUpdate) *Sender {
	s.ui = ui
	return s
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
