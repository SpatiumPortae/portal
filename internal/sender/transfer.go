//go:build !js

package sender

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

// handshake performs the file transfer, either directly or using the Rendezvous server as a relay.
// This version is built for other platforms other than js (wasm)
func handshake(tc conn.Transfer, payload io.Reader, payloadSize int64, writers ...io.Writer) (Transferer, error) {
	_, err := tc.ReadMsg(transfer.ReceiverHandshake)
	if err != nil {
		return nil, err
	}
	port, err := getOpenPort()
	if err != nil {
		return nil, err
	}
	serverErr := make(chan error)
	server := newServer(port, tc.Key(), payload, payloadSize, serverErr, writers...)
	// Start server for direct transfers.
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("%v", err)
		}
		close(serverErr)
	}()

	ip, err := getLocalIP()
	if err != nil {
		return nil, err
	}

	if err := tc.WriteMsg(transfer.Msg{
		Type: transfer.SenderHandshake,
		Payload: transfer.Payload{
			IP:          ip,
			Port:        port,
			PayloadSize: payloadSize,
		},
	}); err != nil {
		return nil, err
	}

	msg, err := tc.ReadMsg()
	if err != nil {
		return nil, err
	}

	switch msg.Type {
	// Direct transfer.
	case transfer.ReceiverDirectCommunication:
		if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderDirectAck}); err != nil {
			return nil, err
		}

		// Wait for server to finish and return potential error that occurred in transfer handler.

		return directTransferer{errC: serverErr}, nil

	// Relay transfer.
	case transfer.ReceiverRelayCommunication:
		if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderRelayAck}); err != nil {
			return nil, err
		}

		return relayTransferer{
			tc:          tc,
			payload:     payload,
			payloadSize: payloadSize,
			writers:     writers,
		}, nil

	default:
		return nil, transfer.Error{
			Expected: []transfer.MsgType{
				transfer.ReceiverDirectCommunication,
				transfer.ReceiverRelayCommunication},
			Got: msg.Type}
	}
}

// ------------------------------------------------------ Helpers ------------------------------------------------------

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
