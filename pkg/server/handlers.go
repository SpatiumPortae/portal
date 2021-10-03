package server

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {
		establishMessage := models.SenderToServerEstablishMessage{}
		err := wsConn.ReadJSON(&establishMessage)
		if err != nil {
			log.Println("failed to read/umarshal initial sender establish request message: ", err)
			return
		}

		mailbox := &Mailbox{
			Sender: &models.Sender{
				Client: *NewClient(wsConn),
				Port:   establishMessage.DesiredPort,
			},
			File: establishMessage.File,
		}

		// data-race between generating a password and adding a mailbox? maybe need a mutex after all?
		password := GeneratePassword(s.mailboxes.Map)
		s.mailboxes.StoreMailbox(password, mailbox)
		wsConn.WriteJSON(&models.ServerToSenderGeneratedPasswordMessage{Password: password})

		for {
			receiverRequestMessage := models.SenderToServerReceiverRequestMessage{}
			// wait for "receiver-connected" message from sender
			wsConn.SetReadDeadline(time.Now().Add(RECEIVER_CONNECT_TIMEOUT))
			err = wsConn.ReadJSON(&receiverRequestMessage)
			if err != nil {
				log.Println(fmt.Sprintf("no receiver connected before timeout (%s): ", RECEIVER_CONNECT_TIMEOUT), err)
				return
			}

			mailbox, err = s.mailboxes.GetMailbox(password)
			if err != nil {
				log.Println("failed to get mailbox: ", err)
				return
			}

			shouldApproveReceiver := mailbox.Receiver.IP.Equal(receiverRequestMessage.ReceiverIP)
			wsConn.WriteJSON(&models.ServerToSenderApproveReceiverMessage{
				Approve:    shouldApproveReceiver,
				ReceiverIP: receiverRequestMessage.ReceiverIP,
			})
			if shouldApproveReceiver {
				s.mailboxes.DeleteMailbox(password)
				break
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
