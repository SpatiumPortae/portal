package main

import (
	"fmt"

	"github.com/ZinoKader/portal/constants"
	"github.com/ZinoKader/portal/pkg/rendezvous"
	"github.com/ZinoKader/portal/tools"
	"github.com/jessevdk/go-flags"
)

var flagOpts struct {
	Port int `short:"p" long:"port" description:"The port to host the rendezvous-server on"`
}

func init() {
	tools.RandomSeed()
}

func main() {
	_, err := flags.Parse(&flagOpts)
	if err != nil {
		fmt.Println("Unable to parse flags. Run \"portal-rendezvous --help\" to see all available flags.")
		return
	}

	port := flagOpts.Port
	if port == 0 {
		port = constants.DEFAULT_RENDEZVOUZ_PORT
	}
	s := rendezvous.NewServer(port)
	s.Start()
}
