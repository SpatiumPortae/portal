package config

import (
	"fmt"
	"net"
	"os"
)

const RENDEZVOUS_ADDRESS_ENV = "PORTAL_RENDEZVOUS_ADDR"

func GetEnvRendezvousAddress() (net.IP, error) {
	envAddr := os.Getenv(RENDEZVOUS_ADDRESS_ENV)
	ip := net.ParseIP(envAddr)
	if ip == nil {
		return nil, fmt.Errorf("no valid IP provided in RENDEZVOUS_ADDRESS_ENV")
	}
	return ip, nil
}
