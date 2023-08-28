package storage

import (
	"context"
	"time"

	corev1 "github.com/rancher/opni/pkg/apis/core/v1"
	"github.com/rancher/opni/pkg/keyring"
	"github.com/rancher/opni/pkg/storage/lock"
)

type Backend interface {
	TokenStore
	ClusterStore
	RBACStore
	KeyringStoreBroker
	KeyValueStoreBroker
}

type MutatorFunc[T any] func(T)

type TokenMutator = MutatorFunc[*corev1.BootstrapToken]
type ClusterMutator = MutatorFunc[*corev1.Cluster]
type RoleMutator = MutatorFunc[*corev1.Role]
type RoleBindingMutator = MutatorFunc[*corev1.RoleBinding]

type TokenStore interface {
	CreateToken(ctx context.Context, ttl time.Duration, opts ...TokenCreateOption) (*corev1.BootstrapToken, error)
	DeleteToken(ctx context.Context, ref *corev1.Reference) error
	GetToken(ctx context.Context, ref *corev1.Reference) (*corev1.BootstrapToken, error)
	UpdateToken(ctx context.Context, ref *corev1.Reference, mutator TokenMutator) (*corev1.BootstrapToken, error)
	ListTokens(ctx context.Context) ([]*corev1.BootstrapToken, error)
}

type ClusterStore interface {
	CreateCluster(ctx context.Context, cluster *corev1.Cluster) error
	DeleteCluster(ctx context.Context, ref *corev1.Reference) error
	GetCluster(ctx context.Context, ref *corev1.Reference) (*corev1.Cluster, error)
	UpdateCluster(ctx context.Context, ref *corev1.Reference, mutator ClusterMutator) (*corev1.Cluster, error)
	WatchCluster(ctx context.Context, cluster *corev1.Cluster) (<-chan WatchEvent[*corev1.Cluster], error)
	WatchClusters(ctx context.Context, known []*corev1.Cluster) (<-chan WatchEvent[*corev1.Cluster], error)
	ListClusters(ctx context.Context, matchLabels *corev1.LabelSelector, matchOptions corev1.MatchOptions) (*corev1.ClusterList, error)
}

type RBACStore interface {
	CreateRole(context.Context, *corev1.Role) error
	UpdateRole(ctx context.Context, ref *corev1.Reference, mutator RoleMutator) (*corev1.Role, error)
	DeleteRole(context.Context, *corev1.Reference) error
	GetRole(context.Context, *corev1.Reference) (*corev1.Role, error)
	CreateRoleBinding(context.Context, *corev1.RoleBinding) error
	UpdateRoleBinding(ctx context.Context, ref *corev1.Reference, mutator RoleBindingMutator) (*corev1.RoleBinding, error)
	DeleteRoleBinding(context.Context, *corev1.Reference) error
	GetRoleBinding(context.Context, *corev1.Reference) (*corev1.RoleBinding, error)
	ListRoles(context.Context) (*corev1.RoleList, error)
	ListRoleBindings(context.Context) (*corev1.RoleBindingList, error)
}

type KeyringStore interface {
	Put(ctx context.Context, keyring keyring.Keyring) error
	Get(ctx context.Context) (keyring.Keyring, error)
	Delete(ctx context.Context) error
}

type KeyValueStoreT[T any] interface {
	Put(ctx context.Context, key string, value T) error
	Get(ctx context.Context, key string) (T, error)
	Delete(ctx context.Context, key string) error
	ListKeys(ctx context.Context, prefix string) ([]string, error)
}

type KeyValueStore KeyValueStoreT[[]byte]

type KeyringStoreBroker interface {
	KeyringStore(namespace string, ref *corev1.Reference) KeyringStore
}

type KeyValueStoreBroker interface {
	KeyValueStore(namespace string) KeyValueStore
}

// A store that can be used to compute subject access rules
type SubjectAccessCapableStore interface {
	ListClusters(ctx context.Context, matchLabels *corev1.LabelSelector, matchOptions corev1.MatchOptions) (*corev1.ClusterList, error)
	GetRole(ctx context.Context, ref *corev1.Reference) (*corev1.Role, error)
	ListRoleBindings(ctx context.Context) (*corev1.RoleBindingList, error)
}

type WatchEventType string

// Lock is a distributed lock that can be used to coordinate access to a resource.
//
// Locks are single use, and return errors when used more than once. Retry mechanisms are built into\
// the Lock and can be configured with LockOptions.
type Lock interface {
	// Lock acquires a lock on the key. If the lock is already held, it will block until the lock is released.\
	//
	// Lock returns an error when acquiring the lock fails.
	Lock() error
	// Unlock releases the lock on the key. If the lock was never held, it will return an error.
	Unlock() error
}

// LockManager replaces sync.Mutex when a distributed locking mechanism is required.
//
// # Usage
//
// ## Transient lock (transactions, tasks, ...)
// ```
//
//		func distributedTransation(lm stores.LockManager, key string) error {
//			keyMu := lm.Locker(key, lock.WithKeepalive(false), lock.WithExpireDuration(1 * time.Second))
//			if err := keyMu.Lock(); err != nil {
//			return err
//			}
//			defer keyMu.Unlock()
//	     // do some work
//		}
//
// ```
//
// ## Persistent lock (leader election)
//
// ```
//
//	func serve(lm stores.LockManager, key string, done chan struct{}) error {
//		keyMu := lm.Locker(key, lock.WithKeepalive(true))
//		if err := keyMu.Lock(); err != nil {
//			return err
//		}
//		go func() {
//			<-done
//			keyMu.Unlock()
//		}()
//		// serve a service...
//	}
//
// ```
type LockManager interface {
	// Instantiates a new Lock instance for the given key, with the given options.
	//
	// Defaults to lock.DefaultOptions if no options are provided.
	Locker(key string, opts ...lock.LockOption) Lock
}

const (
	WatchEventCreate WatchEventType = "PUT"
	WatchEventUpdate WatchEventType = "UPDATE"
	WatchEventDelete WatchEventType = "DELETE"
)

type WatchEvent[T any] struct {
	EventType WatchEventType
	Current   T
	Previous  T
}

type HttpTtlCache[T any] interface {
	// getter for default cache's configuration
	MaxAge() time.Duration

	Get(key string) (resp T, ok bool)
	// If 0 is passed as ttl, the default cache's configuration will be used
	Set(key string, resp T)
	Delete(key string)
}

type GrpcTtlCache[T any] interface {
	// getter for default cache's configuration
	MaxAge() time.Duration

	Get(key string) (resp T, ok bool)
	// If 0 is passed as ttl, the default cache's configuration will be used
	Set(key string, resp T, ttl time.Duration)
	Delete(key string)
}

var (
	storeBuilderCache = map[string]func(...any) (any, error){}
)

func RegisterStoreBuilder[T ~string](name T, builder func(...any) (any, error)) {
	storeBuilderCache[string(name)] = builder
}

func GetStoreBuilder[T ~string](name T) func(...any) (any, error) {
	return storeBuilderCache[string(name)]
}
