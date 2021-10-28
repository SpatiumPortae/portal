package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func TestIntegration(t *testing.T) {
	s := NewServer()

	mux := http.NewServeMux()

	mux.HandleFunc("/establish-sender", tools.WebsocketHandler(s.handleEstablishSender()))
	mux.HandleFunc("/establish-receiver", tools.WebsocketHandler(s.handleEstablishReceiver()))

	server := httptest.NewServer(mux)

	senderWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-sender", nil)
	senderIP := senderWsConn.LocalAddr().(*net.TCPAddr).IP
	assert.NoError(t, err)

	receiverWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-receiver", nil)
	assert.NoError(t, err)

	var generatedPassword models.Password
	fileName := "file.png"
	fileSize := 100
	desiredSenderPort := 69

	t.Run("SenderHandshake", func(t *testing.T) {
		senderWsConn.WriteJSON(&protocol.RendezvousMessage{
			Type: protocol.SenderToRendezvousEstablish,
			Payload: &protocol.SenderToRendezvousEstablishPayload{
				DesiredPort: desiredSenderPort,
				File: models.File{
					Name:  fileName,
					Bytes: int64(fileSize),
				},
			},
		})

		message := &protocol.RendezvousMessage{}
		err := senderWsConn.ReadJSON(message)
		assert.NoError(t, err)

		establishedPayload := protocol.RendezvousToSenderGeneratedPasswordPayload{}
		err = tools.DecodePayload(message.Payload, &establishedPayload)
		assert.NoError(t, err)
		assert.Equal(t, protocol.RendezvousToSenderGeneratedPassword, message.Type)
		assert.Regexp(t, regexp.MustCompile(`^\d+-[a-z]+-[a-z]+-[a-z]+$`), establishedPayload.Password)

		generatedPassword = establishedPayload.Password
	})

	t.Run("ReceiverHandshake", func(t *testing.T) {
		receiverWsConn.WriteJSON(&protocol.RendezvousMessage{
			Type: protocol.ReceiverToRendezvousEstablish,
			Payload: protocol.ReceiverToRendezvousEstablishPayload{
				Password: generatedPassword,
			},
		})

		message := &protocol.RendezvousMessage{}
		err := receiverWsConn.ReadJSON(message)
		assert.NoError(t, err)

		approvePayload := protocol.RendezvousToReceiverApprovePayload{}
		err = tools.DecodePayload(message.Payload, &approvePayload)
		assert.NoError(t, err)
		assert.Equal(t, protocol.RendezvousToReceiverApprove, message.Type)

		assert.True(t, net.IP.Equal(approvePayload.SenderIP, senderIP))
		assert.Equal(t, approvePayload.SenderPort, desiredSenderPort)
		assert.True(t, reflect.DeepEqual(approvePayload.File, models.File{
			Name:  fileName,
			Bytes: int64(fileSize),
		}))
	})

}
