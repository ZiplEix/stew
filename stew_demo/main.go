package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ZiplEix/stew/sdk/live"
	"github.com/a-h/templ"
)

func main() {
	mux := http.NewServeMux()

	// Main route to serve the template
	mux.Handle("/", templ.Handler(Page("Developer")))

	// Apply the Stew middleware only in dev
	var handler http.Handler = mux
	if os.Getenv("STEW_DEV") == "true" {
		fmt.Println("🛠️  Development mode: Stew Middleware enabled")
		handler = live.Middleware(mux)
	}

	port := ":8080"
	fmt.Printf("🚀 Server ready at http://localhost%s\n", port)

	if err := http.ListenAndServe(port, handler); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
