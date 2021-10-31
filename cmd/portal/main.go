package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/sender"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
)

const SEND_COMMAND = "send"
const RECEIVE_COMMAND = "receive"

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("Usage: 'portal %s' to send files and 'portal %s [password]' to receive files\n", SEND_COMMAND, RECEIVE_COMMAND)
		return
	}

	sendCmd := flag.NewFlagSet(SEND_COMMAND, flag.ExitOnError)
	receiveCmd := flag.NewFlagSet(RECEIVE_COMMAND, flag.ExitOnError)

	switch os.Args[1] {
	case SEND_COMMAND:
		if len(os.Args) <= 2 {
			fmt.Println("Provide either one or more files/folder delimited by spaces, or a text string enclosed by quotes.")
			return
		}
		sendCmd.Parse(os.Args[2:])
		send(sendCmd.Args())
	case RECEIVE_COMMAND:
		receiveCmd.Parse(os.Args[2:])
		receive()
	default:
		fmt.Printf("Unrecognized command. Recognized commands: '%s' and '%s'.\n", SEND_COMMAND, RECEIVE_COMMAND)
	}
}

func send(fileNames []string) {
	// create and start sender UI
	senderUI := ui.NewSenderUI()
	go func() {
		if err := senderUI.Start(); err != nil {
			fmt.Println("Error initializing  UI", err)
			os.Exit(1)
		}
	}()

	fileContentsBufferCh := make(chan *bytes.Buffer, 1)
	senderReadyCh := make(chan bool)
	// read, archive and compress files in parallel
	go func() {
		files, err := tools.ReadFiles(fileNames)
		if err != nil {
			fmt.Printf("Error reading file(s): %s\n", err.Error())
			return
		}
		compressedBytes, err := tools.CompressFiles(files)
		for _, file := range files {
			file.Close()
		}
		if err != nil {
			fmt.Printf("Error compressing file(s): %s\n", err.Error())
			return // TODO: replace with graceful shutdown, this does nothing!
		}
		fileContentsBufferCh <- compressedBytes
		senderReadyCh <- true
	}()

	// initiate communications with rendezvous-server
	senderPortCh := make(chan int)
	receiverIPCh := make(chan net.IP)
	passCh := make(chan models.Password)
	go func() {
		senderPort, receiverIP, err := sender.ConnectToRendevouz(passCh, senderReadyCh)
		if err != nil {
			fmt.Printf("Failed connecting to rendezvous server: %s\n", err.Error())
			return // TODO: replace with graceful shutdown, this does nothing!
		}
		senderPortCh <- senderPort
		receiverIPCh <- receiverIP
	}()

	connectionPassword := <-passCh
	senderUI.Send(ui.PasswordMsg{Password: string(connectionPassword)})

	// send payload to receiver
	uiCh := make(chan sender.UIUpdate)
	senderPort := <-senderPortCh
	receiverIP := <-receiverIPCh
	fileContentsBuffer := <-fileContentsBufferCh
	s := sender.WithUI(sender.NewServer(
		senderPort, fileContentsBuffer, fileContentsBuffer.Len(), receiverIP, log.New(ioutil.Discard, "", 0)),
		uiCh)

	go func() {
		for uiUpdate := range uiCh {
			senderUI.Send(ui.ProgressMsg{Progress: uiUpdate.Progress})
		}
	}()

	// start sender server
	s.Start()
}

func receive() {
}
