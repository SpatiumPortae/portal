package main

import (
	"fmt"
	"math"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/models/protocol"
	"www.github.com/ZinoKader/portal/pkg/receiver"
	"www.github.com/ZinoKader/portal/tools"
	"www.github.com/ZinoKader/portal/ui"
	receiverui "www.github.com/ZinoKader/portal/ui/receiver"
)

// receiveCmd is the cobra command for `portal receive`
var receiveCmd = &cobra.Command{
	Use:   "receive",
	Short: "Receive files",
	Long:  "The receive command receives files from the sender with the matching password.",
	Args:  cobra.ExactArgs(1),
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
		err = setupLoggingFromViper("receive")
		if err != nil {
			return err
		}
		handleReceiveCommand(args[0])
		return nil
	},
}

// Setup flags
func init() {
	// Add subcommand flags (dummy default values as default values are handled through viper)
	//TODO: recactor this into a single flag for providing a TCPAddr
	receiveCmd.Flags().IntP("rendezvous-port", "p", 0, "port on which the rendezvous server is running")
	receiveCmd.Flags().StringP("rendezvous-address", "a", "", "host address for the rendezvous server")
}

// handleReceiveCommandis the receive application.
func handleReceiveCommand(password string) {
	// communicate ui updates on this channel between receiverClient and handleReceiveCmmand
	uiCh := make(chan receiver.UIUpdate)
	// initialize a receiverClient with a UI
	receiverClient := receiver.New(viper.GetString("rendezvousAddress"), viper.GetInt("rendezvousPort"), receiver.WithUI(uiCh))
	// initialize and start receiver-UI
	receiverUI := receiverui.NewReceiverUI()
	// clean up temporary files previously created by this command
	tools.RemoveTemporaryFiles(constants.RECEIVE_TEMP_FILE_NAME_PREFIX)

	go initReceiverUI(receiverUI)
	time.Sleep(ui.START_PERIOD)
	go listenForReceiverUIUpdates(receiverUI, uiCh)

	parsedPassword, err := tools.ParsePassword(password)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg(fmt.Errorf("Error parsing password, make sure you entered a correctly formatted password (e.g. 1-gamma-ray-quasar).")))
		ui.GracefulUIQuit(receiverUI)
	}

	// initiate communications with rendezvous-server
	wsConnCh := make(chan *websocket.Conn)
	go initiateReceiverRendezvousCommunication(receiverClient, receiverUI, parsedPassword, wsConnCh)

	// keeps program alive until finished
	doneCh := make(chan bool)
	// start receiving files
	go startReceiving(receiverClient, receiverUI, <-wsConnCh, doneCh)

	// wait for shut down to render final UI
	<-doneCh
	ui.GracefulUIQuit(receiverUI)
}

func initReceiverUI(receiverUI *tea.Program) {
	go func() {
		if err := receiverUI.Start(); err != nil {
			fmt.Println("Error initializing UI", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
}

func listenForReceiverUIUpdates(receiverUI *tea.Program, uiCh chan receiver.UIUpdate) {
	latestProgress := 0
	for uiUpdate := range uiCh {
		// limit progress update ui-send events
		newProgress := int(math.Ceil(100 * float64(uiUpdate.Progress)))
		if newProgress > latestProgress {
			latestProgress = newProgress
			receiverUI.Send(ui.ProgressFMsg{Progress: uiUpdate.Progress})
		}
	}
}

func initiateReceiverRendezvousCommunication(receiverClient *receiver.Receiver, receiverUI *tea.Program, password models.Password, connectionCh chan *websocket.Conn) {
	wsConn, err := receiverClient.ConnectToRendezvous(receiverClient.RendezvousAddress(), receiverClient.RendezvousPort(), password)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg(fmt.Errorf("Something went wrong during connection-negotiation (did you enter the correct password?)")))
		ui.GracefulUIQuit(receiverUI)
	}
	receiverUI.Send(ui.FileInfoMsg{Bytes: receiverClient.PayloadSize()})
	connectionCh <- wsConn
}

func startReceiving(receiverClient *receiver.Receiver, receiverUI *tea.Program, wsConnection *websocket.Conn, doneCh chan bool) {
	tempFile, err := os.CreateTemp(os.TempDir(), constants.RECEIVE_TEMP_FILE_NAME_PREFIX)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg(fmt.Errorf("Something went wrong when creating the received file container.")))
		ui.GracefulUIQuit(receiverUI)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// start receiving files from sender
	err = receiverClient.Receive(wsConnection, tempFile)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg(fmt.Errorf("Something went wrong during file transfer.")))
		ui.GracefulUIQuit(receiverUI)
	}
	if receiverClient.UsedRelay() {
		wsConnection.WriteJSON(protocol.RendezvousMessage{Type: protocol.ReceiverToRendezvousClose})
	}

	// reset file position for reading
	tempFile.Seek(0, 0)

	// read received bytes from tmpFile
	receivedFileNames, decompressedSize, err := tools.DecompressAndUnarchiveBytes(tempFile)
	if err != nil {
		receiverUI.Send(ui.ErrorMsg(fmt.Errorf("Something went wrong when expanding the received files.")))
		ui.GracefulUIQuit(receiverUI)
	}
	receiverUI.Send(ui.FinishedMsg{Files: receivedFileNames, PayloadSize: decompressedSize})
	doneCh <- true
}
