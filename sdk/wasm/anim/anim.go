//go:build js && wasm
package anim

import (
	"strings"
	"syscall/js"
	"time"
)

// Direction represents the starting point of a slide animation.
type Direction string

const (
	Left   Direction = "left"
	Right  Direction = "right"
	Top    Direction = "top"
	Bottom Direction = "bottom"
)

// Anim provides Web Animation API primitives for the Stew Wasm SDK.
type Anim struct{}

// Instance is the global singleton for animations.
var Instance = &Anim{}

// Animate is the low-level wrapper around Element.animate()
func (a *Anim) Animate(selector string, keyframes []map[string]any, options map[string]any) {
	el := js.Global().Get("document").Call("querySelector", selector)
	if !el.IsNull() {
		kfArr := js.Global().Get("Array").New()
		for _, kf := range keyframes {
			obj := js.Global().Get("Object").New()
			for k, v := range kf {
				obj.Set(k, v)
			}
			kfArr.Call("push", obj)
		}

		optObj := js.Global().Get("Object").New()
		for k, v := range options {
			optObj.Set(k, v)
		}

		el.Call("animate", kfArr, optObj)
	}
}

// FadeIn animates opacity from 0 to 1.
func (a *Anim) FadeIn(selector string, d time.Duration) {
	a.Animate(selector, []map[string]any{
		{"opacity": 0},
		{"opacity": 1},
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "ease-out",
		"fill":     "forwards",
	})
}

// FadeOut animates opacity from 1 to 0.
func (a *Anim) FadeOut(selector string, d time.Duration) {
	a.Animate(selector, []map[string]any{
		{"opacity": 1},
		{"opacity": 0},
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "ease-in",
		"fill":     "forwards",
	})
}

// SlideIn animates an element sliding in from a direction.
func (a *Anim) SlideIn(selector string, from Direction, d time.Duration) {
	transformFrom := ""
	switch from {
	case Left:
		transformFrom = "translateX(-100%)"
	case Right:
		transformFrom = "translateX(100%)"
	case Top:
		transformFrom = "translateY(-100%)"
	case Bottom:
		transformFrom = "translateY(100%)"
	}

	a.Animate(selector, []map[string]any{
		{"transform": transformFrom, "opacity": 0},
		{"transform": "translate(0)", "opacity": 1},
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "cubic-bezier(0.16, 1, 0.3, 1)", // Smooth spring-like ease
		"fill":     "forwards",
	})
}

// Shake creates a quick horizontal shake effect.
func (a *Anim) Shake(selector string, d time.Duration) {
	a.Animate(selector, []map[string]any{
		{"transform": "translateX(0)"},
		{"transform": "translateX(-5px)"},
		{"transform": "translateX(5px)"},
		{"transform": "translateX(-5px)"},
		{"transform": "translateX(5px)"},
		{"transform": "translateX(0)"},
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "linear",
	})
}

// Pulse creates a scale up and down effect.
func (a *Anim) Pulse(selector string, d time.Duration) {
	a.Animate(selector, []map[string]any{
		{"transform": "scale(1)"},
		{"transform": "scale(1.05)"},
		{"transform": "scale(1)"},
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "ease-in-out",
	})
}

// Transition performs a simple transition between two CSS states.
// fromCSS and toCSS should be in "prop: value; prop2: value" format.
func (a *Anim) Transition(selector string, fromCSS, toCSS string, d time.Duration) {
	parse := func(css string) map[string]any {
		res := make(map[string]any)
		parts := strings.Split(css, ";")
		for _, p := range parts {
			kv := strings.Split(p, ":")
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				// convert camelCase if needed, but Web Animations API accepts spinal-case strings for property names
				res[key] = strings.TrimSpace(kv[1])
			}
		}
		return res
	}

	a.Animate(selector, []map[string]any{
		parse(fromCSS),
		parse(toCSS),
	}, map[string]any{
		"duration": d.Milliseconds(),
		"easing":   "ease",
		"fill":     "forwards",
	})
}
