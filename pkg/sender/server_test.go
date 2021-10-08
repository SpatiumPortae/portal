package sender

import (
	"net"
	"testing"
)

func TestServer(t *testing.T) {
	server, err := NewServer(8080, []byte("Portal this shiiiiet"), net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Fail()
	}
	server.Start()
}
