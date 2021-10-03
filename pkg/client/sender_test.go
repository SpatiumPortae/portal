package client

import "testing"

func TestServer(t *testing.T) {
	server, err := NewServer(8080, []byte("Portal this shiiiiet"))
	if err != nil {
		t.Fail()
	}
	server.Start()
}
