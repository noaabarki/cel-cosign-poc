package main

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/noaabarki/cel-cosign-poc/pkg/mutationwebhook"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func main() {
	fmt.Println("Starting server ...")

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", mutationwebhook.HandleMutate)

	s := &http.Server{
		Addr:           ":8443",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	if err := s.ListenAndServeTLS("/run/secrets/tls/tls.crt", "/run/secrets/tls/tls.key"); err != nil {
		err = s.ListenAndServe()
		if err != nil {
			fmt.Println("Failed to start http server", err.Error())

		}
	}
}
