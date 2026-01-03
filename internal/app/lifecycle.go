package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// ShutdownFunc is a function called during shutdown.
// It receives a context that may be cancelled if shutdown times out.
type ShutdownFunc func(ctx context.Context) error

// Lifecycle manages application lifecycle including graceful shutdown.
// It handles OS signals and coordinates shutdown of registered components.
type Lifecycle struct {
	mu            sync.Mutex
	shutdownFuncs []ShutdownFunc
	shutdownCh    chan struct{}
	doneCh        chan struct{}
	timeout       time.Duration
	shutdownOnce  sync.Once
}

// NewLifecycle creates a new lifecycle manager with the specified shutdown timeout.
func NewLifecycle(timeout time.Duration) *Lifecycle {
	return &Lifecycle{
		shutdownCh: make(chan struct{}),
		doneCh:     make(chan struct{}),
		timeout:    timeout,
	}
}

// OnShutdown registers a function to be called during shutdown.
// Functions are called in reverse order of registration (LIFO).
func (l *Lifecycle) OnShutdown(fn ShutdownFunc) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.shutdownFuncs = append(l.shutdownFuncs, fn)
}

// WaitForSignal blocks until SIGINT or SIGTERM is received,
// or until the shutdown channel is closed programmatically.
func (l *Lifecycle) WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		return sig
	case <-l.shutdownCh:
		return nil
	}
}

// Shutdown initiates graceful shutdown, calling all registered shutdown
// functions in reverse order of registration. Returns the last error
// encountered, if any.
func (l *Lifecycle) Shutdown() error {
	var lastErr error

	l.shutdownOnce.Do(func() {
		// Signal shutdown has started
		close(l.shutdownCh)

		ctx, cancel := context.WithTimeout(context.Background(), l.timeout)
		defer cancel()

		l.mu.Lock()
		funcs := make([]ShutdownFunc, len(l.shutdownFuncs))
		copy(funcs, l.shutdownFuncs)
		l.mu.Unlock()

		// Call shutdown functions in reverse order (LIFO)
		for i := len(funcs) - 1; i >= 0; i-- {
			if err := funcs[i](ctx); err != nil {
				lastErr = err
			}
		}

		close(l.doneCh)
	})

	return lastErr
}

// Done returns a channel that's closed when shutdown is complete.
func (l *Lifecycle) Done() <-chan struct{} {
	return l.doneCh
}

// ShutdownCh returns a channel that's closed when shutdown starts.
func (l *Lifecycle) ShutdownCh() <-chan struct{} {
	return l.shutdownCh
}

// IsShuttingDown returns true if shutdown has been initiated.
func (l *Lifecycle) IsShuttingDown() bool {
	select {
	case <-l.shutdownCh:
		return true
	default:
		return false
	}
}

// Timeout returns the configured shutdown timeout.
func (l *Lifecycle) Timeout() time.Duration {
	return l.timeout
}
