//go:build js && wasm
package js

import (
	"fmt"
	"syscall/js"
)

// JS provides interop primitives for the Stew Wasm SDK.
type JS struct{}

// Instance is the global singleton for JS interop.
var Instance = &JS{}

// Run executes a raw JavaScript code block in the global scope.
func (j *JS) Run(code string) {
	js.Global().Call("eval", code)
}

// Global returns a global JavaScript object by name.
func (j *JS) Global(name string) js.Value {
	return js.Global().Get(name)
}

// Set exposes a Go value or function to the JavaScript global scope.
func (j *JS) Set(name string, val any) {
	js.Global().Set(name, val)
}

// Eval evaluates a JavaScript expression and returns the result as a Go type.
func Eval[T any](code string) (T, error) {
	var res T
	val := js.Global().Call("eval", code)

	switch any(res).(type) {
	case string:
		res = any(val.String()).(T)
	case int:
		res = any(val.Int()).(T)
	case bool:
		res = any(val.Bool()).(T)
	case float64:
		res = any(val.Float()).(T)
	default:
		return res, fmt.Errorf("unsupported type for Eval: %T", res)
	}

	return res, nil
}

// Invoke calls a global JavaScript function with provided arguments.
func (j *JS) Invoke(funcName string, args ...any) (js.Value, error) {
	fn := js.Global().Get(funcName)
	if fn.IsUndefined() || fn.IsNull() {
		return js.Undefined(), fmt.Errorf("function %s is not defined in global scope", funcName)
	}
	return fn.Invoke(args...), nil
}

// IsLoaded checks if a script with the given URL is already in the document.
func (j *JS) IsLoaded(url string) bool {
	scripts := js.Global().Get("document").Call("querySelectorAll", fmt.Sprintf("script[src=\"%s\"]", url))
	return scripts.Get("length").Int() > 0
}

// Load injects a script tag into the document head and calls onLoad when ready.
func (j *JS) Load(url string, onLoad func()) {
	if j.IsLoaded(url) {
		if onLoad != nil {
			onLoad()
		}
		return
	}

	doc := js.Global().Get("document")
	script := doc.Call("createElement", "script")
	script.Set("src", url)

	if onLoad != nil {
		onLoadFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
			onLoad()
			return nil
		})
		script.Call("addEventListener", "load", onLoadFunc)
	}

	doc.Get("head").Call("appendChild", script)
}

// Unload removes a script tag from the document by its source URL.
func (j *JS) Unload(url string) {
	doc := js.Global().Get("document")
	scripts := doc.Call("querySelectorAll", fmt.Sprintf("script[src=\"%s\"]", url))
	length := scripts.Get("length").Int()
	for i := 0; i < length; i++ {
		s := scripts.Call("item", i)
		s.Get("parentNode").Call("removeChild", s)
	}
}
