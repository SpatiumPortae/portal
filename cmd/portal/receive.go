package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/receiver"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
	receiverui "www.github.com/ZinoKader/portal/ui/receiver"
)

// handleReceiveCommandis the receive application.
func handleReceiveCommand(password string) {
	// communicate ui updates on this channel between receiverClient and handleReceiveCmmand
	uiCh := make(chan receiver.UIUpdate)
	// initialize a receiverClient with a UI
	receiverClient := receiver.WithUI(receiver.NewReceiver(log.Default()), uiCh)
	// initialize and start sender-UI
	receiverUI := receiverui.NewReceiverUI()
	// clean up temporary files previously created by this command
	tools.RemoveTemporaryFiles(constants.RECEIVE_TEMP_FILE_NAME_PREFIX)

	go func() {
		if err := receiverUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	uiStartGraceTimeout := time.NewTimer(ui.START_PERIOD)
	<-uiStartGraceTimeout.C

	parsedPassword, err := tools.ParsePassword(password)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg{Message: "Error parsing password, make sure you entered a correct password"})
		ui.GracefulUIQuit(receiverUI)
	}

	wsConnCh := make(chan *websocket.Conn)

	go func(p models.Password) {
		wsConn, err := receiverClient.ConnectToRendezvous(p)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong during connection-negotiation"})
			ui.GracefulUIQuit(receiverUI)
		}
		receiverUI.Send(ui.FileInfoMsg{Bytes: receiverClient.GetPayloadSize()})
		wsConnCh <- wsConn
	}(parsedPassword)

	doneCh := make(chan bool)
	go func(wsConn *websocket.Conn) {
		receivedBytes, err := receiverClient.Receive(wsConn)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong during file transfer"})
			ui.GracefulUIQuit(receiverUI)
		}
		if receiverClient.DidUseRelay() {
			wsConn.WriteJSON(protocol.RendezvousMessage{Type: protocol.ReceiverToRendezvousClose})
		}

		receivedFileNames, decompressedSize, err := tools.DecompressAndUnarchiveBytes(receivedBytes)
		if err != nil {
			receiverUI.Send(ui.ErrorMsg{Message: "Something went wrong when expanding the received files"})
			ui.GracefulUIQuit(receiverUI)
		}
		receiverUI.Send(receiverui.FinishedMsg{ReceivedFiles: receivedFileNames, DecompressedPayloadSize: decompressedSize})
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
	ui.GracefulUIQuit(receiverUI)
}
