package main

import (
	"math/rand"
	"time"

	"www.github.com/ZinoKader/portal/pkg/rendezvous"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	s := rendezvous.NewServer()
	s.Start()
}
