package rendezvous

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/schollz/pake"
	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/crypt"
	"www.github.com/ZinoKader/portal/tools"
)

func TestIntegration(t *testing.T) {
	s := NewServer()
	h := sha256.New()
	testMessage := []byte("A frog walks into a bank...")
	var passStr string
	var password []byte

	mux := http.NewServeMux()
	mux.HandleFunc("/establish-sender", tools.WebsocketHandler(s.handleEstablishSender()))
	mux.HandleFunc("/establish-receiver", tools.WebsocketHandler(s.handleEstablishReceiver()))
	server := httptest.NewServer(mux)

	senderWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-sender", nil)
	assert.NoError(t, err)

	receiverWsConn, _, err := websocket.DefaultDialer.Dial(strings.Replace(server.URL, "http", "ws", 1)+"/establish-receiver", nil)
	assert.NoError(t, err)

	t.Run("Bind", func(t *testing.T) {

		msg := protocol.RendezvousMessage{}
		err = senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, protocol.RendezvousToSenderBind))

		bindPayload := protocol.RendezvousToSenderBindPayload{}
		err = tools.DecodePayload(msg.Payload, &bindPayload)
		assert.NoError(t, err)
		passStr = fmt.Sprintf("%d-Normie", bindPayload.ID)
		h.Write([]byte(passStr))
		password = h.Sum(nil)

		senderWsConn.WriteJSON(protocol.RendezvousMessage{
			Type: protocol.SenderToRendezvousEstablish,
			Payload: &protocol.PasswordPayload{
				Password: string(password),
			},
		})
	})

	receiverWsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.ReceiverToRendezvousEstablish,
		Payload: protocol.PasswordPayload{
			Password: string(password),
		},
	})

	t.Run("RendevouzReady", func(t *testing.T) {

		msg := &protocol.RendezvousMessage{}
		err := senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, protocol.RendezvousToSenderReady))
	})

	senderPake, _ := pake.InitCurve([]byte(passStr), 0, "p256")
	receiverPake, _ := pake.InitCurve([]byte(passStr), 1, "p256")

	senderWsConn.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousPAKE,
		Payload: protocol.PAKEPayload{
			PAKEBytes: senderPake.Bytes(),
		},
	})

	t.Run("ReceiverPAKE", func(t *testing.T) {
		msg := &protocol.RendezvousMessage{}
		err := receiverWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, protocol.RendezvousToReceiverPAKE))

		pakePayload := protocol.PAKEPayload{}
		err = tools.DecodePayload(msg.Payload, &pakePayload)
		assert.NoError(t, err)
		receiverPake.Update(pakePayload.PAKEBytes)

		receiverWsConn.WriteJSON(&protocol.RendezvousMessage{
			Type: protocol.ReceiverToRendezvousPAKE,
			Payload: protocol.PAKEPayload{
				PAKEBytes: receiverPake.Bytes(),
			},
		})
	})

	t.Run("SenderPAKE", func(t *testing.T) {
		msg := &protocol.RendezvousMessage{}
		err := senderWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, protocol.RendezvousToSenderPAKE))

		pakePayload := protocol.PAKEPayload{}
		err = tools.DecodePayload(msg.Payload, &pakePayload)
		assert.NoError(t, err)
		senderPake.Update(pakePayload.PAKEBytes)

	})

	senderKey, _ := senderPake.SessionKey()
	receiverKey, _ := receiverPake.SessionKey()
	senderCrypt, _ := crypt.New(senderKey)
	receiverCrypt := &crypt.Crypt{}
	senderWsConn.WriteJSON(&protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousSalt,
		Payload: protocol.SaltPayload{
			Salt: senderCrypt.Salt,
		},
	})

	t.Run("ReceiverSalt", func(t *testing.T) {
		msg := &protocol.RendezvousMessage{}
		err := receiverWsConn.ReadJSON(&msg)
		assert.NoError(t, err)
		assert.True(t, isExpected(msg.Type, protocol.RendezvousToReceiverSalt))

		saltPayload := protocol.SaltPayload{}
		err = tools.DecodePayload(msg.Payload, &saltPayload)
		assert.NoError(t, err)
		assert.Equal(t, senderCrypt.Salt, saltPayload.Salt)
		receiverCrypt, _ = crypt.New(receiverKey, saltPayload.Salt)
	})

	enc, _ := receiverCrypt.Encrypt(testMessage)
	receiverWsConn.WriteMessage(websocket.BinaryMessage, enc)

	t.Run("SenderEncrypted", func(t *testing.T) {
		_, enc, err := senderWsConn.ReadMessage()
		assert.NoError(t, err)
		dec, err := senderCrypt.Decrypt(enc)
		assert.NoError(t, err)
		assert.Equal(t, testMessage, dec)
	})

	// TODO: test sender-receiver-matching timeout

}
