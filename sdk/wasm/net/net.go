//go:build js && wasm

package net

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"

	"github.com/ZiplEix/stew/v2/sdk/wasm/state"
)

// IsLoading is a global signal to track network activity.
var IsLoading = state.New(false)

// awaitPromise converts a JS Promise into a Go result using channels.
func awaitPromise(promise js.Value) (js.Value, error) {
	resCh := make(chan js.Value)
	errCh := make(chan error)

	onSuccess := js.FuncOf(func(this js.Value, args []js.Value) any {
		// Prevent blocking the main loop
		go func() {
			resCh <- args[0]
		}()
		return nil
	})
	defer onSuccess.Release()

	onFailure := js.FuncOf(func(this js.Value, args []js.Value) any {
		go func() {
			errCh <- fmt.Errorf("JS Promise rejected: %s", args[0].String())
		}()
		return nil
	})
	defer onFailure.Release()

	promise.Call("then", onSuccess, onFailure)

	select {
	case res := <-resCh:
		return res, nil
	case err := <-errCh:
		return js.Undefined(), err
	case <-time.After(30 * time.Second):
		return js.Undefined(), fmt.Errorf("Network timeout (30s)")
	}
}

// Fetch performs a raw network request and returns the JS Response object.
func Fetch(url string, opts map[string]any) (js.Value, error) {
	IsLoading.Set(true)
	defer IsLoading.Set(false)

	jsOpts := js.Global().Get("Object").New()
	for k, v := range opts {
		jsOpts.Set(k, v)
	}

	promise := js.Global().Call("fetch", url, jsOpts)
	resp, err := awaitPromise(promise)
	if err != nil {
		return js.Undefined(), err
	}

	if !resp.Get("ok").Bool() {
		return js.Undefined(), fmt.Errorf("HTTP error: %d %s", resp.Get("status").Int(), resp.Get("statusText").String())
	}

	return resp, nil
}

// execRequest handles the common logic for Get, Post, etc.
func execRequest[T any](method string, url string, body any) (T, error) {
	var zero T
	opts := map[string]any{
		"method": method,
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
	}

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("failed to marshal body: %w", err)
		}
		opts["body"] = string(jsonBody)
	}

	resp, err := Fetch(url, opts)
	if err != nil {
		return zero, err
	}

	// Get JSON promise
	jsonPromise := resp.Call("json")
	jsonVal, err := awaitPromise(jsonPromise)
	if err != nil {
		return zero, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Stringify and unmarshal into Go struct
	jsonStr := js.Global().Get("JSON").Call("stringify", jsonVal).String()

	var result T
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return zero, fmt.Errorf("failed to unmarshal JSON into Go type: %w", err)
	}

	return result, nil
}

func Get[T any](url string) (T, error) {
	return execRequest[T]("GET", url, nil)
}

func Post[T any](url string, body any) (T, error) {
	return execRequest[T]("POST", url, body)
}

func Put[T any](url string, body any) (T, error) {
	return execRequest[T]("PUT", url, body)
}

func Patch[T any](url string, body any) (T, error) {
	return execRequest[T]("PATCH", url, body)
}

func Delete[T any](url string) (T, error) {
	return execRequest[T]("DELETE", url, nil)
}

// Watch polls a URL at a given interval and updates a Signal automatically.
func Watch[T any](url string, interval time.Duration) *state.Signal[T] {
	var zero T
	sig := state.New(zero)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		fetchOnce := func() {
			data, err := Get[T](url)
			if err == nil {
				sig.Set(data)
			}
		}

		// First immediate fetch
		fetchOnce()

		for range ticker.C {
			fetchOnce()
		}
	}()

	return sig
}
