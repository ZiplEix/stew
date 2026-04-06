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

// Range returns a slice of integers from start to end (inclusive)
func Range(start, end int) []int {
	if start > end {
		return nil
	}
	res := make([]int, end-start+1)
	for i := range res {
		res[i] = start + i
	}
	return res
}
