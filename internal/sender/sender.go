package sender

import (
	"bufio"
	"context"
	"crypto/rand"
	crypto_rand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	math_rand "math/rand"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/schollz/pake/v3"
	"nhooyr.io/websocket"
)

const MAX_CHUNK_BYTES = 1e6
const MAX_SEND_CHUNKS = 2e8

func Init() error {
	return randomSeed()
}

// -------------------------------------------------- Sender Functions -------------------------------------------------

// ConnectRendezvous creates a connection with the rendezvous server and acquires a password associated with the connection
func ConnectRendezvous(addr string) (conn.Rendezvous, string, error) {
	ws, _, err := websocket.Dial(context.Background(), fmt.Sprintf("ws://%s/establish-sender", addr), nil)
	if err != nil {
		return conn.Rendezvous{}, "", err
	}

	rc := conn.Rendezvous{Conn: &conn.WS{Conn: ws}}

	msg, err := rc.ReadMsg(rendezvous.RendezvousToSenderBind)
	if err != nil {
		return conn.Rendezvous{}, "", err
	}
	pass := password.Generate(msg.Payload.ID)

	if err := rc.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousEstablish,
		Payload: rendezvous.Payload{
			Password: password.Hashed(pass),
		},
	}); err != nil {
		return conn.Rendezvous{}, "", err
	}
	return rc, string(pass), nil
}

// SecureConnection does the cryptographic handshake in order to resolve a secure channel to do file transfer over.
func SecureConnection(rc conn.Rendezvous, password string) (conn.Transfer, error) {
	p, err := pake.InitCurve([]byte(password), 0, "p256")
	if err != nil {
		return conn.Transfer{}, err
	}
	// Wait for for the receiver to be ready.
	_, err = rc.ReadMsg(rendezvous.RendezvousToSenderReady)
	if err != nil {
		return conn.Transfer{}, err
	}

	// Start the key exchange.
	err = rc.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousPAKE,
		Payload: rendezvous.Payload{
			Bytes: p.Bytes(),
		},
	})
	if err != nil {
		return conn.Transfer{}, err
	}

	msg, err := rc.ReadMsg()
	if err != nil {
		return conn.Transfer{}, err
	}

	if err := p.Update(msg.Payload.Bytes); err != nil {
		return conn.Transfer{}, err
	}

	// create salt and session key.
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return conn.Transfer{}, err
	}

	session, err := p.SessionKey()
	if err != nil {
		return conn.Transfer{}, err
	}

	err = rc.WriteMsg(rendezvous.Msg{
		Type: rendezvous.SenderToRendezvousSalt,
		Payload: rendezvous.Payload{
			Salt: salt,
		},
	})
	if err != nil {
		return conn.Transfer{}, err
	}

	return conn.TransferFromSession(rc.Conn, session, salt), nil
}

func TransferHandshake(tc conn.Transfer, payload io.Reader, payloadSize int64, writers ...io.Writer) (Transferer, error) {
	transferer, err := handshake(tc, payload, payloadSize, writers...)
	if err != nil {
		return nil, err
	}
	return transferer, nil
}

// ----------------------------------------------------- Transferer ----------------------------------------------------

type Transferer interface {
	Transfer() error
	Type() transfer.Type
}

type directTransferer struct {
	errC <-chan error
}

func (directTransferer) Type() transfer.Type {
	return transfer.Direct
}

func (d directTransferer) Transfer() error {
	return <-d.errC
}

type relayTransferer struct {
	tc          conn.Transfer
	payload     io.Reader
	payloadSize int64
	writers     []io.Writer
}

func (relayTransferer) Type() transfer.Type {
	return transfer.Relay
}

func (r relayTransferer) Transfer() error {
	return transferSequence(r.tc, r.payload, r.payloadSize, r.writers...)
}

// ------------------------------------------------------ Helpers ------------------------------------------------------

// transferSequence is a helper method that actually performs the transfer sequence.
func transferSequence(tc conn.Transfer, payload io.Reader, payloadSize int64, writers ...io.Writer) error {
	_, err := tc.ReadMsg(transfer.ReceiverRequestPayload)
	if err != nil {
		return err
	}

	writers = append(writers, tc)
	writer := io.MultiWriter(writers...)

	if err := transferPayload(writer, payload, payloadSize); err != nil {
		return err
	}

	if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderPayloadSent}); err != nil {
		return err
	}

	_, err = tc.ReadMsg(transfer.ReceiverPayloadAck)
	if err != nil {
		return err
	}

	if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderClosing}); err != nil {
		return err
	}
	return nil
}

// transferPayload sends the files in chunks to the sender.
func transferPayload(conn io.Writer, payload io.Reader, payloadSize int64) error {
	bufReader := bufio.NewReader(payload)
	buffer := make([]byte, chunkSize(payloadSize))
	bytesSent := 0
	for {
		n, err := bufReader.Read(buffer)
		bytesSent += n
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_, err = conn.Write(buffer[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

// chunkSize returns an appropriate chunk size for the payload size.
func chunkSize(payloadSize int64) int64 {
	// clamp amount of chunks to be at most MAX_SEND_CHUNKS if it exceeds
	if payloadSize/MAX_CHUNK_BYTES > MAX_SEND_CHUNKS {
		return int64(payloadSize) / MAX_SEND_CHUNKS
	}
	// if not exceeding MAX_SEND_CHUNKS, divide up no. of chunks to MAX_CHUNK_BYTES-sized chunks
	chunkSize := int64(payloadSize) / MAX_CHUNK_BYTES
	// clamp amount of chunks to be at most MAX_CHUNK_BYTES
	if chunkSize <= MAX_CHUNK_BYTES {
		return MAX_CHUNK_BYTES
	}
	return chunkSize
}

func randomSeed() error {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
		return err
	}
	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
	return nil
}
