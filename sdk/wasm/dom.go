//go:build js && wasm

package wasm

import (
	"encoding/json"
	"syscall/js"

	"github.com/ZiplEix/stew/v2/sdk/stew"
	"github.com/ZiplEix/stew/v2/sdk/wasm/state"
)

var document = js.Global().Get("document")

// isStarted tracks if the Wasm loop is running
var isStarted bool

// BindContent is maintained for legacy bindings that aren't reactive functions.
// It uses Idiomorph to set the content initially and then does a basic reactive wrap.
func BindContent(id string, ptr *string) {
	state.Effect(func() {
		el := document.Call("getElementById", id)
		if !el.IsNull() && !el.IsUndefined() {
			el.Set("innerHTML", *ptr)
		}
	})
}

// GetElement is a shorthand for document.getElementById(id)
func GetElement(id string) js.Value {
	return document.Call("getElementById", id)
}

// BindInput binds an input element to a string pointer (legacy).
func BindInput(id string, ptr *string) {
	el := document.Call("getElementById", id)
	if el.IsNull() || el.IsUndefined() {
		return
	}

	// Update DOM when ptr changes (requires manual trigger if not a signal,
	// but Effect helps in the generic reactivity loop context).
	state.Effect(func() {
		if el.Get("value").String() != *ptr {
			el.Set("value", *ptr)
		}
	})

	// Update ptr when DOM changes
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		*ptr = el.Get("value").String()
		return nil
	})
	el.Call("addEventListener", "input", cb)
}

// BindBlock binds a render function to a DOM element.
// It uses Idiomorph to morph the element's content whenever signal dependencies change.
func BindBlock(id string, render func() string) {
	state.Effect(func() {
		newHTML := render()
		el := document.Call("getElementById", id)
		if !el.IsNull() && !el.IsUndefined() {
			idiomorph := js.Global().Get("Idiomorph")
			if !idiomorph.IsUndefined() && !idiomorph.IsNull() {
				opts := js.Global().Get("Object").New()
				opts.Set("morphStyle", "innerHTML")
				idiomorph.Call("morph", el, newHTML, opts)
			} else {
				el.Set("innerHTML", newHTML)
			}
		}
	})
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

// GetPageData retrieves and unmarshals the page data injected by the server.
func GetPageData() stew.PageData {
	jsonStr := GetPageDataJSON()
	var data stew.PageData
	json.Unmarshal([]byte(jsonStr), &data)
	return data
}

// StartReactivityLoop blocks the main Wasm thread so it doesn't exit.
func StartReactivityLoop() {
	if isStarted {
		return
	}
	isStarted = true
	// Block main Wasm thread forever since reactivity is now event/signal driven.
	select {}
}
