package main

import (
	"www.github.com/ZinoKader/portal/pkg/rendezvous"
	"www.github.com/ZinoKader/portal/tools"
)

func init() {
	tools.RandomSeed()
}

func main() {
	s := rendezvous.NewServer()
	s.Start()
}
