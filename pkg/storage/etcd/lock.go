package etcd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rancher/opni/pkg/storage"
	"github.com/rancher/opni/pkg/storage/etcd/concurrencyx"
	"github.com/rancher/opni/pkg/storage/lock"
	"github.com/samber/lo"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type EtcdLockManager struct {
	client *clientv3.Client
	prefix string

	lg *slog.Logger
	// sessionMu sync.Mutex
	// session   *concurrency.Session
}

func NewEtcdLockManager(client *clientv3.Client, prefix string) (*EtcdLockManager, error) {
	lm := &EtcdLockManager{
		client: client,
		prefix: prefix,
		lg:     slog.Default().WithGroup("etcd-lock-manager"),
	}
	// if err := lm.renewSessionLocked(); err != nil {
	// 	return nil, fmt.Errorf("failed to create etcd client: %w", err)
	// }
	return lm, nil
}

// func (lm *EtcdLockManager) Session() *concurrency.Session {
// 	lm.sessionMu.Lock()
// 	defer lm.sessionMu.Unlock()
// 	if lm.session == nil {
// 		lm.renewSessionLocked()
// 	} else {
// 		select {
// 		case <-lm.session.Done():
// 			lm.renewSessionLocked()
// 		default:
// 		}
// 	}
// 	return lm.session
// }

// func (lm *EtcdLockManager) newSession() (*concurrency.Session, error) {
// 	session, err := concurrency.NewSession(lm.client, concurrency.WithTTL(mutexLeaseTtlSeconds))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create etcd client: %w", err)
// 	}
// 	return session, nil
// }

// Locker implements storage.LockManager.
func (lm *EtcdLockManager) Locker(key string, opts ...lock.LockOption) storage.Lock {
	options := lock.DefaultLockOptions(lm.client.Ctx())
	options.Apply(opts...)

	// m := concurrencyx.NewMutex(lm.newSession(), path.Join(lm.prefix, key), options.InitialValue)
	return &EtcdLock{
		client:  lm.client,
		mutex:   nil,
		options: options,
		prefix:  lm.prefix + "/",
		lg:      lm.lg.With("prefix", lm.prefix, "key", key),
		setup:   lo.ToPtr(uint32(0)),
	}
}

type EtcdLock struct {
	lg *slog.Logger

	sessionMu sync.Mutex
	client    *clientv3.Client
	session   *concurrency.Session

	mutex *concurrencyx.Mutex

	prefix string
	key    string

	// only accessed atomically
	setup *uint32

	startLock   lock.LockPrimitive
	startUnlock lock.LockPrimitive
	options     *lock.LockOptions
}

func (e *EtcdLock) Session() (*concurrency.Session, error) {
	e.sessionMu.Lock()
	defer e.sessionMu.Unlock()
	if e.session == nil {
		if err := e.renewSession(); err != nil {
			return nil, err
		}
	} else {
		select {
		case <-e.session.Done():
			if err := e.renewSession(); err != nil {
				return nil, err
			}
		default:
		}
	}
	return e.session, nil
}

func (e *EtcdLock) renewSession() error {
	session, err := concurrency.NewSession(e.client, concurrency.WithTTL(mutexLeaseTtlSeconds))
	if err != nil {
		return fmt.Errorf("failed to create etcd session: %w", err)
	}
	e.session = session
	return nil
}

var _ storage.Lock = (*EtcdLock)(nil)

// TODO : this doesn't handle setting up repeated mutexes, which the original implementation forbade for good reason
func (e *EtcdLock) setupMutex() error {
	e.lg.Info("setting up mutex and renewing session")
	if err := e.renewSession(); err != nil {
		return err
	}
	if atomic.CompareAndSwapUint32(e.setup, 0, 1) {
		e.lg.Debug("set mutex once and only once")
		e.mutex = concurrencyx.NewMutex(e.session, path.Join(e.prefix, e.key), e.options.InitialValue)
	}
	return nil
}

// TODO FIXME  this keeps trying if etcd is unreachable during the acquisition phase
// TODO : grpc.WaitForReady(true) will cause lock acquisitions to block forever if the endpoint it connects to is unreachabale
// https://github.com/etcd-io/etcd/issues/14631
func (e *EtcdLock) Lock(ctx context.Context) error {
	e.lg.Info("setting up mutex")
	if err := e.setupMutex(); err != nil {
		return err
	}

	// ctx := e.client.Ctx()
	// if e.options.AcquireContext != nil {
	// 	ctx = e.options.AcquireContext
	// }

	return e.mutex.Lock(ctx)
	// return e.startLock.Do(func() error {
	// 	ctxca, ca := context.WithCancelCause(e.client.Ctx())
	// 	signalAcquired := make(chan struct{})
	// 	defer close(signalAcquired)
	// 	var lockErr error
	// 	var mu sync.Mutex
	// 	go func() {
	// 		select {
	// 		case <-e.options.AcquireContext.Done():
	// 			mu.Lock()
	// 			lockErr = errors.Join(lockErr, lock.ErrAcquireLockCancelled)
	// 			mu.Unlock()
	// 			ca(lock.ErrAcquireLockCancelled)
	// 		case <-time.After(e.options.AcquireTimeout):
	// 			mu.Lock()
	// 			lockErr = errors.Join(lockErr, lock.ErrAcquireLockTimeout)
	// 			mu.Unlock()
	// 			ca(lock.ErrAcquireLockTimeout)
	// 		}
	// 	}()
	// 	err := e.mutex.Lock(ctxca)
	// 	mu.Lock()
	// 	err = errors.Join(lockErr, err)
	// 	mu.Unlock()
	// 	if err != nil {
	// 		e.mutex.Unlock(e.client.Ctx())
	// 		return err
	// 	}
	// 	atomic.StoreUint32(&e.acquired, 1)
	// 	return nil
	// })
}

func (e *EtcdLock) TryLock(ctx context.Context) (bool, error) {
	e.lg.Info("setting up mutex for try lock")

	if err := e.setupMutex(); err != nil {
		return false, err
	}
	err := e.mutex.TryLock(ctx)
	if err != nil {
		if errors.Is(err, concurrency.ErrLocked) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// TODO FIXME : this blocks until the unlock succeeds
func (e *EtcdLock) Unlock() error {
	if e.mutex == nil {
		return errors.New("mutex not acquired")
	}
	return e.mutex.Unlock(e.client.Ctx())
}

func (e *EtcdLock) Key() string {
	return strings.TrimPrefix(e.mutex.Key(), e.prefix)
}
