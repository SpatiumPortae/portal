package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
)

var wsUpgrader = websocket.Upgrader{}

func (s *Server) handleEstablishSender() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("failed to upgrade connection: ", err)
			return
		}
		defer wsConn.Close()

		// read initial sender establish request from sender
		establishMessage := models.SenderEstablishMessage{}
		err = wsConn.ReadJSON(&establishMessage)
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
