//go:build js && wasm
package event

import (
	"syscall/js"
)

// Event provides global event listener primitives for the Stew Wasm SDK.
type Event struct{}

// Instance is the global singleton for event management.
var Instance = &Event{}

// OnKeyDown registers a callback for a specific key press on the window.
func (e *Event) OnKeyDown(key string, cb func()) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("key").String() == key {
			cb()
		}
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "keydown", handler)
}

// OnKeyUp registers a callback for a specific key release on the window.
func (e *Event) OnKeyUp(key string, cb func()) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("key").String() == key {
			cb()
		}
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "keyup", handler)
}

// OnResize registers a callback for window resize events.
func (e *Event) OnResize(cb func(width, height int)) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		w := js.Global().Get("innerWidth").Int()
		h := js.Global().Get("innerHeight").Int()
		cb(w, h)
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "resize", handler)
}

// OnOnline registers a callback for when the browser goes online.
func (e *Event) OnOnline(cb func()) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		cb()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "online", handler)
}

// OnOffline registers a callback for when the browser goes offline.
func (e *Event) OnOffline(cb func()) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		cb()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "offline", handler)
}

// IsOnline returns true if the browser is currently online.
func (e *Event) IsOnline() bool {
	return js.Global().Get("navigator").Get("onLine").Bool()
}

// OnScroll registers a callback for window scroll events.
func (e *Event) OnScroll(cb func(top, left int)) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) any {
		top := js.Global().Get("scrollY").Int()
		left := js.Global().Get("scrollX").Int()
		cb(top, left)
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "scroll", handler)
}
