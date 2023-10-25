package lock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrAcquireLockTimeout        = errors.New("acquiring lock error: timeout")
	// ErrAcquireLockRetryExceeded  = errors.New("acquiring lock error: retry limit exceeded")
	// ErrAcquireLockCancelled      = errors.New("acquiring lock error: context cancelled")
	// ErrAcquireLockConflict       = errors.New("acquiring lock error: request has conflicting lock value")
	// ErrLockNotFound              = errors.New("lock not found")
	// ErrAcquireUnlockTimeout      = errors.New("acquiring unlock error: timeout")
	// ErrAcquireUnlockCancelled    = errors.New("acquiring unlock error: cancelled")
	// ErrAcquireUnockRetryExceeded = errors.New("acquiring unlock error: retry limit exceeded")
	// ErrLockNotAcquired = errors.New("unlock failed: lock not acquired")

	ErrLockActionRequested = errors.New("lock action already requested")
)

var (
	DefaultRetryDelay     = 10 * time.Millisecond
	DefaultAcquireTimeout = 100 * time.Millisecond
	DefaultTimeout        = 10 * time.Second
)

// Modified sync.Once primitive
type LockPrimitive struct {
	done uint32
	m    sync.Mutex
}

func (l *LockPrimitive) Do(f func() error) error {
	if atomic.LoadUint32(&l.done) == 0 {
		return l.doSlow(f)
	}
	return ErrLockActionRequested
}

func (l *LockPrimitive) doSlow(f func() error) error {
	l.m.Lock()
	defer l.m.Unlock()
	if l.done == 0 {
		defer atomic.StoreUint32(&l.done, 1)
		return f()
	}
	return nil
}

type LockOptions struct {
	// Continually retries acquiring the lock for the lifetime of this context
	AcquireContext context.Context

	// An optional initial value set on the mutex key when it is created after
	// acquiring the lock
	InitialValue string
}

func DefaultLockOptions(acquireCtx context.Context) *LockOptions {
	return &LockOptions{
		AcquireContext: acquireCtx,
	}
}

func (o *LockOptions) Apply(opts ...LockOption) {
	for _, op := range opts {
		op(o)
	}
}

type LockOption func(o *LockOptions)

func WithAcquireContext(ctx context.Context) LockOption {
	return func(o *LockOptions) {
		o.AcquireContext = ctx
	}
}

func WithInitialValue(value string) LockOption {
	return func(o *LockOptions) {
		o.InitialValue = value
	}
}
