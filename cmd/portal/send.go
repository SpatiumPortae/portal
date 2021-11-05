package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/sender"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
	senderui "www.github.com/ZinoKader/portal/ui/sender"
)

func handleSendCommand(fileNames []string) {
	// communicate ui updates on this channel between senderClient and handleSendCommand
	uiCh := make(chan sender.UIUpdate)
	// initialize a senderClient with a UI
	senderClient := sender.WithUI(sender.NewSender(log.New(ioutil.Discard, "", 0)), uiCh)
	// initialize and start sender-UI
	senderUI := senderui.NewSenderUI()
	// clean up temporary files previously created by this command
	tools.RemoveTemporaryFiles(constants.SEND_TEMP_FILE_NAME_PREFIX)

	go func() {
		if err := senderUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	uiStartGraceTimeout := time.NewTimer(ui.START_PERIOD)
	<-uiStartGraceTimeout.C

	senderReadyCh := make(chan bool, 1)
	// read, archive and compress files in parallel
	go func() {
		files, err := tools.ReadFiles(fileNames)
		if err != nil {
			senderUI.Send(ui.ErrorMsg{Message: "Error reading files"})
			ui.GracefulUIQuit(senderUI)
		}
		fileSizesBytes, err := tools.FilesTotalSize(files)
		if err != nil {
			senderUI.Send(ui.ErrorMsg{Message: "Error during file preparation"})
			ui.GracefulUIQuit(senderUI)
		}
		senderUI.Send(ui.FileInfoMsg{FileNames: fileNames, Bytes: fileSizesBytes})
		// TODO: close this file in a good way
		tempFile, fileSize, err := tools.ArchiveAndCompressFiles(files)
		for _, file := range files {
			file.Close()
		}
		if err != nil {
			senderUI.Send(ui.ErrorMsg{Message: "Error compressing files"})
			ui.GracefulUIQuit(senderUI)
		}
		sender.WithPayload(senderClient, tempFile, fileSize)
		senderUI.Send(ui.FileInfoMsg{FileNames: fileNames, Bytes: fileSize})
		senderReadyCh <- true
		senderUI.Send(senderui.ReadyMsg{})
	}()

	// initiate communications with rendezvous-server
	startServerCh := make(chan sender.ServerOptions)
	relayCh := make(chan *websocket.Conn)
	passCh := make(chan models.Password)
	go func() {
		err := senderClient.ConnectToRendezvous(passCh, startServerCh, senderReadyCh, relayCh)
		if err != nil {
			senderUI.Send(ui.ErrorMsg{Message: "Failed to communicate with rendezvous server"})
			ui.GracefulUIQuit(senderUI)
		}
	}()

	// receive password and send to UI
	senderUI.Send(senderui.PasswordMsg{Password: string(<-passCh)})

	go func() {
		latestProgress := 0
		for uiUpdate := range uiCh {
			// make sure progress is 100 if connection is to be closed
			if uiUpdate.State == sender.WaitForCloseMessage {
				latestProgress = 100
				senderUI.Send(ui.ProgressMsg{Progress: 1})
				continue
			}
			// limit progress update ui-send events
			newProgress := int(math.Ceil(100 * float64(uiUpdate.Progress)))
			if newProgress > latestProgress {
				latestProgress = newProgress
				senderUI.Send(ui.ProgressMsg{Progress: uiUpdate.Progress})
			}
		}
	}()

	// keeps program alive
	doneCh := make(chan bool)
	// attach server to senderClient
	senderClient = sender.WithServer(senderClient, <-startServerCh)

	// start sender-server to be able to respond to receiver direct-communication-probes
	go func() {
		if err := senderClient.StartServer(); err != nil {
			senderUI.Send(ui.ErrorMsg{Message: fmt.Sprintf("Something went wrong during file transfer: %e", err)})
			ui.GracefulUIQuit(senderUI)
		}
		doneCh <- true
	}()

	if relayWsConn, closed := <-relayCh; closed {
		// close our direct-communication server and start transferring to the rendezvous-relay
		senderClient.CloseServer()
		go func() {
			if err := senderClient.Transfer(relayWsConn); err != nil {
				senderUI.Send(ui.ErrorMsg{Message: fmt.Sprintf("Something went wrong during file transfer: %e", err)})
				ui.GracefulUIQuit(senderUI)
			}
			doneCh <- true
		}()
	}

	// wait for shut down to render final UI
	<-doneCh
	ui.GracefulUIQuit(senderUI)
}
