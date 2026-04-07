//go:build js && wasm
package cookies

import (
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

// Options represents the configuration for setting a cookie.
type Options struct {
	Expires  time.Time
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool // Note: Cannot be set from client-side JS, but included for completeness if used in later contexts.
	SameSite string
}

// Cookies provides cookie manipulation primitives for the Stew Wasm SDK.
type Cookies struct{}

// Instance is the global singleton for cookie management.
var Instance = &Cookies{}

// Set creates or updates a cookie with given options.
func (c *Cookies) Set(name, value string, opts Options) {
	var cookie strings.Builder
	cookie.WriteString(fmt.Sprintf("%s=%s", name, value))

	if !opts.Expires.IsZero() {
		cookie.WriteString(fmt.Sprintf("; Expires=%s", opts.Expires.UTC().Format(time.RFC1123)))
	}
	if opts.MaxAge > 0 {
		cookie.WriteString(fmt.Sprintf("; Max-Age=%d", opts.MaxAge))
	}
	if opts.Path != "" {
		cookie.WriteString(fmt.Sprintf("; Path=%s", opts.Path))
	} else {
		cookie.WriteString("; Path=/")
	}
	if opts.Domain != "" {
		cookie.WriteString(fmt.Sprintf("; Domain=%s", opts.Domain))
	}
	if opts.Secure {
		cookie.WriteString("; Secure")
	}
	if opts.SameSite != "" {
		cookie.WriteString(fmt.Sprintf("; SameSite=%s", opts.SameSite))
	}

	js.Global().Get("document").Set("cookie", cookie.String())
}

// Get returns the value of the cookie with specified name.
func (c *Cookies) Get(name string) string {
	all := js.Global().Get("document").Get("cookie").String()
	cookies := strings.Split(all, ";")
	for _, cookie := range cookies {
		parts := strings.Split(strings.TrimSpace(cookie), "=")
		if len(parts) == 2 && parts[0] == name {
			return parts[1]
		}
	}
	return ""
}

// Has checks if a cookie with specified name exists.
func (c *Cookies) Has(name string) bool {
	return c.Get(name) != ""
}

// Delete removes a cookie by setting its expiration date to the past.
func (c *Cookies) Delete(name string) {
	js.Global().Get("document").Set("cookie", fmt.Sprintf("%s=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT", name))
}

// GetAll returns all cookies as a map.
func (c *Cookies) GetAll() map[string]string {
	res := make(map[string]string)
	all := js.Global().Get("document").Get("cookie").String()
	if all == "" {
		return res
	}
	cookies := strings.Split(all, ";")
	for _, cookie := range cookies {
		parts := strings.Split(strings.TrimSpace(cookie), "=")
		if len(parts) == 2 {
			res[parts[0]] = parts[1]
		}
	}
	return res
}
