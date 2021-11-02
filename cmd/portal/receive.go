package main

import (
	"fmt"
	"os"

	receiverui "www.github.com/ZinoKader/portal/ui/receiver"
)

func handleReceiveCommand(password string) {
	// communicate ui updates on this channel between receiverClient and handleReceiveCmmand
	// uiCh := make(chan sender.UIUpdate)
	// initialize a receiverClient with a UI
	// receiverClient := receiver.WithUI(receiver.NewReceiver(log.New(ioutil.Discard, "", 0)), uiCh)
	// initialize and start sender-UI
	receiverUI := receiverui.NewReceiverUI()

	go func() {
		if err := receiverUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
		}
		os.Exit(0)
	}()

	// just keep this alive
	for {

	}
}
