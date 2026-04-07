//go:build js && wasm

package data

import (
	"encoding/json"

	"github.com/ZiplEix/stew/v2/sdk/wasm"
)

// WasmPageData represents the subset of PageData accessible in the Wasm context.
type WasmPageData struct {
	URL    string              `json:"url"`
	Params map[string]string   `json:"params"`
	Query  map[string][]string `json:"query"`
	Store  map[string]any      `json:"store"`
}

// Data is the global instance of the current page's data, automatically initialized at startup.
var Data WasmPageData

func init() {
	raw := wasm.GetPageDataJSON()
	if raw != "" && raw != "{}" {
		_ = json.Unmarshal([]byte(raw), &Data)
	}
}
