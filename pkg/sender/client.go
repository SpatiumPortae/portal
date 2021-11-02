package sender

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake/v3"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/crypt"
	"www.github.com/ZinoKader/portal/tools"
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

func (s *Sender) Transfer(wsConn *websocket.Conn) error {

	if s.ui != nil {
		defer close(s.ui)
	}

	s.state = WaitForHandShake
	s.updateUI()

	// messaging loop (with state variables)
	for {
		msg := &protocol.TransferMessage{}
		err := wsConn.ReadJSON(msg)
		if err != nil {
			s.logger.Printf("Shutting down portal due to websocket error: %s", err)
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil
		}

		switch msg.Type {

		case protocol.ReceiverHandshake:
			if !stateInSync(wsConn, s.state, WaitForHandShake) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}

			wsConn.WriteJSON(protocol.TransferMessage{
				Type: protocol.SenderHandshake,
				Payload: protocol.SenderHandshakePayload{
					PayloadSize: s.payloadSize,
				},
			})
			s.state = WaitForFileRequest

		case protocol.ReceiverRequestPayload:
			if !stateInSync(wsConn, s.state, WaitForFileRequest) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}
			buffered := bufio.NewReader(s.payload)
			chunkSize := getChunkSize(s.payloadSize)
			b := make([]byte, chunkSize)
			var bytesSent int
			for {
				n, err := buffered.Read(b)
				bytesSent += n
				wsConn.WriteMessage(websocket.BinaryMessage, b[:n]) //TODO: handle error?
				progress := float32(bytesSent) / float32(s.payloadSize)
				s.updateUI(progress)
				if err == io.EOF {
					break
				}
			}
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderPayloadSent,
				Payload: "Portal transfer completed",
			})
			s.state = WaitForFileAck
			s.updateUI()

		case protocol.ReceiverPayloadAck:
			if !stateInSync(wsConn, s.state, WaitForFileAck) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.closeServer <- syscall.SIGTERM
				return nil
			}
			s.state = WaitForCloseMessage
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderClosing,
				Payload: "Closing down the Portal, as requested",
			})
			s.state = WaitForCloseAck

		case protocol.ReceiverClosingAck:
			if s.state != WaitForCloseAck {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
			}
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil

		case protocol.TransferError:
			s.updateUI()
			s.logger.Printf("Shutting down Portal due to Alien error")
			wsConn.Close()
			s.closeServer <- syscall.SIGTERM
			return nil
		}
	}
}

func (s *Sender) ConnectToRendezvous(passwordCh chan<- models.Password, startServerCh chan<- ServerOptions, payloadReady <-chan bool, transitCh chan<- *websocket.Conn) error {

	// establish websocket connection to rendezvous
	wsConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender", DEFAULT_RENDEVOUZ_ADDRESS, DEFAULT_RENDEVOUZ_PORT), nil)
	if err != nil {
		return err
	}

	// Bind connection
	rendezvousMsg, err := readRendevouzMessage(wsConn, protocol.RendezvousToSenderBind)
	if err != nil {
		return err
	}

	bindPayload := protocol.RendezvousToSenderBindPayload{}
	err = tools.DecodePayload(rendezvousMsg.Payload, &bindPayload)
	if err != nil {
		return err
	}

	// Establish sender
	password := tools.GeneratePassword(bindPayload.ID)
	hashed := tools.HashPassword(password)

	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: hashed,
		},
	})

	// send the generated password to the UI so it can be displayed.
	passwordCh <- password

	/* START cryptographic exchange */
	// Init PAKE2 (NOTE: This takes a couple of seconds, here it is fine as we have to wait for the receiver)
	pake, err := pake.InitCurve([]byte(password), 0, "siec")
	if err != nil {
		return err
	}

	// Ready to exchange crypto information.
	rendezvousMsg, err = readRendevouzMessage(wsConn, protocol.RendezvousToSenderReady)
	if err != nil {
		return err
	}

	// PAKE sender -> receiver.
	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PAKEPayload{
			PAKEBytes: pake.Bytes(),
		},
	})

	// PAKE receiver -> sender.
	rendezvousMsg, err = readRendevouzMessage(wsConn, protocol.RendezvousToSenderPAKE)
	if err != nil {
		return err
	}

	pakePayload := protocol.PAKEPayload{}
	err = tools.DecodePayload(rendezvousMsg.Payload, &pakePayload)
	if err != nil {
		return err
	}

	err = pake.Update(pakePayload.PAKEBytes)
	if err != nil {
		return err
	}

	// Setup crypt.Crypt struct in Sender.
	sessionkey, err := pake.SessionKey()
	if err != nil {
		return err
	}
	s.crypt, err = crypt.New(sessionkey)
	if err != nil {
		return err
	}

	// Send salt to receiver.
	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousSalt,
		Payload: protocol.SaltPayload{
			Salt: s.crypt.Salt,
		},
	})
	/* END cryptographic exchange, safe Encrypted channel established! */

	transferMsg, err := readEncryptedMessage(wsConn, s.crypt)
	if err != nil {
		return err
	}

	if transferMsg.Type != protocol.ReceiverHandshake {
		return protocol.NewWrongMessageTypeError(protocol.ReceiverHandshake, transferMsg.Type)
	}

	handshakePayload := protocol.ReceiverHandshakePayload{}
	err = tools.DecodePayload(transferMsg.Payload, &handshakePayload)
	if err != nil {
		return err
	}

	senderPort, err := tools.GetOpenPort()
	if err != nil {
		return err
	}

	// wait for payload to be ready
	<-payloadReady
	startServerCh <- ServerOptions{port: senderPort, receiverIP: handshakePayload.IP}

	tcpAddr, _ := wsConn.LocalAddr().(*net.TCPAddr)

	handshake := protocol.TransferMessage{
		Type: protocol.SenderHandshake,
		Payload: protocol.SenderHandshakePayload{
			IP:          tcpAddr.IP,
			Port:        senderPort,
			PayloadSize: s.payloadSize,
		},
	}
	writeEncryptedMessage(wsConn, handshake, s.crypt)
	transferMsg, err = readEncryptedMessage(wsConn, s.crypt)

	if err != nil {
		if e, ok := err.(*websocket.CloseError); !ok || e.Code != websocket.CloseNormalClosure {
			return err
		}
		// if websocket was closed, but __not__ due to an error, rather due to direct communication
		// from this point on, we can close the rendezvous server
		close(transitCh)
	}

	if transferMsg.Type != protocol.ReceiverTransit {
		return protocol.NewWrongMessageTypeError(protocol.ReceiverTransit, transferMsg.Type)
	}

	writeEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.SenderTransitAck}, s.crypt)
	transitCh <- wsConn
	return nil
}

// stateInSync is a helper that checks the states line up, and reports errors to the receiver in case the states are out of sync
func stateInSync(wsConn *websocket.Conn, state, expected TransferState) bool {
	synced := state == expected
	if !synced {
		wsConn.WriteJSON(protocol.TransferMessage{
			Type:    protocol.TransferError,
			Payload: "Portal unsynchronized, shutting down",
		})
	}
	return synced
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

// getChunkSize returns an appropriate chunk size for the payload size
func getChunkSize(payloadSize int64) int64 {
	// clamp amount of chunks to be at most MAX_SEND_CHUNKS if it exceeds
	if payloadSize/MAX_CHUNK_BYTES > MAX_SEND_CHUNKS {
		return int64(payloadSize) / MAX_SEND_CHUNKS
	}
	// if not exceeding MAX_SEND_CHUNKS, divide up no. of chunks to MAX_CHUNK_BYTES-sized chunks
	chunkSize := int64(payloadSize) / MAX_CHUNK_BYTES
	// clamp amount of chunks to be at least MAX_CHUNK_BYTES
	if chunkSize <= MAX_CHUNK_BYTES {
		return MAX_CHUNK_BYTES
	}
	return chunkSize
}

func readRendevouzMessage(wsConn *websocket.Conn, expected protocol.RendezvousMessageType) (protocol.RendezvousMessage, error) {
	msg := protocol.RendezvousMessage{}
	err := wsConn.ReadJSON(&msg)
	if err != nil {
		return protocol.RendezvousMessage{}, err
	}

	if msg.Type != expected {
		return protocol.RendezvousMessage{}, fmt.Errorf("expected message type: %d. Got type:%d", expected, msg.Type)
	}
	return msg, nil
}

func writeEncryptedMessage(wsConn *websocket.Conn, msg protocol.TransferMessage, crypt *crypt.Crypt) error {
	enc, err := crypt.Encrypt(msg.Bytes())
	if err != nil {
		return err
	}
	wsConn.WriteMessage(websocket.BinaryMessage, enc)
	return nil
}

func readEncryptedMessage(wsConn *websocket.Conn, crypt *crypt.Crypt) (protocol.TransferMessage, error) {
	_, enc, err := wsConn.ReadMessage()
	if err != nil {
		return protocol.TransferMessage{}, err
	}

	dec, err := crypt.Decrypt(enc)
	if err != nil {
		return protocol.TransferMessage{}, err
	}

	msg := protocol.TransferMessage{}
	err = json.Unmarshal(dec, &msg)
	if err != nil {
		return protocol.TransferMessage{}, err
	}
	return msg, nil
}
