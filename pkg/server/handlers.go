package server

import (
	"log"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/communication"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {

		state := AwaitingSender
		var generatedPassword models.Password

		for {
			message := communication.EstablishMessage{}
			// timeout after RECEIVER_CONNECT_TIMEOUT if no receiver requests are received
			if state == AwaitingReceiverRequests {
				wsConn.SetReadDeadline(time.Now().Add(RECEIVER_CONNECT_TIMEOUT))
			}
			err := wsConn.ReadJSON(&message)
			if err != nil {
				log.Println("message did not follow protocol: ", err)
				return
			}

			switch message.Type {
			case communication.SenderToServerEstablish:
				if state != AwaitingSender {
					return
				}
				establishPayload := communication.SenderToServerEstablishPayload{}
				err := tools.DecodePayload(message.Payload, &establishPayload)
				if err != nil {
					log.Println("faulty SenderToServerEstablish payload: ", err)
					return
				}

				mailbox := &Mailbox{
					Sender: communication.Sender{
						Client: *NewClient(wsConn),
						Port:   establishPayload.DesiredPort,
					},
					File: establishPayload.File,
				}
				generatedPassword = GeneratePassword(s.mailboxes.Map)
				s.mailboxes.StoreMailbox(generatedPassword, mailbox)

				wsConn.WriteJSON(&communication.EstablishMessage{
					Type: communication.ServerToSenderGeneratedPassword,
					Payload: communication.ServerToSenderGeneratedPasswordPayload{
						Password: generatedPassword,
					},
				})
				state = AwaitingReceiverRequests

			case communication.SenderToServerReceiverRequest:
				if state != AwaitingReceiverRequests {
					return
				}
				requestPayload := communication.SenderToServerReceiverRequestPayload{}
				err := tools.DecodePayload(message.Payload, &requestPayload)
				if err != nil {
					log.Println("faulty SenderToServerReceiverRequest payload: ", err)
					return
				}

				mailbox, err := s.mailboxes.GetMailbox(generatedPassword)
				if err != nil {
					log.Println("failed to get mailbox: ", err)
					return
				}

				shouldApproveReceiver := mailbox.Receiver.IP.Equal(requestPayload.ReceiverIP)
				wsConn.WriteJSON(&communication.ServerToSenderApproveReceiverPayload{
					Approve:    shouldApproveReceiver,
					ReceiverIP: requestPayload.ReceiverIP,
				})
				if shouldApproveReceiver {
					s.mailboxes.DeleteMailbox(generatedPassword)
					break
				}
			}
		}
	}
}

func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {
		establishMessage := models.ReceiverToServerEstablishMessage{}
		err := wsConn.ReadJSON(&establishMessage)
		if err != nil {
			log.Println("failed to read/umarshal initial sender establish request message: ", err)
			return
		}

		mailbox, err := s.mailboxes.GetMailbox(establishMessage.Password)
		if err != nil {
			log.Println("failed to get mailbox: ", err)
			return
		}
		if mailbox.Receiver != nil {
			log.Println("mailbox already has a receiver: ", err)
			return
		}
		// we were first, reserve this mailbox for us to receive
		mailbox.Receiver = NewClient(wsConn)
		s.mailboxes.StoreMailbox(establishMessage.Password, mailbox)

		wsConn.WriteJSON(&models.ServerToReceiverApproveMessage{
			SenderIP:   mailbox.Sender.IP,
			SenderPort: mailbox.Sender.Port,
			File:       mailbox.File,
		})
	}
}
