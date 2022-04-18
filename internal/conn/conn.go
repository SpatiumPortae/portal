package conn

import (
	"encoding/json"
	"io"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models/protocol"
)

// Conn is an interface that wraps a network connection.
type Conn interface {
	Write([]byte) error
	Read() ([]byte, error)
}

// WS is a wrapper around a websocket connection.
type WS struct {
	Conn *websocket.Conn
}

func (ws *WS) Write(payload []byte) error {
	return ws.Conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (ws *WS) Read() ([]byte, error) {
	_, payload, err := ws.Conn.ReadMessage()
	return payload, err
}

// RendezvousConn specifies a connection to the rendezvous server.
type RendezvousConn struct {
	Conn Conn
}

// WriteMsg writes a rendezvous message to the underlying connection.
func (r RendezvousConn) WriteMsg(msg protocol.RendezvousMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Conn.Write(payload)
}

// ReadMsg reads a rendezvous message from the underlying connection.
func (r RendezvousConn) ReadMsg(expected ...protocol.RendezvousMessageType) (protocol.RendezvousMessage, error) {
	b, err := r.Conn.Read()
	if err != nil {
		return protocol.RendezvousMessage{}, err
	}
	var msg protocol.RendezvousMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		return protocol.RendezvousMessage{}, err
	}
	if len(expected) != 0 && expected[0] != msg.Type {
		return protocol.RendezvousMessage{}, protocol.NewRendezvousError(expected, msg.Type)
	}
	return protocol.DecodeRendezvousPayload(msg)
}

// TransferConn specifies a encrypted connection safe to transfer files over.
type TransferConn struct {
	Conn  Conn
	crypt crypt
}

func NewTransferConn(conn Conn, sessionkey, salt []byte) TransferConn {
	return TransferConn{
		Conn:  conn,
		crypt: NewCrypt(sessionkey, salt),
	}
}

// Write is used to write the payload to the connection.
// Implements the io.Writer interface, but at the level of websocket messages.
func (tc TransferConn) Write(payload []byte) (int, error) {
	if err := tc.WriteBytes(payload); err != nil {
		return 0, nil
	}
	return len(payload), nil
}

// Read is used to read the payload from the connection.
// Implements the io.Reader interface, but at the level of websocket messages.
// Will return a io.EOF error once it receives a SenderPayloadSent message.
func (tc TransferConn) Read(buf []byte) (int, error) {
	b, err := tc.ReadBytes()
	if err != nil {
		return 0, err
	}
	var msg protocol.TransferMessage
	err = json.Unmarshal(b, &msg)
	if err != nil {
		//NOTE: need to make sure that the provided buffer can read an entire message
		// Alternatively you could buffer the data in the struct.
		n := copy(buf, b)
		return n, nil
	}

	if msg.Type != protocol.SenderPayloadSent {
		return 0, protocol.NewWrongTransferMessageTypeError([]protocol.TransferMessageType{protocol.SenderPayloadSent}, msg.Type)
	}
	return 0, io.EOF
}

// WriteBytes encrypts and writes the specified bytes to the underlying connection.
func (t TransferConn) WriteBytes(b []byte) error {
	enc, err := t.crypt.Encrypt(b)
	if err != nil {
		return nil
	}
	return t.Conn.Write(enc)
}

// ReadBytes reads and decrypts bytes from the underlying connection.
func (t TransferConn) ReadBytes() ([]byte, error) {
	b, err := t.Conn.Read()
	if err != nil {
		return nil, err
	}
	return t.crypt.Decrypt(b)
}

// WriteMsg encrypts and writes the specified transfer message to the underlying connection.
func (t TransferConn) WriteMsg(msg protocol.TransferMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return t.WriteBytes(b)
}

// ReadMsg reads and encrypts the specified transfer message to the underlying connection.
func (t TransferConn) ReadMsg(expected ...protocol.TransferMessageType) (protocol.TransferMessage, error) {
	dec, err := t.ReadBytes()
	if err != nil {
		return protocol.TransferMessage{}, err
	}
	var msg protocol.TransferMessage
	if err = json.Unmarshal(dec, &msg); err != nil {
		return protocol.TransferMessage{}, err
	}

	if len(expected) != 0 && expected[0] != msg.Type {
		return protocol.TransferMessage{}, protocol.NewWrongTransferMessageTypeError(expected, msg.Type)
	}
	return protocol.DecodeTransferPayload(msg)
}
