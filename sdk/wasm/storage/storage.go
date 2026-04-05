//go:build js && wasm
package storage

import (
	"syscall/js"
	"github.com/ZiplEix/stew/sdk/wasm"
)

// Store represents a browser storage area (Local or Session).
type Store struct {
	jsStore js.Value
}

// Get retrieves a string value from the store by its key.
func (s *Store) Get(key string) string {
	val := s.jsStore.Call("getItem", key)
	if val.IsNull() {
		return ""
	}
	return val.String()
}

// Set saves a string value in the store for the given key.
func (s *Store) Set(key string, value string) {
	s.jsStore.Call("setItem", key, value)
}

// Remove deletes a key-value pair from the store.
func (s *Store) Remove(key string) {
	s.jsStore.Call("removeItem", key)
}

// Clear removes all key-value pairs from the store.
func (s *Store) Clear() {
	s.jsStore.Call("clear")
}

// Bind synchronizes a Go string pointer with a storage key.
// It loads the initial value from storage and then monitors the pointer for changes.
func (s *Store) Bind(ptr *string, key string) {
	// Initial load
	if val := s.Get(key); val != "" {
		*ptr = val
	}

	lastValue := *ptr

	// Reactive sync Go -> Storage
	wasm.OnUpdate(func() {
		if *ptr != lastValue {
			lastValue = *ptr
			s.Set(key, *ptr)
		}
	})

	// Optional: Storage -> Go sync (only for LocalStorage across tabs)
	if s.jsStore.Equal(js.Global().Get("localStorage")) {
		onStorageChange := js.FuncOf(func(this js.Value, args []js.Value) any {
			event := args[0]
			if event.Get("key").String() == key {
				newValue := event.Get("newValue").String()
				if newValue != *ptr {
					*ptr = newValue
					lastValue = newValue
				}
			}
			return nil
		})
		js.Global().Get("window").Call("addEventListener", "storage", onStorageChange)
	}
}

// Storage contains the Local and Session stores.
type Storage struct {
	Local   *Store
	Session *Store
}

// Instance is the global singleton for storage.
var Instance = &Storage{
	Local:   &Store{jsStore: js.Global().Get("localStorage")},
	Session: &Store{jsStore: js.Global().Get("sessionStorage")},
}
