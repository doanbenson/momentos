package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/doanbenson/momentos/services/media-service/internal/api"
	"github.com/doanbenson/momentos/services/media-service/internal/business"
	"github.com/doanbenson/momentos/services/media-service/internal/database"
	"github.com/doanbenson/momentos/services/media-service/internal/storage"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal(err)
	}
	defer pool.Close()

	store := database.NewStore(pool)
	if cfg.AutoMigrate {
		if err := store.Migrate(ctx, cfg.MigrationsDir); err != nil {
			fatal(fmt.Errorf("migrate: %w", err))
		}
		log.Printf("migrations applied from %s", cfg.MigrationsDir)
	}
	if err := business.EnsureMVPSongPickSeeded(ctx, store); err != nil {
		fatal(err)
	}

	storageClient := storage.New(cfg.SupabaseURL, cfg.SupabaseSecretKey, cfg.StorageBucket)
	svc := business.NewService(store, storageClient)
	mux := api.NewMux(api.Deps{
		Service: svc,
		Auth:    &api.Auth{InternalToken: cfg.InternalServiceToken},
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("media-service listening on %s (bucket=%s)", cfg.Addr, cfg.StorageBucket)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

type config struct {
	Addr                   string
	DatabaseURL            string
	SupabaseURL            string
	SupabaseSecretKey    string
	StorageBucket        string
	InternalServiceToken string
	MigrationsDir        string
	AutoMigrate          bool
}

func loadConfig() (config, error) {
	cfg := config{
		Addr:                 env("MEDIA_SERVICE_ADDR", ":8082"),
		DatabaseURL:          env("DATABASE_URL", env("MEDIA_DATABASE_URL", "")),
		SupabaseURL:          env("SUPABASE_URL", ""),
		SupabaseSecretKey:    env("SUPABASE_SECRET_KEY", ""),
		StorageBucket:        env("MEDIA_STORAGE_BUCKET", "momentos-media"),
		InternalServiceToken: env("INTERNAL_SERVICE_TOKEN", ""),
		MigrationsDir:        env("MEDIA_MIGRATIONS_DIR", ""),
		AutoMigrate:          env("MEDIA_AUTO_MIGRATE", "true") != "false",
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL or MEDIA_DATABASE_URL is required")
	}
	if cfg.SupabaseURL == "" {
		return cfg, fmt.Errorf("SUPABASE_URL is required")
	}
	if cfg.SupabaseSecretKey == "" {
		return cfg, fmt.Errorf("SUPABASE_SECRET_KEY is required")
	}
	if cfg.InternalServiceToken == "" {
		return cfg, fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if cfg.MigrationsDir == "" {
		cfg.MigrationsDir = defaultMigrationsDir()
	}
	return cfg, nil
}

func defaultMigrationsDir() string {
	// Prefer ./migrations when running from services/media-service.
	candidates := []string{
		"migrations",
		filepath.Join("services", "media-service", "migrations"),
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "migrations"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "migrations"
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "media-service: %v\n", err)
	os.Exit(1)
}
