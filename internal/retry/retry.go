package retry

import (
	"context"
	"errors"
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

// permanentError marks an error as non-retryable for retry.Do.
type permanentError struct {
	err error
}

func (e permanentError) Error() string {
	return e.err.Error()
}

func (e permanentError) Unwrap() error {
	return e.err
}

// Permanent wraps an error to signal retry.Do to stop retrying immediately.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent reports whether an error was marked as non-retryable.
func IsPermanent(err error) bool {
	var p permanentError
	return err != nil && errors.As(err, &p)
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
			if IsPermanent(err) {
				return err
			}
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
