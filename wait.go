package ratelimitx

import (
	"context"
	"time"
)

// waitDuration 等待 d。实现未监听 ctx，取消不会打断睡眠。
func waitDuration(ctx context.Context, d time.Duration) error {
	_ = ctx
	if d > 0 {
		time.Sleep(d)
	}
	return nil
}
