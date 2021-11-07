package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jessevdk/go-flags"
	"www.github.com/ZinoKader/portal/constants"
	"www.github.com/ZinoKader/portal/models"
	"www.github.com/ZinoKader/portal/tools"
)

type SendCommandOptions struct{}
type ReceiveCommandOptions struct{}
type AddCompletionsCommandOptions struct{}

const SHELL_COMPLETION_SCRIPT = `_portal_completions() {
	args=("${COMP_WORDS[@]:1:$COMP_CWORD}")

	local IFS=$'\n'
	COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
	return 1
}
complete -F _portal_completions portal
`

var sendCommand SendCommandOptions
var receiveCommand ReceiveCommandOptions
var addCompletionsCommand AddCompletionsCommandOptions

var programOptions struct {
	Verbose           string `short:"v" long:"verbose" optional:"true" optional-value:"no-file-specified" description:"Log detailed debug information (optional argument: specify output file with v=mylogfile or --verbose=mylogfile)"`
	RendezvousAddress string `short:"s" long:"server" description:"IP or hostname of the rendezvous server to use"`
	RendezvousPort    int    `short:"p" long:"port" description:"Port of the rendezvous server to use" default:"80"`
}

var parser = flags.NewParser(&programOptions, flags.Default)

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

	parser.AddCommand("add-completions",
		"Add command line completions for bash and zsh",
		"The add-completions command adds command line completions to your shell. Uses the value from the $SHELL environment variable.",
		&addCompletionsCommand)

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

// Execute is executed when the "send" command is invoked
func (s *SendCommandOptions) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("No files provided. The send command takes file(s) delimited by spaces as arguments.")
	}

	err := validateRendezvousAddress()
	if err != nil {
		return err
	}

	if len(programOptions.Verbose) != 0 {
		logFileName := programOptions.Verbose
		if programOptions.Verbose == "no-file-specified" {
			logFileName = "portal-send.log"
		}
		f, err := tea.LogToFile(logFileName, "portal-send: ")
		if err != nil {
			return errors.New("Could not log to the provided file.")
		}
		defer f.Close()
	} else {
		log.SetOutput(io.Discard)
	}

	handleSendCommand(models.ProgramOptions{
		RendezvousAddress: programOptions.RendezvousAddress,
		RendezvousPort:    programOptions.RendezvousPort,
	}, args)
	return nil
}

// Execute is executed when the "receive" command is invoked
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

	if len(programOptions.Verbose) != 0 {
		logFileName := programOptions.Verbose
		if programOptions.Verbose == "no-file-specified" {
			logFileName = "portal-receive.log"
		}
		f, err := tea.LogToFile(logFileName, "portal-receive: ")
		if err != nil {
			return errors.New("Could not log to the provided file.")
		}
		defer f.Close()
	} else {
		log.SetOutput(io.Discard)
	}

	handleReceiveCommand(models.ProgramOptions{
		RendezvousAddress: programOptions.RendezvousAddress,
		RendezvousPort:    programOptions.RendezvousPort,
	}, args[0])
	return nil
}

// Execute is executed when the "add-completions" command is invoked
func (a *AddCompletionsCommandOptions) Execute(args []string) error {
	shellBinPath := os.Getenv("SHELL")
	if len(shellBinPath) == 0 {
		return fmt.Errorf(
			"Completions not added - could not find which shell is used.\nTo add completions manually, add the following to your config:\n\n%s", SHELL_COMPLETION_SCRIPT)
	}

	shellPathComponents := strings.Split(os.Getenv("SHELL"), "/")
	usedShell := shellPathComponents[len(shellPathComponents)-1]
	if !tools.Contains([]string{"bash", "zsh"}, usedShell) {
		return fmt.Errorf("Unsupported shell \"%s\" at path: \"%s\".", usedShell, shellBinPath)
	}

	err := writeShellCompletionScript(usedShell)
	if err != nil {
		return fmt.Errorf("Failed when adding script to shell config file: %e", err)
	}

	fmt.Println("Successfully added completions to your shell config. Run 'source' on your shell config or restart your shell.")
	return nil
}

// writeShellCompletionScript writes the completion script to the specified shell name
func writeShellCompletionScript(shellName string) error {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	shellConfigName := fmt.Sprintf(".%src", shellName)
	shellConfigPath := path.Join(homedir, shellConfigName)
	f, err := os.OpenFile(shellConfigPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("\n# portal shell completion\n%s\n", SHELL_COMPLETION_SCRIPT)); err != nil {
		return err
	}

	return nil
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
