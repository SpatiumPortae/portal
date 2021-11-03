package main

import (
	"fmt"
	"os"

	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/receiver"
	"www.github.com/ZinoKader/portal/tools"
	receiverui "www.github.com/ZinoKader/portal/ui/receiver"
)

func handleReceiveCommand(password string) {
	// communicate ui updates on this channel between receiverClient and handleReceiveCmmand
	uiCh := make(chan receiver.UIUpdate)
	// initialize a receiverClient with a UI
	receiverClient := receiver.WithUI(receiver.NewReceiver(), uiCh)
	// initialize and start sender-UI
	receiverUI := receiverui.NewReceiverUI()

	go func() {
		if err := receiverUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
		}
		os.Exit(0)
	}()

	parsedPassword, err := tools.ParsePassword(password)
	if err != nil {
		fmt.Println(err)
		return // TODO: be a better person
	}

	wsConn, err := receiverClient.ConnectToRendezvous(parsedPassword)
	if err != nil {
		fmt.Println(err)
		return // TODO: be a better person
	}

	receivedBytes, err := receiverClient.Receive(wsConn)
	if err != nil {
		fmt.Println(err)
		return // TODO: be a better person
	}
	if receiverClient.DidUseRelay() {
		wsConn.WriteJSON(protocol.RendezvousMessage{Type: protocol.ReceiverToRendezvousClose})
	}

	fmt.Println(receivedBytes.Len())

	// just keep this alive
	for {

	}
}
