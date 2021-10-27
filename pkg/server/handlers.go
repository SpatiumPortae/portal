package server

import (
	"fmt"
	"log"
	"strings"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {
		var generatedPassword models.Password
		message := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&message)
		if err != nil {
			// FIXME: why is this not an error type returned by gorilla-websocket???
			if strings.Contains(err.Error(), "timeout") {
				log.Println("read deadline timed out, connection closed:", err)
			} else {
				log.Println("message did not follow protocol:", err)
			}
			return
		}

		if message.Type == protocol.SenderToRendezvousEstablish {
			establishPayload := protocol.SenderToRendezvousEstablishPayload{}
			err := tools.DecodePayload(message.Payload, &establishPayload)
			if err != nil {
				log.Println("error in SenderToRendezvousEstablish payload:", err)
				return
			}

			mailbox := &Mailbox{
				Sender: &protocol.RendezvousSender{
					RendezvousClient: *NewClient(wsConn),
					Port:             establishPayload.DesiredPort,
				},
				File:                 establishPayload.File,
				CommunicationChannel: make(chan bool),
			}
			generatedPassword = GeneratePassword(s.mailboxes.Map)
			s.mailboxes.StoreMailbox(generatedPassword, mailbox)

			wsConn.WriteJSON(&protocol.RendezvousMessage{
				Type: protocol.RendezvousToSenderGeneratedPassword,
				Payload: protocol.RendezvousToSenderGeneratedPasswordPayload{
					Password: generatedPassword,
				},
			})

			timeout := tools.NewTimeoutChannel(RECEIVER_CONNECT_TIMEOUT)

			// wait for receiver connection
			select {
			case <-mailbox.CommunicationChannel:
				wsConn.WriteJSON(&protocol.RendezvousMessage{
					Type: protocol.RendezvousToSenderApprove,
					Payload: protocol.RendezvousToSenderApprovePayload{
						ReceiverIP: mailbox.Receiver.IP,
					},
				})
			case <-timeout:
				log.Println(fmt.Sprintf("Receiver connection timed out after %s", RECEIVER_CONNECT_TIMEOUT))
				return
			}

		} else {
			log.Println(fmt.Sprintf("Expected message of type %d (SenderToRendezvousEstablish)", protocol.SenderToRendezvousEstablish))
			return
		}
	}
}

func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		message := protocol.RendezvousMessage{}
		err := wsConn.ReadJSON(&message)
		if err != nil {
			log.Println("message did not follow protocol:", err)
			return
		}
		if message.Type != protocol.ReceiverToRendezvousEstablish {
			return
		}

		establishPayload := protocol.ReceiverToRendezvousEstablishPayload{}
		err = tools.DecodePayload(message.Payload, &establishPayload)
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

		wsConn.WriteJSON(&protocol.RendezvousMessage{
			Type: protocol.RendezvousToReceiverApprove,
			Payload: &protocol.RendezvousToReceiverApprovePayload{
				SenderIP:   mailbox.Sender.IP,
				SenderPort: mailbox.Sender.Port,
				File:       mailbox.File,
			}})

		mailbox.CommunicationChannel <- true
	}
}
