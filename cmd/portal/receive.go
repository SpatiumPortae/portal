package main

import (
	"bytes"
	"fmt"
	"math"
	"os"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/receiver"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
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

	receivedBytesCh := make(chan *bytes.Buffer)
	wsConnCh := make(chan *websocket.Conn)

	go func(p models.Password) {
		wsConn, err := receiverClient.ConnectToRendezvous(p)
		if err != nil {
			fmt.Println(err)
			return // TODO: be a better person
		}
		wsConnCh <- wsConn
	}(parsedPassword)

	doneCh := make(chan bool)
	go func(wsConn *websocket.Conn) {
		receivedBytes, err := receiverClient.Receive(wsConn)
		if err != nil {
			fmt.Println(err)
			return // TODO: be a better person
		}
		if receiverClient.DidUseRelay() {
			wsConn.WriteJSON(protocol.RendezvousMessage{Type: protocol.ReceiverToRendezvousClose})
		}
		receivedBytesCh <- receivedBytes
	}(<-wsConnCh)

	go func() {
		// TODO: use this
		<-receivedBytesCh
		receiverUI.Send(receiverui.FinishedMsg{})
	}()

	go func() {
		latestProgress := 0
		for uiUpdate := range uiCh {
			// limit progress update ui-send events
			newProgress := int(math.Ceil(100 * float64(uiUpdate.Progress)))
			if newProgress > latestProgress {
				latestProgress = newProgress
				receiverUI.Send(ui.ProgressMsg{Progress: uiUpdate.Progress})
			}
		}
	}()

	// just keep this alive
	<-doneCh
}
