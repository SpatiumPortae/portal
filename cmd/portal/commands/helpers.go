package commands

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

const (
	relayFlagDesc = `Address of relay server. Accepted formats:
  - 127.0.0.1:8080
  - [::1]:8080
  - somedomain.com
	`
	tuiStyleFlagDesc = "Style of the tui (rich|raw)"
)

var validate = validator.New()
var ErrInvalidAddress = errors.New("invalid address provided")

// validateAddress validates a hostname or IP, optionally with a port.
func validateAddress(addr string) error {

	// IPv4 and IPv6 address validation.
	err := validate.Var(addr, "ip")
	if err == nil {
		return nil
	}

	// IPv4 or IPv6 or domain or localhost.
	err = validate.Var(addr, "hostname")
	if err == nil {
		return nil
	}

	// IPv4 or domain or localhost and a port. Or just a shortand port (:1234).
	err = validate.Var(addr, "hostname_port")
	if err == nil {
		return nil
	}

	// Also validate IPv6 host + port combination. The hostname_port validator does not validate this.
	_, port, hostPortErr := net.SplitHostPort(addr)
	// Additionally, validate the port range.
	if p, err := strconv.Atoi(port); err != nil || p < 0 || p > 65535 {
		return ErrInvalidAddress
	}
	if hostPortErr == nil {
		return nil
	}

	return ErrInvalidAddress
}

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
