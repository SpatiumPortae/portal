package commands

import (
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

const (
	relayFlagDesc = `Address of relay server. Accepted formats:
  - 127.0.0.1:8080
  - [::1]:8080
  - somedomain.com/relay
	- ...
	`
	tuiStyleFlagDesc = "Style of the tui (rich|raw)"
)

func setupLoggingFromViper(cmd string) (*os.File, error) {
	if viper.GetBool("verbose") {
		f, err := tea.LogToFile(fmt.Sprintf(".portal-%s.log", cmd), fmt.Sprintf("portal-%s: \n", cmd))
		if err != nil {
			return nil, fmt.Errorf("could not log to the provided file: %w", err)
		}
		return f, nil
	}
	log.SetOutput(io.Discard)
	return nil, nil
}
