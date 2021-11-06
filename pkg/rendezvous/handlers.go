// handlers.go sepcifies the websocket handlers that rendezvous server uses to facilitate communcation between sender and receiver.
package rendezvous

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

// handleEstablishSender returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		// Bind an ID to this communication and send ot to the sender
		id := s.ids.Bind()
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderBind,
			Payload: protocol.RendezvousToSenderBindPayload{
				ID: id,
			},
		})

		msg := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, protocol.SenderToRendezvousEstablish) {
			return
		}

		// receive the password (hashed) from the sender.
		establishPayload := protocol.PasswordPayload{}
		err = tools.DecodePayload(msg.Payload, &establishPayload)
		if err != nil {
			log.Println("error in SenderToRendezvousEstablish payload:", err)
			return
		}

		// Allocate a mailbox for this communication.
		mailbox := &Mailbox{
			Sender: &protocol.RendezvousSender{
				RendezvousClient: *NewClient(wsConn),
			},
			CommunicationChannel: make(chan []byte),
			Quit:                 make(chan bool),
		}
		s.mailboxes.StoreMailbox(establishPayload.Password, mailbox)
		_, err = s.mailboxes.GetMailbox(establishPayload.Password)

		if err != nil {
			log.Println("The createdd mailbox could not be retrieved")
			return
		}

		// wait for receiver to connect
		timeout := time.NewTimer(RECEIVER_CONNECT_TIMEOUT)
		select {
		case <-timeout.C:
			s.ids.Delete(id)
			return
		case <-mailbox.CommunicationChannel:
			// receiver connected
			s.ids.Delete(id)
			break
		}

		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderReady,
		})

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, protocol.SenderToRendezvousPAKE) {
			return
		}

		pakePayload := protocol.PakePayload{}
		err = tools.DecodePayload(msg.Payload, &pakePayload)
		if err != nil {
			log.Println("error in SenderToRendezvousPAKE payload:", err)
			return
		}

		// send PAKE bytes to receiver
		mailbox.CommunicationChannel <- pakePayload.Bytes
		// respond with receiver PAKE bytes
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderPAKE,
			Payload: protocol.PakePayload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, protocol.SenderToRendezvousSalt) {
			return
		}

		saltPayload := protocol.SaltPayload{}
		err = tools.DecodePayload(msg.Payload, &saltPayload)
		if err != nil {
			log.Println("error in SenderToRendezvousSalt payload:", err)
			return
		}

		// Send the salt to the receiver.
		mailbox.CommunicationChannel <- saltPayload.Salt
		// Start the relay of messgaes between the sender and receiver handlers.
		startRelay(s, wsConn, mailbox, establishPayload.Password)
	}
}

// handleEstablishReceiver returns a websocket handler that that communicates with the sender.
func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		// Establish receiver.
		msg := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, protocol.ReceiverToRendezvousEstablish) {
			return
		}

		establishPayload := protocol.PasswordPayload{}
		err = tools.DecodePayload(msg.Payload, &establishPayload)
		if err != nil {
			log.Println("error in ReceiverToRendezvousEstablish payload:", err)
			return
		}

		mailbox, err := s.mailboxes.GetMailbox(establishPayload.Password)
		if err != nil {
			log.Println("failed to get mailbox:", err)
			return
		}
		if mailbox.Receiver != nil {
			log.Println("mailbox already has a receiver:", err)
			return
		}
		// this reveiver was first, reserve this mailbox for it to receive
		mailbox.Receiver = NewClient(wsConn)
		s.mailboxes.StoreMailbox(establishPayload.Password, mailbox)

		// notify sender we are connected
		mailbox.CommunicationChannel <- nil
		// send back received sender PAKE bytes
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToReceiverPAKE,
			Payload: protocol.PakePayload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, protocol.ReceiverToRendezvousPAKE) {
			return
		}

		receiverPakePayload := protocol.PakePayload{}
		err = tools.DecodePayload(msg.Payload, &receiverPakePayload)
		if err != nil {
			log.Println("error in ReceiverToRendezvousPAKE payload:", err)
			return
		}

		mailbox.CommunicationChannel <- receiverPakePayload.Bytes
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToReceiverSalt,
			Payload: protocol.SaltPayload{
				Salt: <-mailbox.CommunicationChannel,
			},
		})
		startRelay(s, wsConn, mailbox, establishPayload.Password)
	}
}

// starts the relay service, closing it on request (if i.e. clients can communicate directly)
func startRelay(s *Server, wsConn *websocket.Conn, mailbox *Mailbox, mailboxPassword string) {
	relayForwardCh := make(chan []byte)
	// listen for incoming websocket messages from currently handled client
	go func() {
		for {
			_, p, err := wsConn.ReadMessage()
			if err != nil {
				log.Println("error when listening to incoming client messages:", err)
				fmt.Printf("closed by: %s\n", wsConn.RemoteAddr())
				mailbox.Quit <- true
				return
			}
			relayForwardCh <- p
		}
	}()

	for {
		select {
		// received payload from __other client__, relay it to our currently handled client
		case relayReceivePayload := <-mailbox.CommunicationChannel:
			wsConn.WriteMessage(websocket.BinaryMessage, relayReceivePayload)

		// received payload from __currently handled__ client, relay it to other client
		case relayForwardPayload := <-relayForwardCh:
			msg := protocol.RendezvousMessage{}
			err := json.Unmarshal(relayForwardPayload, &msg)
			// failed to unmarshal, we are in (encrypted) relay-mode, forward message directly to client
			if err != nil {
				mailbox.CommunicationChannel <- relayForwardPayload
			} else {
				// close the relay service if sender requested it
				if isExpected(msg.Type, protocol.ReceiverToRendezvousClose) {
					mailbox.Quit <- true
					return
				}
			}

		// deallocate mailbox and quit
		case <-mailbox.Quit:
			s.mailboxes.Delete(mailboxPassword)
			return
		}
	}
}

// isExpected is a convience helper function that checks message types and logs errors.
func isExpected(actual protocol.RendezvousMessageType, expected protocol.RendezvousMessageType) bool {
	wasExpected := actual == expected
	if !wasExpected {
		log.Printf("Expected message of type: %d. Got type %d\n", expected, actual)
	}
	return wasExpected
}
