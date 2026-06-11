package leader

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandle satisfies lockHandle for testing.
type mockHandle struct{}

func (h *mockHandle) sendHeartbeat(_ context.Context) error { return nil }
func (h *mockHandle) release() error                        { return nil }

// failingHandle succeeds for failAfter heartbeats, then returns err.
type failingHandle struct {
	err       error
	failAfter int32
	beats     atomic.Int32
}

func (h *failingHandle) sendHeartbeat(_ context.Context) error {
	if h.beats.Add(1) > h.failAfter {
		return h.err
	}
	return nil
}

func (h *failingHandle) release() error { return nil }

// heartbeatFailLocker acquires successfully but returns a handle that
// fails after a configured number of heartbeats.
type heartbeatFailLocker struct {
	handleErr      error
	failAfter      int32
	acquireCount   atomic.Int32
	reacquireErr   error
	reacquireAfter int32 // fail reacquisition after this many acquires (0 = always succeed)
}

func (l *heartbeatFailLocker) acquireContext(_ context.Context, _ string) (lockHandle, error) {
	n := l.acquireCount.Add(1)
	if l.reacquireAfter > 0 && n > l.reacquireAfter {
		return nil, l.reacquireErr
	}
	return &failingHandle{err: l.handleErr, failAfter: l.failAfter}, nil
}

// failingLocker is a lockClient that fails a configurable number of times
// before succeeding. Set failCount to -1 to always fail.
type failingLocker struct {
	err       error
	failCount int32
	calls     atomic.Int32
}

func (f *failingLocker) acquireContext(_ context.Context, _ string) (lockHandle, error) {
	n := f.calls.Add(1)
	if f.failCount < 0 || n <= f.failCount {
		return nil, f.err
	}
	return &mockHandle{}, nil
}

func newTestElector(ctx context.Context, locker lockClient, threshold int32) *LeaderElector {
	e := &LeaderElector{
		ctx:                ctx,
		lockName:           "test-health",
		heartbeatFreq:      50 * time.Millisecond,
		locker:             locker,
		done:               make(chan struct{}),
		unhealthyThreshold: threshold,
		retryBackoff:       &backoff{current: time.Millisecond, max: time.Millisecond},
	}
	e.consecutiveFailures.Store(threshold)
	go e.run()
	return e
}

func TestHealthyStartsUnhealthy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := &failingLocker{
		err:       errors.New("connection refused"),
		failCount: -1,
	}

	e := newTestElector(ctx, locker, 3)

	assert.False(t, e.Healthy(), "should start unhealthy before first acquisition")

	require.Eventually(t, func() bool {
		return locker.calls.Load() >= 3
	}, 2*time.Second, 5*time.Millisecond, "should have attempted acquisition at least 3 times")

	assert.False(t, e.Healthy(), "should remain unhealthy with sustained failures")

	cancel()
	_ = e.Wait()
}

func TestHealthyRecoveryAfterAcquisitionSucceeds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := &failingLocker{
		err:       errors.New("connection refused"),
		failCount: 4,
	}

	e := newTestElector(ctx, locker, 3)

	assert.False(t, e.Healthy(), "should start unhealthy")

	require.Eventually(t, func() bool {
		return e.Healthy()
	}, 2*time.Second, 5*time.Millisecond, "should recover to healthy after successful acquisition")

	assert.Equal(t, int32(0), e.consecutiveFailures.Load(), "consecutive failures should be reset")

	cancel()
	_ = e.Wait()
}

func TestHealthyBecomesHealthyOnFirstSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := &failingLocker{
		err:       errors.New("connection refused"),
		failCount: 0, // succeeds on first call
	}

	e := newTestElector(ctx, locker, 3)

	assert.False(t, e.Healthy(), "should start unhealthy")

	require.Eventually(t, func() bool {
		return e.Healthy()
	}, 2*time.Second, 5*time.Millisecond, "should become healthy after first successful acquisition")

	assert.Equal(t, int32(0), e.consecutiveFailures.Load())

	cancel()
	_ = e.Wait()
}

func TestHealthyThresholdBoundary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locker := &failingLocker{
		err:       errors.New("db unavailable"),
		failCount: -1,
	}

	e := newTestElector(ctx, locker, 2)

	assert.False(t, e.Healthy(), "should start unhealthy (threshold=2)")

	// Continued failures keep it unhealthy
	require.Eventually(t, func() bool {
		return locker.calls.Load() >= 2
	}, 2*time.Second, 5*time.Millisecond, "should have attempted acquisition")

	assert.False(t, e.Healthy(), "should remain unhealthy with sustained failures")

	cancel()
	_ = e.Wait()
}

func TestHealthyLoseLockBecomesUnhealthy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locker := &heartbeatFailLocker{
		handleErr:      errors.New("heartbeat lost"),
		failAfter:      2,              // heartbeat fails after 2 successful beats
		reacquireErr:   errors.New("db down"),
		reacquireAfter: 1,              // first acquire succeeds, reacquisitions fail
	}

	e := newTestElector(ctx, locker, 3)

	// Should become healthy after first successful acquisition
	require.Eventually(t, func() bool {
		return e.Healthy()
	}, 2*time.Second, 5*time.Millisecond, "should become healthy after lock acquisition")

	// After heartbeat loss + 3 failed reacquisitions → unhealthy
	require.Eventually(t, func() bool {
		return !e.Healthy()
	}, 5*time.Second, 10*time.Millisecond, "should become unhealthy after losing lock and failing reacquisition")

	cancel()
	_ = e.Wait()
}
