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

		if !correctMessage(msg.Type, protocol.SenderToRendezvousEstablish) {
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
			Quit:                 make(chan struct{}),
		}
		s.mailboxes.StoreMailbox(establishPayload.Password, mailbox)
		_, err = s.mailboxes.GetMailbox(establishPayload.Password)

		if err != nil {
			log.Println("NotFound")
		}

		timeout := tools.NewTimeoutChannel(RECEIVER_CONNECT_TIMEOUT)
		select {
		case <-timeout:
			return
		case <-mailbox.CommunicationChannel:
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

		if !correctMessage(msg.Type, protocol.SenderToRendezvousPAKE) {
			return
		}

		pakePayload := protocol.PAKEPayload{}
		err = tools.DecodePayload(msg.Payload, &pakePayload)
		if err != nil {
			log.Println("error in SenderToRendezvousPAKE payload:", err)
			return
		}
		mailbox.CommunicationChannel <- pakePayload.PAKEBytes

		receiverPakeBytes := <-mailbox.CommunicationChannel

		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToSenderPAKE,
			Payload: protocol.PAKEPayload{
				PAKEBytes: receiverPakeBytes,
			},
		})

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !correctMessage(msg.Type, protocol.SenderToRendezvousSalt) {
			return
		}

		saltPayload := protocol.SaltPayload{}
		err = tools.DecodePayload(msg.Payload, &saltPayload)
		if err != nil {
			log.Println("error in SenderToRendezvousSalt payload:", err)
			return
		}

		mailbox.CommunicationChannel <- saltPayload.Salt

		// wait for receiver connection
		wsChan := make(chan []byte)
		defer close(wsChan)

		// listen to websocket and forward to channel
		go func() {
			for {
				_, p, _ := wsConn.ReadMessage()
				wsChan <- p
			}
		}()

		for {
			select {
			// forward payload from receiver
			case comPayload := <-mailbox.CommunicationChannel:
				wsConn.WriteMessage(websocket.BinaryMessage, comPayload)

			// check if close message: true -> close connection; false -> forward message to receiver
			case wsPayload := <-wsChan:
				msg := protocol.RendezvousMessage{}
				err := json.Unmarshal(wsPayload, &msg)
				if err != nil {
					mailbox.CommunicationChannel <- wsPayload
				} else {
					if correctMessage(msg.Type, protocol.SenderToRendezvousClose) {
						mailbox.Quit <- struct{}{}
						return
					}
				}

			// deallocate mailbox, and quit
			case <-mailbox.Quit:
				s.mailboxes.Delete(establishPayload.Password)
				return
			}
		}
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

		if !correctMessage(msg.Type, protocol.ReceiverToRendezvousEstablish) {
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

		mailbox.CommunicationChannel <- nil //notify sender we are connected.

		senderPakeBytes := <-mailbox.CommunicationChannel

		wsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.RendezvousToReceiverPAKE,
			Payload: protocol.PAKEPayload{
				PAKEBytes: senderPakeBytes,
			},
		})

		msg = protocol.RendezvousMessage{}
		err = wsConn.ReadJSON(&msg)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}

		if !correctMessage(msg.Type, protocol.ReceiverToRendezvousPAKE) {
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

		// wait for receiver connection
		wsChan := make(chan []byte)
		defer close(wsChan)

		// listen to websocket and forward to channel
		go func() {
			for {
				_, p, _ := wsConn.ReadMessage()
				wsChan <- p
			}
		}()

		for {
			select {
			// forward payload from sender
			case comPayload := <-mailbox.CommunicationChannel:
				wsConn.WriteMessage(websocket.BinaryMessage, comPayload)

			// check if close message: true -> close connection; false -> forward message to receiver
			case wsPayload := <-wsChan:
				msg := protocol.RendezvousMessage{}
				err := json.Unmarshal(wsPayload, &msg)
				if err != nil {
					mailbox.CommunicationChannel <- wsPayload
				} else {
					if correctMessage(msg.Type, protocol.SenderToRendezvousClose) {
						mailbox.Quit <- struct{}{}
						return
					}
				}

			// deallocate mailbox, and quit
			case <-mailbox.Quit:
				s.mailboxes.Delete(establishPayload.Password)
				return
			}
		}
	}
}

func correctMessage(actual protocol.RendezvousMessageType, expected protocol.RendezvousMessageType) bool {
	correct := actual == expected

	if !correct {
		log.Printf("Expected message of type: %d.  Got type %d\n", expected, actual)
	}
	return correct
}
