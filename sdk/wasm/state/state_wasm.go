//go:build js && wasm

package state

import (
	"sync"
)

var (
	currentContext *effect
	mu             sync.Mutex
)

// effect struct tracks a reactive execution context
type effect struct {
	cb func()
}

// Effect registers a function to be called immediately, and whenever its signal dependencies change.
func Effect(cb func()) {
	e := &effect{
		cb: cb,
	}

	mu.Lock()
	prevContext := currentContext
	currentContext = e
	mu.Unlock()

	cb() // Execute to establish initial dependencies

	mu.Lock()
	currentContext = prevContext
	mu.Unlock()
}

// Signal represents a reactive value
type Signal[T any] struct {
	value       T
	subscribers []*effect
}

// New creates a new reactive signal
func New[T any](initial T) *Signal[T] {
	return &Signal[T]{
		value: initial,
	}
}

// Get returns the value and tracks the current executing Effect (if any) as a dependency.
func (s *Signal[T]) Get() T {
	mu.Lock()
	defer mu.Unlock()

	if currentContext != nil {
		// Prevent duplicate subscriptions
		alreadySubscribed := false
		for _, sub := range s.subscribers {
			if sub == currentContext {
				alreadySubscribed = true
				break
			}
		}
		if !alreadySubscribed {
			s.subscribers = append(s.subscribers, currentContext)
		}
	}
	return s.value
}

// Set updates the value and notifies all subscribed Effects to re-run.
func (s *Signal[T]) Set(val T) {
	mu.Lock()
	s.value = val
	subs := make([]*effect, len(s.subscribers))
	copy(subs, s.subscribers)
	mu.Unlock()

	for _, sub := range subs {
		mu.Lock()
		prevContext := currentContext
		currentContext = sub
		mu.Unlock()

		sub.cb()

		mu.Lock()
		currentContext = prevContext
		mu.Unlock()
	}
}
