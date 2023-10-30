package jetstream

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rancher/opni/pkg/storage"
	"github.com/rancher/opni/pkg/storage/lock"
)

func newLease(key string) *nats.StreamConfig {
	return &nats.StreamConfig{
		Name:         key,
		Retention:    nats.InterestPolicy,
		Subjects:     []string{fmt.Sprintf("%s.lease.*", key)},
		MaxConsumers: 1,
	}
}

type Lock struct {
	key  string
	uuid string

	retryDelay   time.Duration
	lockValidity time.Duration

	js   nats.JetStreamContext
	sub  *nats.Subscription
	msgQ chan *nats.Msg
	*lock.LockOptions

	signalUnlock chan struct{}
	done         chan struct{}

	lg *slog.Logger
}

var _ storage.Lock = (*Lock)(nil)

func NewLock(js nats.JetStreamContext, key string, options *lock.LockOptions) *Lock {
	return &Lock{
		key:          key,
		js:           js,
		uuid:         uuid.New().String(),
		msgQ:         make(chan *nats.Msg, 16),
		LockOptions:  options,
		retryDelay:   time.Second,
		lockValidity: time.Second * 60,
		lg:           slog.Default(),
		signalUnlock: make(chan struct{}),
	}
}

func (l *Lock) Lock(ctx context.Context) (chan struct{}, error) {
	return l.lock(ctx)
}

func (l *Lock) lock(ctx context.Context) (chan struct{}, error) {
	tTicker := time.NewTicker(l.retryDelay)
	defer tTicker.Stop()

	var lockErr error
	for {
		select {
		case <-tTicker.C:
			done, err := l.doLock()
			if err == nil {
				l.done = done
				return done, nil
			}
			lockErr = err
		case <-ctx.Done():
			return nil, errors.Join(l.AcquireContext.Err(), lockErr)
		}
	}
}

func (l *Lock) doLock() (chan struct{}, error) {
	l.lg.With("key", l.key).Debug("trying to acquire lock")
	var err error
	if _, err := l.js.AddStream(newLease(l.key)); err != nil {
		return nil, err
	}
	cfg := &nats.ConsumerConfig{
		Durable:           l.uuid,
		AckPolicy:         nats.AckExplicitPolicy,
		InactiveThreshold: l.lockValidity,
		DeliverSubject:    l.uuid,
	}
	cfg.Heartbeat = max(l.retryDelay, 100*time.Millisecond)

	if _, err := l.js.AddConsumer(l.key, cfg); err != nil {
		l.lg.Error(fmt.Sprintf("failed to add consumer : %s", err.Error()))
		return nil, err
	}
	l.sub, err = l.js.ChanSubscribe(l.uuid, l.msgQ, nats.Bind(l.key, l.uuid))
	if err != nil {
		l.lg.Error(fmt.Sprintf("failed to subscribe : %s", err.Error()))
		return nil, err
	}
	go l.keepalive()
	return nil, nil
}

func (l *Lock) signalDone() {
	if l.done == nil {
		return
	}
	select {
	case <-l.done:
	default:
	}
}

func (l *Lock) keepalive() {
	defer l.signalDone()
	for {
		select {
		case msg, ok := <-l.msgQ:
			if !ok {
				l.lg.Warn("releasing keepalive loop")
				return
			} else {
				if err := msg.Ack(); err != nil {
					l.lg.Error(fmt.Sprintf("failed to ack : %s", err.Error()))
				}
			}
		case <-l.signalUnlock:
			return
		}
	}
}

func (l *Lock) Unlock() error {
	ctx, ca := context.WithTimeout(context.Background(), 60*time.Second)
	defer ca()
	return l.unlock(ctx)
}

// best effort unlock until context is done, at which point we
// basically disconnect the connection keepalive semantic
// which delegates unlock the key to the KV server-side,
// giving the guarantee that unlock always actually unlocks when called
func (l *Lock) unlock(ctx context.Context) error {
	// timeout := time.After(l.AcquireTimeout)
	tTicker := time.NewTicker(l.retryDelay)
	defer tTicker.Stop()
	defer func() {
		l.lg.Info("released lock")
		l.signalUnlock <- struct{}{}
		close(l.signalUnlock)
	}()

	for {
		select {
		case <-tTicker.C:
			err := l.tryUnlock()
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *Lock) isReleased(err error) bool {
	return errors.Is(err, nats.ErrConsumerNotFound) || errors.Is(err, nats.ErrConsumerNotActive)
}

// FYI : do not treats nats closed connections as successful unlocks, this could lead to inconsistent states
func (l *Lock) tryUnlock() error {
	consumerErr := l.js.DeleteConsumer(l.key, l.uuid)
	if !l.isReleased(consumerErr) {
		l.lg.Error(fmt.Sprintf("failed to delete consumer : %s", consumerErr.Error()))
		return consumerErr
	}
	if err := l.sub.Unsubscribe(); err != nil {
		l.lg.Error(fmt.Sprintf("failed to unsubscribe : %s", err.Error()))
		return err
	}
	return nil
}

func (l *Lock) Key() string {
	return l.key
}

func (l *Lock) TryLock(_ context.Context) (acquired bool, done chan struct{}, err error) {
	done, err = l.doLock()
	if err == nil {
		return true, done, nil
	}

	// hack : jetstream client does not have an error type for : maxium consumers limit reached
	if strings.Contains(err.Error(), "maximum consumers limit reached") {
		// the request has gone through but someone else has the lock
		return false, nil, nil
	}

	return false, nil, err
}
