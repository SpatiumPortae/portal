package tools

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WsHandlerFunc func(*websocket.Conn)

func WebsocketHandler(wsHandler WsHandlerFunc) http.HandlerFunc {
	wsUpgrader := websocket.Upgrader{}
	return func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("failed to upgrade connection: ", err)
			return
		}
		defer wsConn.Close()
		wsHandler(wsConn)
	}
}
