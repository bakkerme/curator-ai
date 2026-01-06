package retry

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type Config struct {
	Attempts  int
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Jitter    time.Duration
}

func Do(ctx context.Context, config Config, fn func() error) error {
	attempts := config.Attempts
	if attempts <= 0 {
		attempts = 1
	}
	baseDelay := config.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 200 * time.Millisecond
	}
	maxDelay := config.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 2 * time.Second
	}
	jitter := config.Jitter
	if jitter <= 0 {
		jitter = 100 * time.Millisecond
	}

	var lastErr error
	delay := baseDelay
	for attempt := 0; attempt < attempts; attempt++ {
		if err := fn(); err != nil {
			lastErr = err
			if attempt == attempts-1 {
				break
			}
			sleep := delay + time.Duration(rand.Int63n(int64(jitter)))
			if sleep > maxDelay {
				sleep = maxDelay
			}
			timer := time.NewTimer(sleep)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("retry failed: %w", lastErr)
}
