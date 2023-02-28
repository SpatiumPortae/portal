package conn_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/stretchr/testify/assert"
)

type mockConn struct {
	conn chan []byte
}

func (m mockConn) Write(ctx context.Context, b []byte) error {
	m.conn <- b
	return nil
}

func (m mockConn) Read(ctx context.Context) ([]byte, error) {
	return <-m.conn, nil
}

func TestConn(t *testing.T) {
	c := make(chan []byte, 2)
	conn1 := mockConn{conn: c}
	conn2 := mockConn{conn: c}

	t.Run("rendezvous conn", func(t *testing.T) {
		r1 := conn.Rendezvous{Conn: conn1}
		r2 := conn.Rendezvous{Conn: conn2}

		ctx := context.Background()
		err := r1.WriteMsg(ctx, rendezvous.Msg{Type: rendezvous.SenderToRendezvousEstablish})
		assert.NoError(t, err)

		msg, err := r2.ReadMsg(ctx)
		assert.NoError(t, err)
		assert.Equal(t, msg.Type, rendezvous.SenderToRendezvousEstablish)
	})

	t.Run("transfer conn", func(t *testing.T) {
		sessionkey := []byte("sssshh... very secret secret")
		salt := make([]byte, 8)
		_, err := rand.Read(salt)
		assert.NoError(t, err)

		ctx := context.Background()
		t1 := conn.TransferFromSession(&conn1, sessionkey, salt)
		t2 := conn.TransferFromSession(&conn2, sessionkey, salt)

		err = t1.WriteMsg(ctx, transfer.Msg{Type: transfer.ReceiverHandshake})
		assert.NoError(t, err)

		msg, err := t2.ReadMsg(ctx)
		assert.NoError(t, err)
		assert.Equal(t, msg.Type, transfer.ReceiverHandshake)
	})
}
