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

	"github.com/doanbenson/momentos/services/moment-service/internal/api"
	"github.com/doanbenson/momentos/services/moment-service/internal/business"
	"github.com/doanbenson/momentos/services/moment-service/internal/database"
	"github.com/doanbenson/momentos/services/moment-service/internal/media"
)

func main() {
	addr := env("MOMENT_SERVICE_ADDR", ":8081")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "moment-service: DATABASE_URL is required")
		os.Exit(1)
	}
	internalToken := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if internalToken == "" {
		fmt.Fprintln(os.Stderr, "moment-service: INTERNAL_SERVICE_TOKEN is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	store, err := database.NewStore(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "moment-service: db: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	mediaClient := media.NewClient(os.Getenv("MEDIA_SERVICE_URL"), internalToken)
	svc := business.NewService(store, mediaClient)

	mux := api.NewMux(api.Deps{Svc: svc, InternalToken: internalToken})
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("moment-service listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
