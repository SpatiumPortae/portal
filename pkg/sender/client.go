package sender

import (
	"fmt"
	"net"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/tools"
)

func ConnectToRendevouz(passwordCh chan<- models.Password, senderReadyCh <-chan bool) (net.IP, error) {

	defer close(passwordCh)
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s:%s/establish-sender", models.DEAFAULT_RENDEVOUZ_ADDRESS, models.DEFAULT_RENDEVOUZ_PORT), nil)
	if err != nil {
		return nil, err
	}

	port, err := tools.GetOpenPort()

	if err != nil {
		return nil, err
	}

	ws.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousEstablish,
		Payload: protocol.SenderToRendezvousEstablishPayload{
			DesiredPort: port,
		},
	})

	msg := protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return nil, err
	}
	passwordPayload := protocol.RendezvousToSenderGeneratedPasswordPayload{}
	err = tools.DecodePayload(msg.Payload, &passwordPayload)
	if err != nil {
		return nil, err
	}

	passwordCh <- passwordPayload.Password

	// wait for file-preparations to be ready
	<-senderReadyCh
	fmt.Println("ready!")
	ws.WriteJSON(protocol.RendezvousMessage{
		Type: protocol.SenderToRendezvousReady,
	})

	//TODO: Handle payload timeouts when Zino has added that message.
	msg = protocol.RendezvousMessage{}
	err = ws.ReadJSON(&msg)
	if err != nil {
		return nil, err
	}
	approvePayload := protocol.RendezvousToSenderApprovePayload{}
	err = tools.DecodePayload(msg.Payload, &approvePayload)

	return approvePayload.ReceiverIP, err
}
