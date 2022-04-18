package conn_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"www.github.com/ZinoKader/portal/internal/conn"
	"www.github.com/ZinoKader/portal/models/protocol"
)

type mockConn struct {
	conn chan []byte
}

func (m mockConn) Write(b []byte) error {
	m.conn <- b
	return nil
}

func (m mockConn) Read() ([]byte, error) {
	return <-m.conn, nil
}

func TestConn(t *testing.T) {
	c := make(chan []byte, 2)
	conn1 := mockConn{conn: c}
	conn2 := mockConn{conn: c}

	t.Run("rendezvous conn", func(t *testing.T) {
		r1 := conn.RendezvousConn{Conn: conn1}
		r2 := conn.RendezvousConn{Conn: conn2}

		err := r1.WriteMsg(protocol.RendezvousMessage{Type: protocol.SenderToRendezvousEstablish})
		assert.NoError(t, err)

		msg, err := r2.ReadMsg()
		assert.NoError(t, err)
		assert.Equal(t, msg.Type, protocol.SenderToRendezvousEstablish)
	})

	t.Run("transfer conn", func(t *testing.T) {
		sessionkey := []byte("sssshh... very secret secret")
		salt := make([]byte, 8)
		rand.Read(salt)
		t1 := conn.TransferFromSession(&conn1, sessionkey, salt)
		t2 := conn.TransferFromSession(&conn2, sessionkey, salt)

		err := t1.WriteMsg(protocol.TransferMessage{Type: protocol.ReceiverHandshake})
		assert.NoError(t, err)

		msg, err := t2.ReadMsg()
		assert.NoError(t, err)
		assert.Equal(t, msg.Type, protocol.ReceiverHandshake)
	})
}
