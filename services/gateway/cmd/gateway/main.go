package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doanbenson/momentos/services/gateway/internal/api"
	"github.com/doanbenson/momentos/services/gateway/internal/business"
)

func main() {
	addr := env("GATEWAY_ADDR", ":8080")
	jwtCfg, err := api.JWTConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gateway: %v\n", err)
		os.Exit(1)
	}
	internalToken := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if internalToken == "" {
		fmt.Fprintln(os.Stderr, "gateway: INTERNAL_SERVICE_TOKEN is required")
		os.Exit(1)
	}
	momentURL := env("MOMENT_SERVICE_URL", "http://localhost:8081")
	mediaURL := env("MEDIA_SERVICE_URL", "http://localhost:8082")
	renderURL := env("RENDER_SERVICE_URL", "http://localhost:8083")

	bff := business.NewBFF(momentURL, mediaURL, renderURL, internalToken)
	mux := api.NewMux(api.Deps{BFF: bff, JWTConfig: jwtCfg})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("gateway listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
