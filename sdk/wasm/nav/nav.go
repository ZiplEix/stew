//go:build js && wasm
package nav

import (
	"net/url"
	"syscall/js"
)

// Nav provides navigation primitives for the Stew Wasm SDK.
type Nav struct {
	URL   string
	Query url.Values
}

// Instance is the global singleton for navigation.
var Instance = &Nav{}

func init() {
	updateState()

	// Listen for popstate (back/forward) to update state
	onPopState := js.FuncOf(func(this js.Value, args []js.Value) any {
		updateState()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "popstate", onPopState)
}

func updateState() {
	loc := js.Global().Get("location")
	Instance.URL = loc.Get("pathname").String()

	// Parse Query parameters
	search := loc.Get("search").String()
	if search != "" {
		q, _ := url.ParseQuery(search[1:]) // strip '?'
		Instance.Query = q
	} else {
		Instance.Query = make(url.Values)
	}
}

// To performs a standard browser navigation (full page reload).
func (n *Nav) To(target string) {
	js.Global().Get("location").Set("href", target)
}

// Morph performs a smooth navigation using HTMX and Idiomorph if available.
// It falls back to a standard navigation if HTMX is not present.
func (n *Nav) Morph(target string) {
	htmx := js.Global().Get("htmx")
	if !htmx.IsUndefined() && !htmx.IsNull() {
		// htmx.ajax(method, url, targetSelector)
		htmx.Call("ajax", "GET", target, "body")
	} else {
		n.To(target)
	}
}

// Replace replaces the current history entry with a new URL.
func (n *Nav) Replace(target string) {
	js.Global().Get("location").Call("replace", target)
}

// Back goes back in the browser history.
func (n *Nav) Back() {
	js.Global().Get("history").Call("back")
}

// Forward goes forward in the browser history.
func (n *Nav) Forward() {
	js.Global().Get("history").Call("forward")
}

// Reload reloads the current page.
func (n *Nav) Reload() {
	js.Global().Get("location").Call("reload")
}
