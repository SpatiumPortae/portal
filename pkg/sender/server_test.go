package sender

import (
	"bytes"
	"encoding/json"
	"log"
	"net"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
)

func TestPositiveIntegration(t *testing.T) {
	expectedPayload := []byte("Portal this shiiiiet")
	buf := bytes.NewBuffer(expectedPayload)
	logger := log.New(os.Stderr, "", log.Default().Flags())
	s, err := NewServer(8080, buf, net.ParseIP("127.0.0.1"), logger)
	if err != nil {
		t.Fail()
	}
	server := httptest.NewServer(s.handleTransfer())

	ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)
	if err != nil {
		log.Println(err)
	}
	t.Run("HandShake", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverHandshake, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderHandshake, msg.Type)
	})
	t.Run("Request", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverRequestPayload, Message: ""})
		out := &bytes.Buffer{}

		msg := &protocol.TransferMessage{}
		for {
			code, b, err := ws.ReadMessage()
			assert.NoError(t, err)
			if code != websocket.BinaryMessage {
				err = json.Unmarshal(b, msg)
				assert.NoError(t, err)
				break
			}
			out.Write(b)
		}
		assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
		assert.Equal(t, expectedPayload, out.Bytes())
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverAckPayload, Message: ""})
	})

	t.Run("Close", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverClosing, Message: ""})
		msg := &protocol.TransferMessage{}
		err := ws.ReadJSON(msg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderClosing, msg.Type)
	})
	t.Run("CloseAck", func(t *testing.T) {
		ws.WriteJSON(protocol.TransferMessage{Type: protocol.ReceiverClosingAck, Message: ""})
		_, _, err = ws.ReadMessage()
		assert.True(t, websocket.IsUnexpectedCloseError(err))
	})
}
