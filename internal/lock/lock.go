/*
Code in this package is meant for testing only. It is not used for production process sychronization.

The command built using this package uses non-standard exit codes.
*/
package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/rancher/opni/pkg/config/v1beta1"
	"github.com/rancher/opni/pkg/storage"
	"github.com/rancher/opni/pkg/storage/etcd"
	"github.com/rancher/opni/pkg/storage/jetstream"
	"github.com/rancher/opni/pkg/storage/lock"
	"github.com/rancher/opni/pkg/validation"
	"github.com/spf13/cobra"
)

const (
	CodeValidationFailed = 1
	CodeSetupFailure     = 2
	CodeAcquired         = 3
	CodeFailedToAcquire  = 4
)

func ToCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrLockValidationFailed) {
		return CodeValidationFailed
	}
	if errors.Is(err, ErrLockSetupFailure) {
		return CodeSetupFailure
	}
	if errors.Is(err, ErrLockAcquired) {
		return CodeAcquired
	}
	if errors.Is(err, ErrFailedToAcquireLock) {
		return CodeFailedToAcquire
	}
	return -1
}

func ToError(
	code int,
) error {
	switch code {
	case 0:
		return nil
	case CodeValidationFailed:
		return ErrLockValidationFailed
	case CodeSetupFailure:
		return ErrLockSetupFailure
	case CodeAcquired:
		return ErrLockAcquired
	case CodeFailedToAcquire:
		return ErrFailedToAcquireLock
	}
	return fmt.Errorf("unhandled error code %d", code)
}

var (
	ErrLockValidationFailed = errors.New("validation failed")
	ErrLockSetupFailure     = errors.New("setup failure")
	ErrLockAcquired         = errors.New("lock already acquired")
	ErrFailedToAcquireLock  = errors.New("failed to acquire lock")
)

func BuildLockCommand() *cobra.Command {
	var lockFile string
	var lockKey string
	var validFor time.Duration
	var tryLock bool
	cmd := &cobra.Command{
		Use:          "lock",
		Short:        "lock",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), LockConfig{
				Key:        lockKey,
				ValidFor:   validFor,
				ConfigPath: lockFile,
				Try:        tryLock,
			})
		},
	}
	cmd.Flags().StringVarP(&lockFile, "config", "f", "/tmp/lock.json", "lock file config")
	cmd.Flags().StringVarP(&lockKey, "key", "k", "", "lock key")
	cmd.Flags().DurationVarP(&validFor, "valid-for", "t", 0, "how long the lock is valid for (0 is infinite)")
	cmd.Flags().BoolVarP(&tryLock, "try-lock", "q", false, "if lock is already held, do not wait for it to be released")
	return cmd
}

type LockBackendConfig struct {
	Etcd      *v1beta1.EtcdStorageSpec
	Jetstream *v1beta1.JetStreamStorageSpec
}

type LockConfig struct {
	Key        string
	ValidFor   time.Duration
	ConfigPath string
	Try        bool
}

func (l *LockConfig) Validate() error {
	if l.Key == "" {
		return validation.Error("key required")
	}
	if l.ConfigPath == "" {
		return validation.Error("lock config path is required")
	}
	return nil
}

func (l *LockBackendConfig) Validate() error {
	if l.Etcd == nil && l.Jetstream == nil {
		return validation.Error("must specify one lock backend")
	}

	if l.Etcd != nil && l.Jetstream != nil {
		return validation.Error("only one lock backend can be used at a time")
	}
	return nil
}

func run(ctx context.Context, lockcfg LockConfig) error {
	lg := slog.Default()
	lg.With("config", lockcfg.ConfigPath, "key", lockcfg.Key, "validity", lockcfg.ValidFor).Info("Starting lock")
	if err := lockcfg.Validate(); err != nil {
		lg.Error("failed to validate lock config")
		return errors.Join(err, ErrLockValidationFailed)
	}
	config, err := getConfig(lockcfg.ConfigPath)
	if err != nil {
		lg.Error("failed to get config")
		return errors.Join(err, ErrLockSetupFailure)
	}

	lm, err := getLockManager(ctx, config)
	if err != nil {
		configB, _ := json.Marshal(config)
		lg.With("config", string(configB)).Error(err.Error())
		return errors.Join(err, ErrLockSetupFailure)
	}
	return acquire(ctx, lockcfg, lm, lg)
}

func getLockManager(ctx context.Context, config *LockBackendConfig) (storage.LockManager, error) {
	if config.Etcd != nil {
		client, err := etcd.NewEtcdClient(ctx, config.Etcd)
		if err != nil {
			return nil, err
		}
		lm, err := etcd.NewEtcdLockManager(client, "test/lock")
		if err != nil {
			return nil, err
		}
		return lm, nil
	}
	if config.Jetstream != nil {
		lm, err := jetstream.NewJetstreamLockManager(ctx, config.Jetstream)
		if err != nil {
			return nil, err
		}
		return lm, nil
	}
	return nil, fmt.Errorf("unsupported lock backend")
}

func getConfig(configPath string) (*LockBackendConfig, error) {
	if configPath == "" {
		return nil, validation.Error("config path required")
	}
	f, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rawConfig, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	config := &LockBackendConfig{}
	if err := json.Unmarshal(rawConfig, config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func acquire(ctx context.Context, config LockConfig, lm storage.LockManager, lg *slog.Logger) error {
	lg = lg.With("key", config.Key)
	done := make(chan os.Signal, 1)
	ctxca, ca := context.WithCancel(ctx)
	if config.ValidFor > 0 {
		ctxca, ca = context.WithTimeout(ctx, config.ValidFor)
	}
	defer ca()
	signal.Notify(done, os.Interrupt)
	lg.Info("acquiring lock...")
	go func() {
		defer close(done)
		<-done
		ca()
	}()

	lock := lm.Locker(config.Key, lock.WithAcquireContext(ctx))
	if config.Try {
		ack, err := lock.TryLock()
		if err != nil {
			lg.With("err", err, "failed to acquire lock")
			return errors.Join(err, ErrFailedToAcquireLock)
		}
		if !ack {
			return ErrLockAcquired
		}
	} else {
		if err := lock.Lock(); err != nil {
			lg.With("err", err, "failed to acquire lock")
			return errors.Join(err, ErrFailedToAcquireLock)
		}
	}
	lg.Info("acquired lock")
	defer func() {
		if err := lock.Unlock(); err != nil {
			lg.With("err", err, "failed to release lock")
		}
	}()

	<-ctxca.Done()
	lg.Info("releasing lock...")
	return nil
}
