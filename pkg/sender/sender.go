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
	doneCh := make(chan os.Signal, 1)
	// hook up os signals to the done chanel
	signal.Notify(doneCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	return &Sender{
		closeServer: doneCh,
		logger:      logger,
		state:       Initial,
	}
}

func WithPayload(s *Sender, payload io.Reader, payloadSize int64) *Sender {
	s.payload = payload
	s.payloadSize = payloadSize
	return s
}

// WithUI specifies the option to run the sender with an UI channel that reports the state of the transfer
func WithUI(s *Sender, ui chan<- UIUpdate) *Sender {
	s.ui = ui
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
