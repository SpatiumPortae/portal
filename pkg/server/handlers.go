package server

import (
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		var state SenderState = AwaitingSenderConnection
		var generatedPassword models.Password

		for {
			message := protocol.RendezvousMessage{}
			// timeout after RECEIVER_CONNECT_TIMEOUT if no receiver requests are received
			if state == AwaitingReceiverRequests {
				wsConn.SetReadDeadline(time.Now().Add(RECEIVER_CONNECT_TIMEOUT))
			}
			err := wsConn.ReadJSON(&message)
			if err != nil {
				// TODO: why is this not an error type returned by gorilla-websocket???
				if strings.Contains(err.Error(), "timeout") {
					log.Println("read deadline timed out, connection closed:", err)
				} else {
					log.Println("message did not follow protocol:", err)
				}
				return
			}

			switch message.Type {
			case protocol.SenderToRendezvousEstablish:
				if state != AwaitingSenderConnection {
					return
				}
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
					File: establishPayload.File,
				}
				generatedPassword = GeneratePassword(s.mailboxes.Map)
				s.mailboxes.StoreMailbox(generatedPassword, mailbox)

				wsConn.WriteJSON(&protocol.RendezvousMessage{
					Type: protocol.RendezvousToSenderGeneratedPassword,
					Payload: protocol.RendezvousToSenderGeneratedPasswordPayload{
						Password: generatedPassword,
					},
				})
				state = AwaitingReceiverRequests

			case protocol.SenderToRendezvousReceiverRequest:
				if state != AwaitingReceiverRequests {
					return
				}
				requestPayload := protocol.SenderToRendezvousReceiverRequestPayload{}
				err := tools.DecodePayload(message.Payload, &requestPayload)
				if err != nil {
					log.Println("error in SenderToRendezvousReceiverRequest payload:", err)
					return
				}

				mailbox, err := s.mailboxes.GetMailbox(generatedPassword)
				if err != nil {
					log.Println("failed to get mailbox:", err)
					return
				}

				shouldApproveReceiver := mailbox.Receiver.IP.Equal(requestPayload.ReceiverIP)
				wsConn.WriteJSON(&protocol.RendezvousMessage{
					Type: protocol.RendezvousToSenderApproveReceiver,
					Payload: protocol.RendezvousToSenderApproveReceiverPayload{
						Approve:    shouldApproveReceiver,
						ReceiverIP: requestPayload.ReceiverIP,
					}})
				if shouldApproveReceiver {
					s.mailboxes.DeleteMailbox(generatedPassword)
					return
				}
			}
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
	}
}
