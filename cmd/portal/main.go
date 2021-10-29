package main

import (
	"flag"
	"fmt"
	"os"

	"www.github.com/ZinoKader/portal/models"
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
	// compressionCh := make(chan )
	passCh := make(chan models.Password)
	files, err := readFiles(fileNames)
	if err != nil {
		fmt.Printf("Error reading file(s): %s\n", err.Error())
		return
	}

	fileName := "combined"
	if len(files) == 1 {
		fileName := files[0].Name()
	}

}

func receive() {

}

func readFiles(fileNames []string) ([]*os.File, error) {
	files := make([]*os.File, len(fileNames))
	for _, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("file '%s' not found", fileName)
		}
		files = append(files, f)
	}
	return files, nil
}

func compressFiles([]*os.File files) {
	
}