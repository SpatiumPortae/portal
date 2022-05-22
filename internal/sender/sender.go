package sender

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/internal/password"
	"github.com/SpatiumPortae/portal/protocol/rendezvous"
	"github.com/SpatiumPortae/portal/protocol/transfer"
	"github.com/gorilla/websocket"
	"github.com/schollz/pake/v3"
)

const MAX_CHUNK_BYTES = 1e6
const MAX_SEND_CHUNKS = 2e8

// ConnectRendezvous creates a connection with the rendezvous server and acquires a password associated with the connection
func ConnectRendezvous(addr net.TCPAddr) (conn.Rendezvous, string, error) {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s/establish-sender", addr.String()), nil)
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
	pake, err := pake.InitCurve([]byte(password), 0, "p256")
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
			Bytes: pake.Bytes(),
		},
	})
	if err != nil {
		return conn.Transfer{}, err
	}

	msg, err := rc.ReadMsg()
	if err != nil {
		return conn.Transfer{}, err
	}

	if err := pake.Update(msg.Payload.Bytes); err != nil {
		return conn.Transfer{}, err
	}

	// create salt and session key.
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return conn.Transfer{}, err
	}

	session, err := pake.SessionKey()
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

// Transfer preforms the file transfer, either directly or using the Rendezvous server as a relay.
func Transfer(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(transfer.ReceiverHandshake)
	if err != nil {
		return err
	}

	port, err := getOpenPort()
	if err != nil {
		return err
	}

	server := newServer(port, tc.Key(), payload, payloadSize, msgs...)
	serverDone := make(chan struct{})
	// Start server for direct transfers.
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("%v", err)
		}
		close(serverDone)
	}()
	defer server.Shutdown()

	ip, err := getLocalIP()
	if err != nil {
		return err
	}

	if err := tc.WriteMsg(transfer.Msg{
		Type: transfer.SenderHandshake,
		Payload: transfer.Payload{
			IP:          ip,
			Port:        port,
			PayloadSize: payloadSize,
		},
	}); err != nil {
		return err
	}

	msg, err := tc.ReadMsg()
	if err != nil {
		return err
	}

	switch msg.Type {
	// Direct transfer.
	case transfer.ReceiverDirectCommunication:
		if len(msgs) > 0 {
			msgs[0] <- transfer.Direct
		}
		if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderDirectAck}); err != nil {
			return err
		}

		// Wait for server to finish and return potential error that occurred in transfer handler.
		<-serverDone
		return server.Err

	// Relay transfer.
	case transfer.ReceiverRelayCommunication:
		if len(msgs) > 0 {
			msgs[0] <- transfer.Relay
		}
		if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderRelayAck}); err != nil {
			return err
		}

		return transferSequence(tc, payload, payloadSize, msgs...)

	default:
		return transfer.Error{
			Expected: []transfer.MsgType{
				transfer.ReceiverDirectCommunication,
				transfer.ReceiverRelayCommunication},
			Got: msg.Type}
	}
}

// transferSequence is a helper method that actually preforms the transfer sequence.
func transferSequence(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(transfer.ReceiverRequestPayload)
	if err != nil {
		return err
	}

	err = transferPayload(tc, payload, payloadSize, msgs...)

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
func transferPayload(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
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
		err = tc.WriteBytes(buffer[:n])
		if err != nil {
			return err
		}

		if len(msgs) > 0 {
			msgs[0] <- bytesSent
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
	// clamp amount of chunks to be at least MAX_CHUNK_BYTES
	if chunkSize <= MAX_CHUNK_BYTES {
		return MAX_CHUNK_BYTES
	}
	return chunkSize
}

func getLocalIP() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP, nil
			}
		}
	}
	return nil, fmt.Errorf("unable to resolve local IP")
}

func getOpenPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
