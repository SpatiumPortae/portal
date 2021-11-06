// handlers.go sepcifies the websocket handlers that rendezvous server uses to facilitate communcation between sender and receiver.
package rendezvous

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

// handleEstablishSender returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		log.Println(1)
		// Bind a ID to this communication and send ot to the sender
		id := s.ids.Bind()
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderBind,
			Payload: protocol.RendezvousToSenderBindPayload{
				ID: id,
			},
		})

		log.Println(2)

		msg := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		log.Println(3)

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
			log.Println("NotFound")
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

		log.Println(4)

		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderReady,
		})

		log.Println(5)

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		log.Println(6)

		if !isExpected(msg.Type, protocol.SenderToRendezvousPAKE) {
			return
		}

		log.Println(7)

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

		log.Println(8)

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		log.Println(9)

		if !isExpected(msg.Type, protocol.SenderToRendezvousSalt) {
			return
		}

		log.Println(10)

		saltPayload := protocol.SaltPayload{}
		err = tools.DecodePayload(msg.Payload, &saltPayload)
		if err != nil {
			log.Println("error in SenderToRendezvousSalt payload:", err)
			return
		}

		log.Println(11)

		// Send the salt to the receiver.
		mailbox.CommunicationChannel <- saltPayload.Salt
		// Start the relay of messgaes between the sender and receiver handlers.
		log.Println(12)
		startRelay(s, wsConn, mailbox, establishPayload.Password)
	}
}

// handleEstablishReceiver returns a websocket handler that that communicates with the sender.
func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		log.Println("receive 1")

		// Establish receiver.
		msg := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		log.Println("receive 2")

		if !isExpected(msg.Type, protocol.ReceiverToRendezvousEstablish) {
			return
		}

		log.Println("receive 3")

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

		log.Println("receive 4")

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
