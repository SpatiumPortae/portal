package main

import (
	"fmt"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	senderui "www.github.com/ZinoKader/portal/ui/sender"
)

// sendCmd cobra command for `portal send`.
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send one or more files",
	Long:  "The send command adds one or more files to be sent. Files are archived and compressed before sending.",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind flags to viper
		viper.BindPFlag("rendezvousPort", cmd.Flags().Lookup("rendezvous-port"))
		viper.BindPFlag("rendezvousAddress", cmd.Flags().Lookup("rendezvous-address"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := validateRendezvousAddressInViper()
		if err != nil {
			return err
		}

		err = setupLoggingFromViper("send")
		if err != nil {
			return err
		}

		handleSendCommand(args)
		return nil
	},
}

// Set flags.
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper)
	//TODO: recactor this into a single flag for providing a TCPAddr
	sendCmd.Flags().IntP("rendezvous-port", "p", 0, "port on which the rendezvous server is running")
	sendCmd.Flags().StringP("rendezvous-address", "a", "", "host address for the rendezvous server")
}

// handleSendCommand is the sender application.
func handleSendCommand(fileNames []string) {
	addr := viper.GetString("rendezvousAddress")
	port := viper.GetInt("rendezvousPort")
	sender := senderui.NewSenderUI(fileNames, net.TCPAddr{IP: net.ParseIP(addr), Port: port})
	initSenderUI(sender)
}

func initSenderUI(senderUI *tea.Program) {
	if err := senderUI.Start(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}

// func listenForSenderUIUpdates(senderUI *tea.Program, uiCh chan sender.UIUpdate) {
// 	latestProgress := 0
// 	for uiUpdate := range uiCh {
// 		// make sure progress is 100 if connection is to be closed
// 		if uiUpdate.State == sender.WaitForCloseMessage {
// 			latestProgress = 100
// 			senderUI.Send(ui.ProgressMsg{Progress: 1})
// 			continue
// 		}
// 		// limit progress update ui-send events
// 		newProgress := int(math.Ceil(100 * float64(uiUpdate.Progress)))
// 		if newProgress > latestProgress {
// 			latestProgress = newProgress
// 			senderUI.Send(ui.ProgressMsg{Progress: uiUpdate.Progress})
// 		}
// 	}
// }

// func prepareFiles(senderClient *sender.Sender, senderUI *tea.Program, fileNames []string, readyCh chan bool, closeFileCh chan *os.File) {
// 	files, err := tools.ReadFiles(fileNames)
// 	if err != nil {
// 		log.Println("Error reading files: ", err)
// 		senderUI.Send(ui.ErrorMsg{Message: "Error reading files."})
// 		ui.GracefulUIQuit(senderUI)
// 	}
// 	uncompressedFileSize, err := tools.FilesTotalSize(files)
// 	if err != nil {
// 		log.Println("Error during file preparation: ", err)
// 		senderUI.Send(ui.ErrorMsg{Message: "Error during file preparation."})
// 		ui.GracefulUIQuit(senderUI)
// 	}
// 	senderUI.Send(ui.FileInfoMsg{FileNames: fileNames, Bytes: uncompressedFileSize})

// 	tempFile, fileSize, err := tools.ArchiveAndCompressFiles(files)
// 	for _, file := range files {
// 		file.Close()
// 	}
// 	if err != nil {
// 		log.Println("Error compressing files: ", err)
// 		senderUI.Send(ui.ErrorMsg{Message: "Error compressing files."})
// 		ui.GracefulUIQuit(senderUI)
// 	}
// 	sender.WithPayload(tempFile, fileSize)(senderClient)
// 	senderUI.Send(ui.FileInfoMsg{FileNames: fileNames, Bytes: fileSize})
// 	readyCh <- true
// 	senderUI.Send(senderui.ReadyMsg{})
// 	closeFileCh <- tempFile
// }

// func initiateSenderRendezvousCommunication(senderClient *sender.Sender, senderUI *tea.Program, passCh chan models.Password,
// 	startServerCh chan sender.ServerOptions, relayCh chan *websocket.Conn) {
// 	err, wsConn := senderClient.ConnectToRendezvous(passCh, startServerCh)

// 	if err != nil {
// 		senderUI.Send(ui.ErrorMsg{Message: "Failed to communicate with rendezvous server."})
// 		ui.GracefulUIQuit(senderUI)
// 	}

// 	if wsConn != nil {
// 		relayCh <- wsConn
// 	}
// }

// func startDirectCommunicationServer(senderClient *sender.Sender, senderUI *tea.Program, doneCh chan bool) {
// 	if err := senderClient.StartServer(); err != nil {
// 		senderUI.Send(ui.ErrorMsg{Message: fmt.Sprintf("Something went wrong during file transfer: %e", err)})
// 		ui.GracefulUIQuit(senderUI)
// 	}
// 	doneCh <- true
// }

// func prepareRelayCommunicationFallback(senderClient *sender.Sender, senderUI *tea.Program, relayCh chan *websocket.Conn, doneCh chan bool) {
// 	if relayWsConn, closed := <-relayCh; closed {
// 		// start transferring to the rendezvous-relay
// 		go func() {
// 			if err := senderClient.Transfer(relayWsConn); err != nil {
// 				senderUI.Send(ui.ErrorMsg{Message: fmt.Sprintf("Something went wrong during file transfer: %e", err)})
// 				ui.GracefulUIQuit(senderUI)
// 			}
// 			doneCh <- true
// 		}()
// 	}
// }
