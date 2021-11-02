package main

import (
	"flag"
	"fmt"
	"os"
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
		handleSendCommand(sendCmd.Args())

	case RECEIVE_COMMAND:
		if len(os.Args) <= 2 {
			fmt.Println("You must provide the password associated to the desired file.")
			return
		}
		receiveCmd.Parse(os.Args[2:])
		handleReceiveCommand(receiveCmd.Arg(0))

	default:
		fmt.Printf("Unrecognized command. Recognized commands: '%s' and '%s'.\n", SEND_COMMAND, RECEIVE_COMMAND)

	}
}
