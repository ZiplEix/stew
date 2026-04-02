package stew

import (
	"net/http"
	"net/url"
)

type PageData struct {
	URL     string
	Params  map[string]string
	Query   url.Values
	Request *http.Request
	Store   map[string]any
}
