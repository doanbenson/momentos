package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/doanbenson/momentos/services/render-service/internal/api"
)

func main() {
	addr := env("RENDER_SERVICE_ADDR", ":8083")
	mux := api.NewMux()
	log.Printf("render-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "render-service: %v\n", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
