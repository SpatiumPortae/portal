package rendezvous

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

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

		establishPayload := protocol.PasswordPayload{}
		err = tools.DecodePayload(msg.Payload, &establishPayload)
		if err != nil {
			log.Println("error in SenderToRendezvousEstablish payload:", err)
			return
		}

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
		timeout := tools.NewTimeoutChannel(RECEIVER_CONNECT_TIMEOUT)
		select {
		case <-timeout:
			return
		case <-mailbox.CommunicationChannel:
			// receiver connected
			break
		}
		s.ids.Delete(id)

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

		pakePayload := protocol.PAKEPayload{}
		err = tools.DecodePayload(msg.Payload, &pakePayload)
		if err != nil {
			log.Println("error in SenderToRendezvousPAKE payload:", err)
			return
		}

		// send PAKE bytes to receiver
		mailbox.CommunicationChannel <- pakePayload.PAKEBytes
		// respond with receiver PAKE bytes
		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderPAKE,
			Payload: protocol.PAKEPayload{
				PAKEBytes: <-mailbox.CommunicationChannel,
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

		mailbox.CommunicationChannel <- saltPayload.Salt
		startRelay(s, wsConn, mailbox, establishPayload.Password)
	}
}

func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

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
			Payload: protocol.PAKEPayload{
				PAKEBytes: <-mailbox.CommunicationChannel,
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

		receiverPakePayload := protocol.PAKEPayload{}
		err = tools.DecodePayload(msg.Payload, &receiverPakePayload)
		if err != nil {
			log.Println("error in ReceiverToRendezvousPAKE payload:", err)
			return
		}

		mailbox.CommunicationChannel <- receiverPakePayload.PAKEBytes
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
			_, p, _ := wsConn.ReadMessage()
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
				// close the relay  service if sender requested it
				if isExpected(msg.Type, protocol.SenderToRendezvousClose) {
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

func isExpected(actual protocol.RendezvousMessageType, expected protocol.RendezvousMessageType) bool {
	wasExpected := actual == expected
	if !wasExpected {
		log.Printf("Expected message of type: %d. Got type %d\n", expected, actual)
	}
	return wasExpected
}
