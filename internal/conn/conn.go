package conn

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

type Conn interface {
	Write([]byte) error
	Read() ([]byte, error)
}

type WS struct {
	conn *websocket.Conn
}

func (ws *WS) Write(payload []byte) error {
	return ws.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (ws *WS) Read() ([]byte, error) {
	_, payload, err := ws.conn.ReadMessage()
	return payload, err
}

// RendezvousConn specifies a connection to the rendezvous server.
type RendezvousConn struct {
	conn Conn
}

// WriteMsg writes a rendezvous message to the underlying connection.
func (r *RendezvousConn) WriteMsg(msg protocol.RendezvousMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.conn.Write(payload)
}

// ReadMsg reads a rendezvous message from the underlying connection.
func (r *RendezvousConn) ReadMsg() (protocol.RendezvousMessage, error) {
	b, err := r.conn.Read()
	if err != nil {
		return protocol.RendezvousMessage{}, err
	}
	var msg protocol.RendezvousMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		return protocol.RendezvousMessage{}, err
	}
	return msg, nil
}

// TransferConn specifies a encrypted connection safe to transfer files over.
type TransferConn struct {
	conn  Conn
	crypt crypt
}

func NewTransferConn(conn Conn, sessionkey, salt []byte) TransferConn {
	return TransferConn{
		conn:  conn,
		crypt: NewCrypt(sessionkey, salt),
	}
}

// WriteBytes encrypts and writes the specified bytes to the underlying connection.
func (t *TransferConn) WriteBytes(b []byte) error {
	enc, err := t.crypt.Encrypt(b)
	if err != nil {
		return nil
	}
	return t.conn.Write(enc)
}

// ReadBytes reads and decrypts bytes from the underlying connection.
func (t *TransferConn) ReadBytes() ([]byte, error) {
	b, err := t.conn.Read()
	if err != nil {
		return nil, err
	}
	return t.crypt.Decrypt(b)
}

// WriteMsg encrypts and writes the specified transfer message to the underlying connection.
func (t *TransferConn) WriteMsg(msg protocol.TransferMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return t.WriteBytes(b)
}

// ReadMsg reads and encrypts the specified transfer message to the underlying connection.
func (t *TransferConn) ReadMsg() (protocol.TransferMessage, error) {
	dec, err := t.ReadBytes()
	if err != nil {
		return protocol.TransferMessage{}, err
	}
	var msg protocol.TransferMessage
	if err = json.Unmarshal(dec, &msg); err != nil {
		return protocol.TransferMessage{}, err
	}
	return msg, nil
}
