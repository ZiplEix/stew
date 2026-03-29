package utils

import (
	"io"
	"sync"
)

type LinePrefixWriter struct {
	Prefix    string
	Output    io.Writer
	mu        sync.Mutex
	isNewLine bool
}

func (w *LinePrefixWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isNewLine && n == 0 {
		w.isNewLine = true
	}

	var output []byte
	for _, b := range p {
		if w.isNewLine {
			output = append(output, []byte(w.Prefix)...)
			w.isNewLine = false
		}
		output = append(output, b)
		if b == '\n' {
			w.isNewLine = true
		}
	}

	_, err = w.Output.Write(output)
	return len(p), err
}
