package main

import (
	"errors"
	"net"
	"regexp"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/net/idna"
)

var ErrInvalidRelay = errors.New("invalid relay provided")

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

	if ip := net.ParseIP(stripPort(relayAddr)); ip != nil {
		return nil
	}

	if _, err := idna.Lookup.ToASCII(relayAddr); err == nil {
		return nil
	}

	return ErrInvalidRelay
}
