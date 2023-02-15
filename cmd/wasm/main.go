//go:build js

package main

import (
	"bytes"
	"context"
	"syscall/js"

	"github.com/SpatiumPortae/portal/portal"
)

// JS constructors
var Error js.Value = js.Global().Get("Error")
var Promise js.Value = js.Global().Get("Promise")
var Uint8Array js.Value = js.Global().Get("Uint8Array")

func main() {
	js.Global().Set("portalSend", SendJs())
	js.Global().Set("portalReceive", ReceiveJs())
	select {}
}

// SendJs returns a JS function that wraps portal.Send.
// The function returns a promise which resovles into an
// object containing the password and a second promise
// that can be awaited to scan for errors during transfer.
func SendJs() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		// Copy the JS data Go.
		jsData := Uint8Array.New(args[0])
		dst := make([]byte, jsData.Get("length").Int())
		js.CopyBytesToGo(dst, jsData)
		payload := bytes.NewBuffer(dst)

		// Get config
		var cnf *portal.Config
		if len(args) > 1 {
			cnf = configFromJs(args[1])
		}
		// Top-level promise.
		transferHandler := promiseHandler(func(resolve, reject js.Value) {
			password, err, errCh := portal.Send(context.Background(), payload, int64(payload.Len()), cnf)
			if err != nil {
				reject.Invoke(Error.New(err.Error()))
				return
			}
			// Second-level promise.
			errorHandler := promiseHandler(func(resolve, reject js.Value) {
				if err := <-errCh; err != nil {
					reject.Invoke(Error.New(err.Error()))
					return
				}
				resolve.Invoke()
			})
			resolve.Invoke(map[string]any{
				"password": password,
				"error":    Promise.New(errorHandler),
			})
		})
		return Promise.New(transferHandler)
	})
}

// ReceiveJs returns a JS function that wraps portal.Receive.
// The function returns a promise that resolves to the payload.
func ReceiveJs() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		password := args[0].String()

		// Get config
		var cnf *portal.Config
		if len(args) > 1 {
			cnf = configFromJs(args[1])
		}

		var buf bytes.Buffer
		transferHandler := promiseHandler(func(resolve, reject js.Value) {
			if err := portal.Receive(context.Background(), &buf, password, cnf); err != nil {
				reject.Invoke(Error.New(err.Error()))
				return
			}
			jsData := Uint8Array.New(buf.Len())
			js.CopyBytesToJS(jsData, buf.Bytes())
			resolve.Invoke(jsData)
		})
		return Promise.New(transferHandler)
	})
}

// promiseHandler utility function to create JS promises handlers.
func promiseHandler(handler func(resolve, reject js.Value)) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[0]
		go handler(resolve, reject)
		return nil
	})
}

func configFromJs(cnfJs js.Value) *portal.Config {
	cnf := portal.Config{}
	cnf.RendezvousAddr = cnfJs.Get("RendezvousAddr").String()
	return &cnf
}
