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

// ------------------ Conn implementations ------------------

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

// ------------------ Rendezvous Conn ------------------------

// Rendezvous specifies a connection to the rendezvous server.
type Rendezvous struct {
	Conn Conn
}

// WriteMsg writes a rendezvous message to the underlying connection.
func (r Rendezvous) WriteMsg(msg protocol.RendezvousMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.Conn.Write(payload)
}

// ReadMsg reads a rendezvous message from the underlying connection.
func (r Rendezvous) ReadMsg(expected ...protocol.RendezvousMessageType) (protocol.RendezvousMessage, error) {
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

// Write is used to write the payload to the connection.
// Implements the io.Writer interface, but at the level of messages.
func (tc Transfer) Write(payload []byte) (int, error) {
	if err := tc.write(payload); err != nil {
		return 0, nil
	}
	return len(payload), nil
}

// Read is used to read the payload from the connection.
// Implements the io.Reader interface, but at the level of messages.
// Will return a io.EOF error once it receives a SenderPayloadSent message.
func (tc Transfer) Read(buf []byte) (int, error) {
	b, err := tc.read()
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

// WriteMsg encrypts and writes the specified transfer message to the underlying connection.
func (t Transfer) WriteMsg(msg protocol.TransferMessage) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return t.write(b)
}

// ReadMsg reads and encrypts the specified transfer message to the underlying connection.
func (t Transfer) ReadMsg(expected ...protocol.TransferMessageType) (protocol.TransferMessage, error) {
	dec, err := t.read()
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

// write encrypts and writes the specified bytes to the underlying connection.
func (t Transfer) write(b []byte) error {
	enc, err := t.crypt.Encrypt(b)
	if err != nil {
		return nil
	}
	return t.Conn.Write(enc)
}

// read reads and decrypts bytes from the underlying connection.
func (t Transfer) read() ([]byte, error) {
	b, err := t.Conn.Read()
	if err != nil {
		return nil, err
	}
	return t.crypt.Decrypt(b)
}
