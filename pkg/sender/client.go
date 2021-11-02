package sender

import (
	"bufio"
	"encoding/json"
	"errors"
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
	receiverAddr net.IP
	done         chan os.Signal
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
		done:   doneCh,
		logger: logger,
		state:  Initial,
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

	state := WaitForHandShake
	updateUI(s.ui, state)

	// messaging loop (with state variables).
	for {
		msg := &protocol.TransferMessage{}
		err := wsConn.ReadJSON(msg)
		if err != nil {
			s.logger.Printf("Shutting down portal due to websocket error: %s", err)
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil
		}

		switch msg.Type {

		case protocol.ReceiverHandshake:
			if !stateInSync(wsConn, state, WaitForHandShake) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return nil
			}

			wsConn.WriteJSON(protocol.TransferMessage{
				Type: protocol.SenderHandshake,
				Payload: protocol.SenderHandshakePayload{
					PayloadSize: s.payloadSize,
				},
			})
			state = WaitForFileRequest

		case protocol.ReceiverRequestPayload:
			if !stateInSync(wsConn, state, WaitForFileRequest) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
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
				updateUI(s.ui, state, progress)
				if err == io.EOF {
					break
				}
			}
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderPayloadSent,
				Payload: "Portal transfer completed",
			})
			state = WaitForFileAck
			updateUI(s.ui, state)

		case protocol.ReceiverAckPayload:
			if !stateInSync(wsConn, state, WaitForFileAck) {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
				wsConn.Close()
				s.done <- syscall.SIGTERM
				return nil
			}
			state = WaitForCloseMessage
			wsConn.WriteJSON(protocol.TransferMessage{
				Type:    protocol.SenderClosing,
				Payload: "Closing down the Portal, as requested",
			})
			state = WaitForCloseAck

		case protocol.ReceiverClosingAck:
			if state != WaitForCloseAck {
				s.logger.Println("Shutting down portal due to unsynchronized messaging")
			}
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil

		case protocol.TransferError:
			updateUI(s.ui, state)
			s.logger.Printf("Shutting down Portal due to Alien error")
			wsConn.Close()
			s.done <- syscall.SIGTERM
			return nil
		}
	}
}

func (s *Sender) ConnectToRendezvous(passwordCh chan<- models.Password, startServerCh chan<- int, payloadReady <-chan bool) error {

	defer close(passwordCh)
	wsConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender", DEFAULT_RENDEVOUZ_ADDRESS, DEFAULT_RENDEVOUZ_PORT), nil)
	if err != nil {
		return err
	}

	msg, err := readRendevouzMessage(wsConn, protocol.RendezvousToSenderBind)
	if err != nil {
		return err
	}

	bindPayload := protocol.RendezvousToSenderBindPayload{}
	err = tools.DecodePayload(msg.Payload, &bindPayload)

	password := tools.GeneratePassword(bindPayload.ID)
	hashed := tools.HashPassword(password)

	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: hashed,
		},
	})

	passwordCh <- password
	// NOTE: This takes a couple of seconds, it is fine as we have to wait for the receiver anyway
	pake, err := pake.InitCurve([]byte(password), 0, "siec")

	msg, err = readRendevouzMessage(wsConn, protocol.RendezvousToSenderReady)
	if err != nil {
		return err
	}

	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PAKEPayload{
			PAKEBytes: pake.Bytes(),
		},
	})

	msg, err = readRendevouzMessage(wsConn, protocol.RendezvousToSenderPAKE)
	if err != nil {
		return err
	}

	pakePayload := protocol.PAKEPayload{}
	err = tools.DecodePayload(msg.Payload, &pakePayload)
	if err != nil {
		return err
	}

	err = pake.Update(pakePayload.PAKEBytes)
	if err != nil {
		return err
	}

	sessionkey, err := pake.SessionKey()
	if err != nil {
		return err
	}
	s.crypt, err = crypt.New(sessionkey)
	if err != nil {
		return err
	}

	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousSalt,
		Payload: protocol.SaltPayload{
			Salt: s.crypt.Salt,
		},
	})

	_, enc, err := wsConn.ReadMessage()

	dec, err := s.crypt.Decrypt(enc)
	if err != nil {
		return err
	}

	transferMsg := protocol.TransferMessage{}

	err = json.Unmarshal(dec, &transferMsg)
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

	s.receiverAddr = handshakePayload.IP

	senderPort, err := tools.GetOpenPort()
	if err != nil {
		return err
	}

	// wait for payload to be ready
	<-payloadReady
	startServerCh <- senderPort

	tcpAddr, ok := wsConn.LocalAddr().(*net.TCPAddr)
	if !ok {
		return errors.New("error assertion tcpAddr")
	}
	handshake := protocol.TransferMessage{
		Type: protocol.SenderHandshake,
		Payload: protocol.SenderHandshakePayload{
			IP:          tcpAddr.IP,
			Port:        senderPort,
			PayloadSize: s.payloadSize,
		},
	}
	enc, err = s.crypt.Encrypt(handshake.Bytes())
	if err != nil {
		return err
	}

	//TODO: Send wsConn over channel in case of transit communication
	//TODO: Tranist message from receiver

	wsConn.WriteMessage(websocket.BinaryMessage, enc)

	wsConn.WriteJSON(protocol.RendezvousMessage{Type: protocol.SenderToRendezvousReady})

	msg = protocol.RendezvousMessage{}
	err = wsConn.ReadJSON(&msg)
	if err != nil {
		return err
	}
	approvePayload := protocol.RendezvousToSenderApprovePayload{}
	err = tools.DecodePayload(msg.Payload, &approvePayload)

	return err
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
func updateUI(ui chan<- UIUpdate, state TransferState, progress ...float32) {
	if ui == nil {
		return
	}
	var p float32
	if len(progress) > 0 {
		p = progress[0]
	}
	ui <- UIUpdate{State: state, Progress: p}
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
