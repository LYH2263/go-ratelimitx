package ratelimitx

import (
	"context"
	"time"
)

// waitDuration 阻塞至 d 结束或 ctx 被取消。
// ctx 先结束（含已取消）时返回 ctx.Err()，不会真正睡眠，取消可打断等待。
func waitDuration(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
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
