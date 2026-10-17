package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/doanbenson/momentos/services/render-service/internal/api"
	"github.com/doanbenson/momentos/services/render-service/internal/business"
	"github.com/doanbenson/momentos/services/render-service/internal/database"
	"github.com/doanbenson/momentos/services/render-service/internal/ffmpeg"
)

func main() {
	addr := env("RENDER_SERVICE_ADDR", ":8083")
	ffmpegBin := env("RENDER_FFMPEG_PATH", "ffmpeg")
	editAudio := env("RENDER_EDIT_AUDIO_PATH", "")
	concurrency := envInt("RENDER_WORKER_CONCURRENCY", 1)

	store := database.NewStore()
	runner := &ffmpeg.Runner{Bin: ffmpegBin}
	svc := &business.Service{
		Store:         store,
		Runner:        runner,
		EditAudioPath: editAudio,
		// StorageUpload / NotifyMoment: wire when Supabase + moment-service HTTP exist.
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	broker := business.NewMomentBrokerFromEnv()
	if broker != nil {
		svc.NotifyMoment = func(job *database.Job) error {
			if job.Status == database.StatusSucceeded {
				return broker.CompleteSuccess(context.Background(), job)
			}
			return broker.CompleteFailure(context.Background(), job.ID, "render_failed", job.LastError)
		}
	}

	worker := &business.Worker{
		Service:     svc,
		Concurrency: concurrency,
		PollEvery:   2 * time.Second,
		PollBroker: func(c context.Context) error {
			if broker == nil {
				return nil
			}
			return broker.PollClaim(c, store)
		},
	}
	go worker.Start(ctx)

	mux := api.NewMux(api.Deps{Service: svc, Runner: runner})
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("render-service listening on %s (ffmpeg=%s concurrency=%d)", addr, ffmpegBin, concurrency)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
