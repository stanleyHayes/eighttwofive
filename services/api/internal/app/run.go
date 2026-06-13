package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const shutdownTimeout = 10 * time.Second

// Run serves HTTP until ctx is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		a.logger.Info(
			"api listening",
			"port", a.cfg.Port,
			"env", a.cfg.Env,
			"email", a.cfg.EmailEnabled(),
			"uploads", a.cfg.UploadsEnabled(),
		)

		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		a.logger.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		err := a.server.Shutdown(shutdownCtx)
		if err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		return nil
	}
}
