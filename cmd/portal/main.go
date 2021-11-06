package main

import (
	"errors"
	"net"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jessevdk/go-flags"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/tools"
)

type SendCommandOptions struct{}

type ReceiveCommandOptions struct{}

var sendCommand SendCommandOptions
var receiveCommand ReceiveCommandOptions

var programOptions struct {
	Verbose           bool   `short:"v" long:"verbose" description:"Log detailed debug information"`
	RendezvousAddress string `short:"s" long:"server" description:"IP or hostname of the rendezvous server to use"`
	RendezvousPort    int    `short:"p" long:"port" description:"Port of the rendezvous server to use" default:"80"`
}

var parser = flags.NewParser(&programOptions, flags.Default)

func (s *SendCommandOptions) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("No files provided. The send command takes file(s) delimited by spaces as arguments.")
	}

	err := validateRendezvousAddress()
	if err != nil {
		return err
	}

	if programOptions.Verbose {
		f, err := tea.LogToFile("portal-send.log", "portal-send: ")
		if err != nil {
			return err
		}
		defer f.Close()
	}

	handleSendCommand(models.ProgramOptions{
		RendezvousAddress: programOptions.RendezvousAddress,
		RendezvousPort:    programOptions.RendezvousPort,
	}, args)
	return nil
}

func (r *ReceiveCommandOptions) Execute(args []string) error {
	if len(args) > 1 {
		return errors.New("Provide a single password, for instance 1-cosmic-ray-quasar.")
	}
	if len(args) < 1 {
		return errors.New("Provide the password that the file sender gave to you, for instance 1-galaxy-dust-aurora.")
	}

	err := validateRendezvousAddress()
	if err != nil {
		return err
	}

	if programOptions.Verbose {
		f, err := tea.LogToFile("portal-receive.log", "portal-receive: ")
		if err != nil {
			return err
		}
		defer f.Close()
	}

	handleReceiveCommand(models.ProgramOptions{
		RendezvousAddress: programOptions.RendezvousAddress,
		RendezvousPort:    programOptions.RendezvousPort,
	}, args[0])
	return nil
}

func init() {
	tools.RandomSeed()

	parser.AddCommand("send",
		"Send one or more files",
		"The send command adds one or more files to be sent. Files are archived and compressed before sending.",
		&sendCommand)

	parser.AddCommand("receive",
		"Receive files",
		"The receive command receives files from the sender with the matching password.",
		&receiveCommand)

	parser.FindOptionByLongName("server").Default = []string{constants.DEFAULT_RENDEZVOUZ_ADDRESS}
}

// entry point for send/receive commands
func main() {
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}

func validateRendezvousAddress() error {
	rendezvouzAdress := net.ParseIP(programOptions.RendezvousAddress)
	err := tools.ValidateHostname(programOptions.RendezvousAddress)
	// neither a valid IP nor a valid hostname was provided
	if (rendezvouzAdress == nil) && err != nil {
		return errors.New("Invalid IP or hostname provided.")
	}
	return nil
}
