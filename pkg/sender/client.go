package sender

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func ConnectToRendevouz(passwordCh chan<- models.Password, senderReadyCh <-chan bool) (int, net.IP, error) {

	defer close(passwordCh)
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender", DEFAULT_RENDEVOUZ_ADDRESS, DEFAULT_RENDEVOUZ_PORT), nil)
	if err != nil {
		return 0, nil, err
	}

	senderPort, err := tools.GetOpenPort()
	if err != nil {
		return 0, nil, err
	}

	ws.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.SenderToRendezvousEstablishPayload{
			DesiredPort: senderPort,
		},
	})

	msg := protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return 0, nil, err
	}
	passwordPayload := protocol.RendezvousToSenderGeneratedPasswordPayload{}
	err = tools.DecodePayload(msg.Payload, &passwordPayload)
	if err != nil {
		return 0, nil, err
	}

	passwordCh <- passwordPayload.Password

	// wait for file-preparations to be ready
	<-senderReadyCh

	ws.WriteJSON(protocol.RendezvousMessage{Type: protocol.SenderToRendezvousReady})

	//TODO: Handle payload timeouts when Zino has added that message.
	msg = protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return 0, nil, err
	}
	approvePayload := protocol.RendezvousToSenderApprovePayload{}
	err = tools.DecodePayload(msg.Payload, &approvePayload)

	return senderPort, approvePayload.ReceiverIP, err
}
