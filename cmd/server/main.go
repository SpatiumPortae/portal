package main

import (
	"math/rand"
	"time"

	"www.github.com/ZinoKader/portal/pkg/server"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	server.Start()
}
