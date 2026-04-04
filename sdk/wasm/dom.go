//go:build js && wasm

package wasm

import (
	"syscall/js"
)

var document = js.Global().Get("document")

type valueBinding struct {
	id        string
	ptr       *string
	lastValue string
}

type contentBinding struct {
	id        string
	ptr       *string
	lastValue string
}

var (
	valBindings []valueBinding
	cntBindings []contentBinding
	updateHooks []func()
	isStarted   bool
)

// OnUpdate registers a computed callback executed right before the DOM diffing every frame.
func OnUpdate(cb func()) {
	updateHooks = append(updateHooks, cb)
}

// BindInput binds a pointer to a string to an input element's value.
// It sets up two-way data binding (DOM -> Go, and Go -> DOM via Dirty Checker).
func BindInput(id string, ptr *string) {
	el := document.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}

	// Update DOM to reflect initial Go value
	el.Set("value", *ptr)

	// DOM -> Go hook
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		*ptr = el.Get("value").String()
		return nil
	})
	el.Call("addEventListener", "input", cb)

	// Register for Go -> DOM dirty checking
	valBindings = append(valBindings, valueBinding{id: id, ptr: ptr, lastValue: *ptr})
}

// BindContent binds a pointer to a string to an element's innerHTML.
// It sets up one-way data binding (Go -> DOM via Dirty Checker).
func BindContent(id string, ptr *string) {
	el := document.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}

	el.Set("innerHTML", *ptr)
	cntBindings = append(cntBindings, contentBinding{id: id, ptr: ptr, lastValue: *ptr})
}

// OnEvent attaches an event listener to an element by its ID.
func OnEvent(id string, eventName string, handler func()) {
	el := document.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		handler()
		return nil
	})
	el.Call("addEventListener", eventName, cb)
}

// OnClick is a shorthand for attaching a click event listener to an element by its ID.
func OnClick(id string, handler func()) {
	OnEvent(id, "click", handler)
}

// GetPageDataJSON retrieves the serialized page data payload injected by the server.
func GetPageDataJSON() string {
	script := document.Call("getElementById", "stew-pagedata")
	if script.IsNull() || script.IsUndefined() {
		return "{}"
	}
	return script.Get("textContent").String()
}

// StartReactivityLoop initiates the requestAnimationFrame dirty checking loop.
func StartReactivityLoop() {
	if isStarted {
		return
	}
	isStarted = true

	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) any {
		// Run user computed updates first
		for _, hook := range updateHooks {
			hook()
		}

		// Check value bindings (Go -> DOM diff)
		for i := range valBindings {
			if *valBindings[i].ptr != valBindings[i].lastValue {
				valBindings[i].lastValue = *valBindings[i].ptr
				el := document.Call("getElementById", valBindings[i].id)
				if !el.IsNull() && !el.IsUndefined() {
					// Avoid overwriting if user is currently typing to prevent cursor jump
					// Simplification: just overwrite.
					el.Set("value", *valBindings[i].ptr)
				}
			}
		}

		// Check content bindings (Go -> DOM diff)
		for i := range cntBindings {
			if *cntBindings[i].ptr != cntBindings[i].lastValue {
				cntBindings[i].lastValue = *cntBindings[i].ptr
				el := document.Call("getElementById", cntBindings[i].id)
				if !el.IsNull() && !el.IsUndefined() {
					el.Set("innerHTML", *cntBindings[i].ptr)
				}
			}
		}

		js.Global().Call("requestAnimationFrame", cb)
		return nil
	})
	js.Global().Call("requestAnimationFrame", cb)

	// Block main Wasm thread
	select {}
}
