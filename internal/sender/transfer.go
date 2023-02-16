//go:build !js

package sender

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

// doTransfer performs the file transfer, either directly or using the Rendezvous server as a relay.
// This version is built for other platforms other than js (wasm)
func doTransfer(ctx context.Context, tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(ctx, transfer.ReceiverHandshake)
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

	if err := tc.WriteMsg(ctx, transfer.Msg{
		Type: transfer.SenderHandshake,
		Payload: transfer.Payload{
			IP:          ip,
			Port:        port,
			PayloadSize: payloadSize,
		},
	}); err != nil {
		return err
	}

	msg, err := tc.ReadMsg(ctx)
	if err != nil {
		return err
	}

	switch msg.Type {
	// Direct transfer.
	case transfer.ReceiverDirectCommunication:
		if len(msgs) > 0 {
			msgs[0] <- transfer.Direct
		}
		if err := tc.WriteMsg(ctx, transfer.Msg{Type: transfer.SenderDirectAck}); err != nil {
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
		if err := tc.WriteMsg(ctx, transfer.Msg{Type: transfer.SenderRelayAck}); err != nil {
			return err
		}

		return transferSequence(ctx, tc, payload, payloadSize, msgs...)

	default:
		return transfer.Error{
			Expected: []transfer.MsgType{
				transfer.ReceiverDirectCommunication,
				transfer.ReceiverRelayCommunication},
			Got: msg.Type}
	}
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
