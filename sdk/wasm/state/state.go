//go:build !js || !wasm

package state

// Signal represents a reactive value
type Signal[T any] struct {
	value T
}

// New creates a new reactive signal initialized with the given value.
func New[T any](initial T) *Signal[T] {
	return &Signal[T]{value: initial}
}

// Get returns the current value of the signal. In SSR context, this just returns the value.
func (s *Signal[T]) Get() T {
	return s.value
}

// Set updates the signal's value. In SSR context, this just sets the value.
func (s *Signal[T]) Set(val T) {
	s.value = val
}

// Effect registers a function to be called when dependencies change.
// In SSR context, it simply executes the function once safely.
func Effect(cb func()) {
	cb()
}
