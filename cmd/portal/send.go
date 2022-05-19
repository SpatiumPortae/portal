package main

import (
	"fmt"
	"net"
	"os"

	"github.com/SpatiumPortae/portal/internal/file"
	"github.com/SpatiumPortae/portal/ui/sender"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		file.RemoveTemporaryFiles(file.SEND_TEMP_FILE_NAME_PREFIX)
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
	sender := sender.New(fileNames, net.TCPAddr{IP: net.ParseIP(addr), Port: port})
	if err := sender.Start(); err != nil {
		fmt.Println("Error initializing UI", err)
		os.Exit(1)
	}
	fmt.Println("")
	os.Exit(0)
}
