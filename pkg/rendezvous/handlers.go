// handlers.go sepcifies the websocket handlers that rendezvous server uses to facilitate communcation between sender and receiver.
package rendezvous

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/protocol/rendezvous"
	"www.github.com/ZinoKader/portal/tools"
)

// handleEstablishSender returns a websocket handler that communicates with the sender.
func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		// Bind an ID to this communication and send ot to the sender
		id := s.ids.Bind()
		wsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderBind,
			Payload: rendezvous.Payload{
				ID: id,
			},
		})

		msg := rendezvous.Msg{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, rendezvous.SenderToRendezvousEstablish) {
			return
		}

		// Allocate a mailbox for this communication.
		mailbox := &Mailbox{
			Sender:               rendezvous.NewClient(wsConn),
			CommunicationChannel: make(chan []byte),
			Quit:                 make(chan bool),
		}
		s.mailboxes.StoreMailbox(msg.Payload.Password, mailbox)
		_, err = s.mailboxes.GetMailbox(msg.Payload.Password)

		password := msg.Payload.Password

		if err != nil {
			log.Println("The created mailbox could not be retrieved")
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

		wsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderReady,
		})

		msg = rendezvous.Msg{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, rendezvous.SenderToRendezvousPAKE) {
			return
		}

		// send PAKE bytes to receiver
		mailbox.CommunicationChannel <- msg.Payload.Bytes
		// respond with receiver PAKE bytes
		wsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.RendezvousToSenderPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})

		msg = rendezvous.Msg{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, rendezvous.SenderToRendezvousSalt) {
			return
		}

		// Send the salt to the receiver.
		mailbox.CommunicationChannel <- msg.Payload.Salt
		// Start the relay of messgaes between the sender and receiver handlers.
		startRelay(s, wsConn, mailbox, password)
	}
}

// handleEstablishReceiver returns a websocket handler that that communicates with the sender.
func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		// Establish receiver.
		msg := rendezvous.Msg{}
		err := wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, rendezvous.ReceiverToRendezvousEstablish) {
			return
		}

		mailbox, err := s.mailboxes.GetMailbox(msg.Payload.Password)
		if err != nil {
			log.Println("failed to get mailbox:", err)
			return
		}
		if mailbox.Receiver != nil {
			log.Println("mailbox already has a receiver:", err)
			return
		}
		// this reveiver was first, reserve this mailbox for it to receive
		mailbox.Receiver = rendezvous.NewClient(wsConn)
		s.mailboxes.StoreMailbox(msg.Payload.Password, mailbox)
		password := msg.Payload.Password

		// notify sender we are connected
		mailbox.CommunicationChannel <- nil
		// send back received sender PAKE bytes
		wsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverPAKE,
			Payload: rendezvous.Payload{
				Bytes: <-mailbox.CommunicationChannel,
			},
		})

		msg = rendezvous.Msg{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !isExpected(msg.Type, rendezvous.ReceiverToRendezvousPAKE) {
			return
		}

		mailbox.CommunicationChannel <- msg.Payload.Bytes
		wsConn.WriteJSON(rendezvous.Msg{
			Type: rendezvous.RendezvousToReceiverSalt,
			Payload: rendezvous.Payload{
				Salt: <-mailbox.CommunicationChannel,
			},
		})
		startRelay(s, wsConn, mailbox, password)
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
			msg := rendezvous.Msg{}
			err := json.Unmarshal(relayForwardPayload, &msg)
			// failed to unmarshal, we are in (encrypted) relay-mode, forward message directly to client
			if err != nil {
				mailbox.CommunicationChannel <- relayForwardPayload
			} else {
				// close the relay service if sender requested it
				if isExpected(msg.Type, rendezvous.ReceiverToRendezvousClose) {
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
func isExpected(actual rendezvous.MsgType, expected rendezvous.MsgType) bool {
	wasExpected := actual == expected
	if !wasExpected {
		log.Printf("Expected message of type: %d. Got type %d\n", expected, actual)
	}
	return wasExpected
}
