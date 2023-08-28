package conformance_storage

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/opni/pkg/storage"
	"github.com/rancher/opni/pkg/storage/lock"
	"github.com/rancher/opni/pkg/util/future"
	"github.com/samber/lo"
)

func LockManagerTestSuite[T storage.LockManager](
	lmF future.Future[T],
) func() {
	return func() {
		var lm T
		BeforeAll(func() {
			lm = lmF.Get()
		})

		When("using distributed locks ", func() {
			It("should only request lock actions once", func() {
				lock1 := lm.Locker("foo")
				err := lock1.Lock()
				Expect(err).NotTo(HaveOccurred())
				err = lock1.Lock()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(lock.ErrLockActionRequested))

				err = lock1.Unlock()
				Expect(err).NotTo(HaveOccurred())
				err = lock1.Unlock()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(lock.ErrLockActionRequested))
			})

			It("should lock and unlock locks of the same type", func() {
				lock1 := lm.Locker("todo")
				Expect(lock1.Lock()).To(Succeed())
				Expect(lock1.Unlock()).To(Succeed())

				lock2 := lm.Locker("todo")
				Expect(lock2.Lock()).To(Succeed())
				Expect(lock2.Unlock()).To(Succeed())
			})
			It("should resolve concurrent lock conflicts", func() {
				locks := []LockWithTransaction{}
				for i := 0; i < 2; i++ {
					lock := lm.Locker("todo")
					locks = append(locks, LockWithTransaction{
						A: lock,
						C: make(chan struct{}),
					})
				}
				errs := make(chan error, 2*len(locks))

				lockOrder := lo.Shuffle(locks)
				var wg sync.WaitGroup
				for _, lock := range lockOrder {
					lock := lock
					wg.Add(1)
					go func() {
						defer wg.Done()
						err := lock.A.Lock()
						sendWithJitter(lock.C)
						errs <- err
					}()
				}
				for _, lock := range lockOrder {
					lock := lock
					wg.Add(1)
					go func() {
						defer wg.Done()
						<-lock.C
						errs <- lock.A.Unlock()
					}()
				}
				wg.Wait()
				Eventually(errs).Should(Receive(BeNil()))
			})

			Specify("locks should be able to acquire the lock if the existing lock has expired", func() {
				exLock := lm.Locker("bar", lock.WithRetryDelay(1*time.Millisecond), lock.WithExpireDuration(time.Millisecond*50))
				otherLock := lm.Locker("bar", lock.WithAcquireTimeout(time.Second))
				err := exLock.Lock()
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(2 * time.Second)

				err = otherLock.Lock()
				Expect(err).NotTo(HaveOccurred())

				By("releasing the expired exclusive lock should not affect releasing newer locks")
				err = exLock.Unlock()
				Expect(err).NotTo(HaveOccurred())

				err = otherLock.Unlock()
				Expect(err).NotTo(HaveOccurred())
			})

			Specify("any lock should timeout after the specified timeout", func() {
				Expect(ExpectTimeout(lm)).To(Succeed())

			})

			Specify("lock should respect their context errors", func() {
				Expect(ExpectCancellable(lm)).To(Succeed())
			})
		})

		When("using exclusive locks", func() {
			It("should allow multiple writers to safely write using locks", func() {
				lock1 := lm.Locker("foo2", lock.WithKeepalive(false), lock.WithExpireDuration(10*time.Millisecond))
				err := ExpectAtomic(lock1, func(opts ...lock.LockOption) storage.Lock {
					return lm.Locker("foo2", opts...)
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should block other locks from acquiring the lock if the process holding the lock is alive", func() {
				lock1 := lm.Locker("foo3", lock.WithKeepalive(true))
				err := ExpectAtomicKeepAlive(lock1, func(opts ...lock.LockOption) storage.Lock {
					return lm.Locker("foo3", opts...)
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

	}
}
