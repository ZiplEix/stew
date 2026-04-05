package stew

import (
	"net/http"
	"net/url"
)

type PageData struct {
	URL     string            `json:"url"`
	Params  map[string]string `json:"params"`
	Query   url.Values        `json:"query"`
	Request *http.Request     `json:"-"`
	Store   map[string]any    `json:"store"`
}
