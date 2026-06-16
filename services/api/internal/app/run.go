package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	shutdownTimeout = 10 * time.Second
	// holdSweepInterval bounds how often expired slot holds are released in the
	// background. Holds last 45m, so a minute-scale sweep frees slots promptly
	// without hammering Mongo or Paystack.
	holdSweepInterval = 5 * time.Minute
)

// Run serves HTTP until ctx is cancelled, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go a.sweepHolds(ctx)

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

// sweepHolds releases lapsed slot holds on a timer until ctx is cancelled. It
// runs an immediate sweep on boot so holds left over from a previous process
// are cleaned up without waiting a full interval.
func (a *App) sweepHolds(ctx context.Context) {
	a.sweeper.ReleaseExpiredHolds(ctx)

	ticker := time.NewTicker(holdSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sweeper.ReleaseExpiredHolds(ctx)
		}
	}
}
