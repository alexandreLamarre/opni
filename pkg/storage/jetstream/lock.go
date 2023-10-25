package jetstream

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
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

	acquired uint32

	retryDelay   time.Duration
	lockValidity time.Duration

	js   nats.JetStreamContext
	sub  *nats.Subscription
	msgQ chan *nats.Msg
	*lock.LockOptions

	startLock   lock.LockPrimitive
	startUnlock lock.LockPrimitive
}

var _ storage.Lock = (*Lock)(nil)

func NewLock(js nats.JetStreamContext, key string, options *lock.LockOptions) *Lock {
	return &Lock{
		key:          key,
		js:           js,
		uuid:         uuid.New().String(),
		msgQ:         make(chan *nats.Msg, 16),
		LockOptions:  options,
		retryDelay:   time.Millisecond * 50,
		lockValidity: time.Second * 60,
	}
}

func (l *Lock) Lock(_ context.Context) error {

	return l.startLock.Do(func() error {
		return l.lock()
	})
}

func (l *Lock) lock() error {
	// timeout := time.After(l.AcquireTimeout)
	tTicker := time.NewTicker(l.retryDelay)
	defer tTicker.Stop()

	var lockErr error
	for {
		select {
		case <-tTicker.C:
			err := l.tryLock()
			if err == nil {
				return nil
			}
			lockErr = err
		case <-l.AcquireContext.Done():
			return errors.Join(l.AcquireContext.Err(), lockErr)
			// case <-timeout:
			// return errors.Join(lockErr, lock.ErrAcquireLockTimeout)
		}
	}
}

func (l *Lock) tryLock() error {
	var err error
	if _, err := l.js.AddStream(newLease(l.key)); err != nil {
		return err
	}
	cfg := &nats.ConsumerConfig{
		Durable:           l.uuid,
		AckPolicy:         nats.AckExplicitPolicy,
		InactiveThreshold: l.lockValidity,
		DeliverSubject:    l.uuid,
	}
	// if l.Keepalive {
	// Push-based details, defaults to 5 * time.Second
	cfg.Heartbeat = max(l.retryDelay, 100*time.Millisecond)
	// }

	if _, err := l.js.AddConsumer(l.key, cfg); err != nil {
		return err
	}
	l.sub, err = l.js.ChanSubscribe(l.uuid, l.msgQ, nats.Bind(l.key, l.uuid))
	if err != nil {
		return err
	}
	go l.keepalive()
	// go l.expire()
	atomic.StoreUint32(&l.acquired, 1)
	return nil
}

// func (l *Lock) expire() {
// 	if l.Keepalive {
// 		return
// 	}
// 	timeout := time.After(l.lockValidity)
// 	<-timeout
// 	l.unlock()
// 	fmt.Println("expiring lock")
// }

func (l *Lock) keepalive() {
	// fmt.Println("keepalive", l.Keepalive)
	defer func() {
		fmt.Println("releasing keepalive loop")
	}()
	// if !l.Keepalive {
	// 	return
	// }
	for {
		select {
		case msg, ok := <-l.msgQ:
			if !ok {
				return
			} else {
				if err := msg.Ack(); err != nil {
					fmt.Println("failed to ack msg", err.Error())
				}
			}
		}
	}
}

func (l *Lock) Unlock() error {
	return l.startUnlock.Do(func() error {
		if atomic.LoadUint32(&l.acquired) == 0 {
			return fmt.Errorf("lock never acquired")
		}
		return l.unlock()
	})
}

func (l *Lock) unlock() error {
	// timeout := time.After(l.AcquireTimeout)
	tTicker := time.NewTicker(l.retryDelay)
	defer tTicker.Stop()

	var unlockErr error
	for {
		select {
		case <-tTicker.C:
			err := l.tryUnlock()
			if err == nil {
				return nil
			}
			unlockErr = err
		case <-l.AcquireContext.Done():
			return errors.Join(l.AcquireContext.Err(), unlockErr)
			// return errors.Join(lock.ErrAcquireUnlockCancelled, unlockErr)
			// case <-timeout:
			// 	return errors.Join(lock.ErrAcquireUnlockTimeout, unlockErr)
		}
	}
}

func (l *Lock) tryUnlock() error {
	if err := l.sub.Unsubscribe(); err != nil {
		if errors.Is(err, nats.ErrBadSubscription) || errors.Is(err, nats.ErrConsumerNotActive) { // lock already released
			return nil
		}
		return err
	}
	if err := l.js.DeleteConsumer(l.key, l.uuid); err != nil {
		if errors.Is(err, nats.ErrConsumerNotFound) || errors.Is(err, nats.ErrConsumerNotActive) { // lock already released
			return nil
		}
		return err
	}
	return nil
}

func (l *Lock) Key() string {
	return l.key
}

func (l *Lock) TryLock(_ context.Context) (acquired bool, err error) {
	// TODO
	err = l.tryLock()
	return err != nil, err
}

func (l *Lock) TryUnlock() (bool, error) {
	// TODO
	err := l.tryUnlock()
	return err != nil, err
}
