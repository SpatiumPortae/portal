package client

import (
	"testing"
)

func TestGetOpenPort(t *testing.T) {

	var ports []int
	for i := 0; i < 5000; i++ {
		port, err := GetOpenPort()
		if err != nil {
			t.Fatal(err)
		}
		ports = append(ports, port)
	}


}