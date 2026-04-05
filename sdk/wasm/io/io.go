//go:build js && wasm

package io

import "syscall/js"

type console struct{}

// Log prints messages to the browser console.
func (c console) Log(args ...any) {
	js.Global().Get("console").Call("log", args...)
}

// Warn prints warning messages to the browser console.
func (c console) Warn(args ...any) {
	js.Global().Get("console").Call("warn", args...)
}

// Warning is an alias for Warn.
func (c console) Warning(args ...any) {
	c.Warn(args...)
}

// Error prints error messages to the browser console.
func (c console) Error(args ...any) {
	js.Global().Get("console").Call("error", args...)
}

// Console provides access to the browser's debug console.
var Console console

// Alert displays a browser alert dialog with a message.
func Alert(msg any) {
	js.Global().Call("alert", msg)
}

// Prompt displays a browser prompt dialog and returns the user input.
func Prompt(msg any) string {
	res := js.Global().Call("prompt", msg)
	if res.IsNull() || res.IsUndefined() {
		return ""
	}
	return res.String()
}

// Confirm displays a browser confirm dialog and returns true if the user clicks OK.
func Confirm(msg any) bool {
	return js.Global().Call("confirm", msg).Bool()
}
