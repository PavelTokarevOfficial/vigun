package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ListenPipelineEvents(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, publish func([]byte)) {
	for ctx.Err() == nil {
		connection, err := pool.Acquire(ctx)
		if err != nil {
			waitForNotificationRetry(ctx)
			continue
		}
		if _, err = connection.Exec(ctx, "LISTEN pipeline_events"); err != nil {
			connection.Release()
			log.Error("pipeline event listener failed", "error", err)
			waitForNotificationRetry(ctx)
			continue
		}
		log.Info("pipeline event listener started")
		for ctx.Err() == nil {
			notification, waitErr := connection.Conn().WaitForNotification(ctx)
			if waitErr != nil {
				if ctx.Err() == nil {
					log.Warn("pipeline event listener disconnected", "error", waitErr)
				}
				break
			}
			publish([]byte(notification.Payload))
		}
		connection.Release()
		waitForNotificationRetry(ctx)
	}
}

func waitForNotificationRetry(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
	}
}
