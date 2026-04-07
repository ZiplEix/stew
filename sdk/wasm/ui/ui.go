//go:build js && wasm
package ui

import (
	"syscall/js"
)

// UI provides high-level UI manipulation primitives for the Stew Wasm SDK.
type UI struct{}

// Instance is the global singleton for UI manipulation.
var Instance = &UI{}

// Title changes the document title.
func (u *UI) Title(text string) {
	js.Global().Get("document").Set("title", text)
}

// Focus sets focus to the element matching the selector.
func (u *UI) Focus(selector string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Call("focus")
	}
}

// ScrollTo scrolls the view to the element matching the selector.
func (u *UI) ScrollTo(selector string, smooth bool) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		behavior := "instant"
		if smooth {
			behavior = "smooth"
		}
		options := js.Global().Get("Object").New()
		options.Set("behavior", behavior)
		options.Set("block", "start")
		el.Call("scrollIntoView", options)
	}
}

// AddClass adds a CSS class to the element matching the selector.
func (u *UI) AddClass(selector string, className string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Get("classList").Call("add", className)
	}
}

// RemoveClass removes a CSS class from the element matching the selector.
func (u *UI) RemoveClass(selector string, className string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Get("classList").Call("remove", className)
	}
}

// ToggleClass toggles a CSS class on the element matching the selector.
func (u *UI) ToggleClass(selector string, className string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Get("classList").Call("toggle", className)
	}
}

// HasClass checks if the element matching the selector has a specific CSS class.
func (u *UI) HasClass(selector string, className string) bool {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		return el.Get("classList").Call("contains", className).Bool()
	}
	return false
}

// GetValue returns the value of an input element.
func (u *UI) GetValue(selector string) string {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		return el.Get("value").String()
	}
	return ""
}

// SetValue sets the value of an input element.
func (u *UI) SetValue(selector string, value string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Set("value", value)
	}
}

// SetStyle sets a CSS property on the element matching the selector.
func (u *UI) SetStyle(selector string, property string, value string) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		el.Get("style").Set(property, value)
	}
}
