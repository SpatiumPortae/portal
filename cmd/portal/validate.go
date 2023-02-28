package main

import (
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/net/idna"
)

var ErrInvalidRelay = errors.New("invalid relay address provided")

var ipv6Rex = regexp.MustCompile(`\[(.*?)\]`)

func stripPort(addr string) string {
	split := strings.Split(addr, ":")
	if len(split) == 2 {
		return split[0]
	}

	matches := ipv6Rex.FindStringSubmatch(addr)
	if len(matches) >= 2 {
		return matches[1]
	}
	return addr
}

// validateRelayInViper validates that the `relay` value in viper is a valid hostname or IP
func validateRelayInViper() error {
	relayAddr := viper.GetString("relay")

	onlyHost := stripPort(relayAddr)

	// Port is present, validate it.
	if relayAddr != onlyHost {
		_, port, err := net.SplitHostPort(relayAddr)
		if err != nil {
			return ErrInvalidRelay
		}
		portNumber, err := strconv.Atoi(port)
		if err != nil {
			return ErrInvalidRelay
		}
		if portNumber < 1 || portNumber > 65535 {
			return ErrInvalidRelay
		}
	}

	// Only port is present, and was valid -- accept an address like ":5432".
	if len(relayAddr) > 0 && len(onlyHost) == 0 {
		return nil
	}

	// On the form localhost or localhost:1234, valid.
	if onlyHost == "localhost" {
		return nil
	}

	if ip := net.ParseIP(onlyHost); ip != nil {
		return nil
	}

	if _, err := idna.Lookup.ToASCII(relayAddr); err == nil {
		return nil
	}

	return ErrInvalidRelay
}
