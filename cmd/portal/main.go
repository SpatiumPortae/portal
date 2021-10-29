package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/pkg/sender"
	"www.github.com/ZinoKader/portal/tools"
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
	compressedBufferCh := make(chan bytes.Buffer)
	passCh := make(chan models.Password)

	files, err := tools.ReadFiles(fileNames)
	if err != nil {
		fmt.Printf("Error reading file(s): %s\n", err.Error())
		return
	}

	fileName := "combined"
	if len(files) == 1 {
		fileName = files[0].Name()
	}

	fileSize, err := tools.FilesTotalSize(files)
	if err != nil {
		fmt.Printf("Error read file sizes: %s\n", err.Error())
		return
	}

	// compress files in parallel
	go func() {
		compressedBytes, err := tools.CompressFiles(files)
		if err != nil {
			fmt.Printf("Error compressing file(s): %s\n", err.Error())
			return
		}
		compressedBufferCh <- compressedBytes
	}()

	sender.ConnectToRendevouz(passCh)
	connectionPassword := <-passCh
	fmt.Println(connectionPassword)

	compressedBuffer := <-compressedBufferCh
	fmt.Println(fileName, fileSize, compressedBuffer.Len())
}

func receive() {

}
