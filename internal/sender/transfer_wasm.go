//go:build js

package sender

import (
	"io"
	"net"

	"github.com/SpatiumPortae/portal/internal/conn"
	"github.com/SpatiumPortae/portal/protocol/transfer"
)

// doTransfer preforms the file transfer directly, no relay. This function is only built for the
// js platform (wasm)
func doTransfer(tc conn.Transfer, payload io.Reader, payloadSize int64, msgs ...chan interface{}) error {
	_, err := tc.ReadMsg(transfer.ReceiverHandshake)
	if err != nil {
		return err
	}

	if err := tc.WriteMsg(transfer.Msg{
		Type: transfer.SenderHandshake,
		Payload: transfer.Payload{
			IP:          net.IP{},
			Port:        80,
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
	case transfer.ReceiverRelayCommunication:
		if err := tc.WriteMsg(transfer.Msg{Type: transfer.SenderRelayAck}); err != nil {
			return err
		}
		return transferSequence(tc, payload, payloadSize)

	default:
		return transfer.Error{
			Expected: []transfer.MsgType{
				transfer.ReceiverDirectCommunication,
				transfer.ReceiverRelayCommunication},
			Got: msg.Type}
	}
}
