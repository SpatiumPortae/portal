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
	"github.com/schollz/pake/v3"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/crypt"
)

func TestTransfer(t *testing.T) {
	// Setup
	weak := []byte("Normie")
	expectedPayload := []byte("A frog walks into a bank...")
	buf := bytes.NewBuffer(expectedPayload)

	logger := log.New(os.Stderr, "", log.Default().Flags())
	sender := NewSender(logger)
	options := ServerOptions{receiverIP: net.ParseIP("127.0.0.1"), port: 8080}
	WithServer(sender, options)
	WithPayload(sender, buf, int64(buf.Len()))

	senderPake, _ := pake.InitCurve(weak, 0, "p256")
	receiverPake, _ := pake.InitCurve(weak, 1, "p256")
	receiverPake.Update(senderPake.Bytes())
	senderPake.Update(receiverPake.Bytes())

	senderKey, _ := senderPake.SessionKey()
	receiverKey, _ := receiverPake.SessionKey()
	sender.crypt, _ = crypt.New(senderKey)
	receiverCrypt, _ := crypt.New(receiverKey, sender.crypt.Salt)

	server := httptest.NewServer(sender.handleTransfer())
	wsConn, _, _ := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/portal", nil)

	t.Run("Request", func(t *testing.T) {
		request := protocol.TransferMessage{Type: protocol.ReceiverRequestPayload}
		writeEncryptedMessage(wsConn, request, receiverCrypt)

		out := &bytes.Buffer{}
		msg := &protocol.TransferMessage{}
		for {
			_, enc, err := wsConn.ReadMessage()
			assert.NoError(t, err)
			dec, _ := receiverCrypt.Decrypt(enc)
			err = json.Unmarshal(dec, msg)
			if err == nil {
				assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
				break
			}
			out.Write(dec)
		}
		assert.Equal(t, msg.Type, protocol.SenderPayloadSent)
		assert.Equal(t, expectedPayload, out.Bytes())
	})

	t.Run("Close", func(t *testing.T) {
		payloadAck := protocol.TransferMessage{Type: protocol.ReceiverPayloadAck}
		writeEncryptedMessage(wsConn, payloadAck, receiverCrypt)
		msg, err := readEncryptedMessage(wsConn, receiverCrypt)
		assert.NoError(t, err)
		assert.Equal(t, protocol.SenderClosing, msg.Type)
	})
	t.Run("CloseAck", func(t *testing.T) {
		closeAck := protocol.TransferMessage{Type: protocol.ReceiverClosingAck}
		writeEncryptedMessage(wsConn, closeAck, receiverCrypt)
		_, _, err := wsConn.ReadMessage()
		e, ok := err.(*websocket.CloseError)
		assert.True(t, ok)
		assert.Equal(t, e, websocket.CloseNormalClosure)
	})
}
