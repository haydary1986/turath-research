package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/haydary1986/turath-research/backend/internal/config"
	"github.com/haydary1986/turath-research/backend/internal/handler"
	"github.com/haydary1986/turath-research/backend/internal/service/journalcheck"
	"github.com/haydary1986/turath-research/backend/internal/service/loginaudit"
	syncsvc "github.com/haydary1986/turath-research/backend/internal/service/sync"
)

// Run starts the HTTP server and blocks until shutdown.
func Run(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client) error {
	journals := journalcheck.New(
		journalcheck.NewPGStore(db),
		journalcheck.NewHTTPLookups(cfg.OpenAlex.APIKey, cfg.CrossRef.Mailto),
	)
	refreshCtx, stopRefresh := context.WithCancel(context.Background())
	defer stopRefresh()
	go journals.Run(refreshCtx)

	orchestrator := syncsvc.NewOrchestrator(db, cfg.CrossRef.Mailto, cfg.OpenAlex.APIKey)
	bulkSync := newBulkSync(cfg, db, orchestrator)
	// A bulk sync interrupted by a restart carries on from where it stopped.
	bulkSync.Resume(refreshCtx)

	// Login events are kept for loginaudit.RetentionDays; older ones go once a day.
	go loginaudit.RunRetention(refreshCtx, handler.PruneLoginEvents(db), 24*time.Hour, time.Now)

	router := newRouter(cfg, db, rdb, journals, orchestrator, bulkSync)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown channel.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
