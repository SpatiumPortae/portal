package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

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
	receiverClient := receiver.WithUI(receiver.NewReceiver(log.Default()), uiCh)
	// initialize and start sender-UI
	receiverUI := receiverui.NewReceiverUI()

	go func() {
		if err := receiverUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()

	parsedPassword, err := tools.ParsePassword(password)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg{Message: "Error parsing password, make sure you entered a correct password"})
		return
	}

	wsConnCh := make(chan *websocket.Conn)

	go func(p models.Password) {
		wsConn, err := receiverClient.ConnectToRendezvous(p)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong during connection-negotiation"})
			os.Exit(1)
		}
		wsConnCh <- wsConn
	}(parsedPassword)

	doneCh := make(chan bool)
	go func(wsConn *websocket.Conn) {
		receivedBytes, err := receiverClient.Receive(wsConn)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong during file transfer"})
			os.Exit(1)
		}
		if receiverClient.DidUseRelay() {
			wsConn.WriteJSON(protocol.RendezvousMessage{Type: protocol.ReceiverToRendezvousClose})
		}

		receivedFileNames, err := tools.DecompressAndUnarchiveBytes(receivedBytes)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong when expanding the received files"})
			os.Exit(1)
		}
		receiverUI.Send(receiverui.FinishedMsg{ReceivedFiles: receivedFileNames})
		doneCh <- true
	}(<-wsConnCh)

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

	// wait for shut down to render final UI
	<-doneCh
	timer := time.NewTimer(ui.SHUTDOWN_PERIOD)
	<-timer.C
}
