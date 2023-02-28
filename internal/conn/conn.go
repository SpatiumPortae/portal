package conn

import (
	"context"
	"encoding/json"
	"math"

	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"nhooyr.io/websocket"
)

// impose no message size limit
const MESSAGE_SIZE_LIMIT_BYTES = math.MaxInt64 - 1

// Conn is an interface that wraps a network connection.
type Conn interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
}

// ------------------ Conn implementations ------------------

// WS is a wrapper around a websocket connection.
type WS struct {
	Conn *websocket.Conn
}

func (ws *WS) Read(ctx context.Context) ([]byte, error) {
	// this limit is per-message and thus needs to be set before each read
	ws.Conn.SetReadLimit(MESSAGE_SIZE_LIMIT_BYTES)
	_, payload, err := ws.Conn.Read(ctx)
	return payload, err
}

func (ws *WS) Write(ctx context.Context, payload []byte) error {
	return ws.Conn.Write(ctx, websocket.MessageBinary, payload)
}

// ------------------ Rendezvous Conn ------------------------

// Rendezvous specifies a connection to the rendezvous server.
type Rendezvous struct {
	Conn Conn
}

// ReadRaw reads raw bytes from the underlying connection.
func (r Rendezvous) ReadRaw(ctx context.Context) ([]byte, error) {
	b, err := r.Conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return b, err
}

// WriteRaw writes raw bytes to the underlying connection.
func (r Rendezvous) WriteRaw(ctx context.Context, b []byte) error {
	err := r.Conn.Write(ctx, b)
	return err
}

// ReadMsg reads a rendezvous message from the underlying connection.
func (r Rendezvous) ReadMsg(ctx context.Context, expected ...rendezvous.MsgType) (rendezvous.Msg, error) {
	b, err := r.Conn.Read(ctx)
	if err != nil {
		return rendezvous.Msg{}, err
	}
	var msg rendezvous.Msg
	if err := json.Unmarshal(b, &msg); err != nil {
		return rendezvous.Msg{}, err
	}
	if len(expected) != 0 && expected[0] != msg.Type {
		return rendezvous.Msg{}, rendezvous.Error{Expected: expected, Got: msg.Type}
	}
	return msg, nil
}

// WriteMsg writes a rendezvous message to the underlying connection.
func (r Rendezvous) WriteMsg(ctx context.Context, msg rendezvous.Msg) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Conn.Write(ctx, payload)
}

// ------------------ Transfer Conn ----------------------------

// Transfer specifies a encrypted connection safe to transfer files over.
type Transfer struct {
	Conn  Conn
	crypt crypt
}

// TransferFromSession returns a secure connection using the provided session key
// and salt.
func TransferFromSession(conn Conn, sessionkey, salt []byte) Transfer {
	return Transfer{
		Conn:  conn,
		crypt: NewCrypt(sessionkey, salt),
	}
}

// TransferFromKey returns a secure connection using the provided cryptographic key.
func TransferFromKey(conn Conn, key []byte) Transfer {
	return Transfer{
		Conn:  conn,
		crypt: crypt{Key: key},
	}
}

// Key returns the cryptographic key associated with this connection.
func (tc Transfer) Key() []byte {
	return tc.crypt.Key
}

// ReadRaw reads and decrypts raw bytes from the underlying connection.
func (t Transfer) ReadRaw(ctx context.Context) ([]byte, error) {
	b, err := t.Conn.Read(ctx)
	if err != nil {
		return nil, err
	}
	return t.crypt.Decrypt(b)
}

// WriteRaw encrypts and writes the raw bytes to the underlying connection.
func (t Transfer) WriteRaw(ctx context.Context, b []byte) error {
	enc, err := t.crypt.Encrypt(b)
	if err != nil {
		return nil
	}
	return t.Conn.Write(ctx, enc)
}

// ReadMsg reads and decrypts the specified transfer message from the underlying connection.
func (t Transfer) ReadMsg(ctx context.Context, expected ...transfer.MsgType) (transfer.Msg, error) {
	dec, err := t.ReadRaw(ctx)
	if err != nil {
		return transfer.Msg{}, err
	}
	var msg transfer.Msg
	if err = json.Unmarshal(dec, &msg); err != nil {
		return transfer.Msg{}, err
	}

	if len(expected) != 0 && expected[0] != msg.Type {
		return transfer.Msg{}, transfer.Error{Expected: expected, Got: msg.Type}
	}
	return msg, nil
}

// WriteMsg encrypts and writes the specified transfer message to the underlying connection.
func (t Transfer) WriteMsg(ctx context.Context, msg transfer.Msg) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return t.WriteRaw(ctx, b)
}
