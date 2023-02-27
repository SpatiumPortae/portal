package main

import (
	"errors"
	"net"

	"github.com/go-playground/validator/v10"
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
	_, _, err = net.SplitHostPort(addr)
	if err == nil {
		return nil
	}

	return ErrInvalidAddress
}
