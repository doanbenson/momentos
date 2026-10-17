package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/doanbenson/momentos/services/moment-service/internal/api"
)

func main() {
	addr := env("MOMENT_SERVICE_ADDR", ":8081")
	mux := api.NewMux()
	log.Printf("moment-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "moment-service: %v\n", err)
		os.Exit(1)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
