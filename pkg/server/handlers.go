package server

import (
	"log"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/tools"
)

func (s *Server) handleEstablishSender() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {
		// read initial sender establish request from sender
		establishMessage := models.SenderEstablishMessage{}
		err := wsConn.ReadJSON(&establishMessage)
		if err != nil {
			log.Println("failed to read/umarshal initial sender establish request message: ", err)
			return
		}

		mailbox := &Mailbox{
			Sender: &SenderClient{
				Client: *NewClient(wsConn),
				Port:   establishMessage.DesiredPort,
			},
			File: establishMessage.File,
		}

		// data-race between generating a password and adding a mailbox? maybe need a mutex after all?
		password := GeneratePassword(s.mailboxes.Map)
		s.mailboxes.AddMailbox(password, mailbox)

		// send password to sender-client
		wsConn.WriteJSON(&models.ServerGeneratedPasswordMessage{Password: password})
	}
}

func (s *Server) handleEstablishReceiver() tools.WsHandlerFunc {
	return func(wsConn *websocket.Conn) {
		// read initial sender establish request from sender
		establishMessage := models.SenderEstablishMessage{}
		err := wsConn.ReadJSON(&establishMessage)
		if err != nil {
			log.Println("failed to read/umarshal initial sender establish request message: ", err)
			return
		}
	}
}
