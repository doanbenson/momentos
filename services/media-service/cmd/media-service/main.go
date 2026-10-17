package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/doanbenson/momentos/services/media-service/internal/api"
)

func main() {
	addr := env("MEDIA_SERVICE_ADDR", ":8082")
	mux := api.NewMux()
	log.Printf("media-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "media-service: %v\n", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
