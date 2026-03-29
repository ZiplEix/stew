package live

import (
	"fmt"
	"net/http"
	"os"
	"sync"
)

var (
	clients = make(map[chan bool]bool)
	mu      sync.Mutex
)

func SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(w, ": ok\n\n")
	w.(http.Flusher).Flush()

	messageChan := make(chan bool)
	mu.Lock()
	clients[messageChan] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, messageChan)
		mu.Unlock()
	}()

	for {
		select {
		case <-messageChan:
			fmt.Fprintf(w, "data: reload\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func NotifyReload() {
	mu.Lock()
	defer mu.Unlock()
	for c := range clients {
		c <- true
	}
}

func init() {
	if os.Getenv("STEW_DEV") == "true" {
		go func() {
			stewMux := http.NewServeMux()
			stewMux.HandleFunc("/stew/trigger-reload", func(w http.ResponseWriter, r *http.Request) {
				NotifyReload()
				w.WriteHeader(http.StatusOK)
			})
			http.ListenAndServe("localhost:9876", stewMux)
		}()
	}
}
