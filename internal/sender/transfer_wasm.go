//go:build js

package sender

import (
	"io"
	"net"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

func handshake(tc conn.Transfer, payload io.Reader, payloadSize int64, writers ...io.Writer) (Transferer, error) {
	_, err := tc.ReadMsg(transfer.ReceiverHandshake)
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
