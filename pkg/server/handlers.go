package server

import (
	"fmt"
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

		// read initial send request from sender
		f := models.File{}
		err = wsConn.ReadJSON(&f)
		if err != nil {
			log.Println("failed to read initial send request message: ", err)
			return
		}

		mailbox := &Mailbox{
			Sender: NewClient(wsConn),
			File:   f,
		}

		password := GeneratePassword(s.mailboxes.Map)
		s.mailboxes.AddMailbox(password, mailbox)

		// TODO: Remove, just for debug
		fmt.Println(password)
	}
}
