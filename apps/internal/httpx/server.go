package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Run starts an HTTP server with graceful shutdown. The server listens on
// addr with handler until ctx is cancelled (typically a SIGINT/SIGTERM
// delivered via signal.NotifyContext), at which point in-flight requests
// have shutdownTimeout to complete before the server is forced closed.
//
// Returns nil on clean shutdown, the listen error if the server failed to
// bind or accept, or the shutdown error if draining exceeded the timeout.
func Run(ctx context.Context, addr string, handler http.Handler, shutdownTimeout time.Duration) error {
	srv := &http.Server{Addr: addr, Handler: handler}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining", "timeout", shutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed", "err", err)
			return err
		}
		slog.Info("server stopped cleanly")
		return nil
	}
}
