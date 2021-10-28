package tools

import "time"

func NewTimeoutChannel(timeout time.Duration) chan bool {
	t := make(chan bool, 1)
	go func() {
		time.Sleep(timeout)
		t <- true
	}()
	return t
}
