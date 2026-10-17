package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/doanbenson/momentos/services/gateway/internal/api"
)

func main() {
	addr := env("GATEWAY_ADDR", ":8080")
	mux := api.NewMux()
	log.Printf("gateway listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
