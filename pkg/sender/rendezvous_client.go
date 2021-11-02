package sender

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake/v3"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/crypt"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Sender) ConnectToRendezvous(passwordCh chan<- models.Password, startServerCh chan<- ServerOptions, payloadReady <-chan bool, relayCh chan<- *websocket.Conn) error {

	// establish websocket connection to rendezvous
	wsConn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender",
		constants.DEFAULT_RENDEZVOUZ_ADDRESS, constants.DEFAULT_RENDEZVOUZ_PORT), nil)
	if err != nil {
		return err
	}

	// bind connection
	rendezvousMsg, err := tools.ReadRendevouzMessage(wsConn, protocol.RendezvousToSenderBind)
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
	// init PAKE2 (NOTE: This takes a couple of seconds, here it is fine as we have to wait for the receiver)
	pake, err := pake.InitCurve([]byte(password), 0, "p256")
	if err != nil {
		return err
	}

	// Ready to exchange crypto information.
	rendezvousMsg, err = tools.ReadRendevouzMessage(wsConn, protocol.RendezvousToSenderReady)
	if err != nil {
		return err
	}

	// PAKE sender -> receiver.
	wsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PakePayload{
			Bytes: pake.Bytes(),
		},
	})

	// PAKE receiver -> sender.
	rendezvousMsg, err = tools.ReadRendevouzMessage(wsConn, protocol.RendezvousToSenderPAKE)
	if err != nil {
		return err
	}

	pakePayload := protocol.PakePayload{}
	err = tools.DecodePayload(rendezvousMsg.Payload, &pakePayload)
	if err != nil {
		return err
	}

	err = pake.Update(pakePayload.Bytes)
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
	/* END cryptographic exchange, safe encrypted channel established! */

	transferMsg, err := tools.ReadEncryptedMessage(wsConn, s.crypt)
	if err != nil {
		return err
	}

	if transferMsg.Type != protocol.ReceiverHandshake {
		return protocol.NewWrongMessageTypeError([]protocol.TransferMessageType{protocol.ReceiverHandshake}, transferMsg.Type)
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
	tools.WriteEncryptedMessage(wsConn, handshake, s.crypt)

	transferMsg, err = tools.ReadEncryptedMessage(wsConn, s.crypt)
	if err != nil {
		return err
	}

	switch transferMsg.Type {
	case protocol.ReceiverDirectCommunication:
		close(relayCh)
		tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.SenderDirectAck}, s.crypt)
		return nil
	case protocol.ReceiverRelayCommunication:
		tools.WriteEncryptedMessage(wsConn, protocol.TransferMessage{Type: protocol.SenderRelayAck}, s.crypt)
		relayCh <- wsConn
		return nil
	default:
		return protocol.NewWrongMessageTypeError(
			[]protocol.TransferMessageType{protocol.ReceiverDirectCommunication, protocol.ReceiverRelayCommunication},
			transferMsg.Type)
	}
}
