package ratelimitx

import (
	"context"
	"time"
)

// waitDuration 等待 d，但 ctx 取消时立即返回 ctx.Err()。
func waitDuration(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
