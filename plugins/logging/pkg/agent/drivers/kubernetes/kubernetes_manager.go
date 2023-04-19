package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	loggingv1beta1 "github.com/banzaicloud/logging-operator/pkg/sdk/logging/api/v1beta1"
	"github.com/banzaicloud/logging-operator/pkg/sdk/logging/model/filter"
	"github.com/banzaicloud/logging-operator/pkg/sdk/logging/model/output"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/rancher/opni/apis"
	opnicorev1beta1 "github.com/rancher/opni/apis/core/v1beta1"
	opniloggingv1beta1 "github.com/rancher/opni/apis/logging/v1beta1"
	"github.com/rancher/opni/pkg/otel"
	"github.com/rancher/opni/pkg/util/k8sutil/provider"
	opnimeta "github.com/rancher/opni/pkg/util/meta"
	"github.com/rancher/opni/plugins/logging/pkg/agent/drivers"
	"github.com/rancher/opni/plugins/logging/pkg/apis/node"
	"github.com/samber/lo"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	secretName         = "opni-opensearch-auth"
	secretKey          = "password"
	dataPrepperName    = "opni-shipper"
	dataPrepperVersion = "2.0.1"
	clusterOutputName  = "opni-output"
	clusterFlowName    = "opni-flow"
	loggingConfigName  = "opni-logging"
)

type KubernetesManagerDriver struct {
	KubernetesManagerDriverOptions
	logger    *zap.SugaredLogger
	namespace string
	k8sClient client.Client
	provider  string

	state reconcilerState
}

type reconcilerState struct {
	sync.Mutex
	running       bool
	backoffCtx    context.Context
	backoffCancel context.CancelFunc
}

type KubernetesManagerDriverOptions struct {
	restConfig *rest.Config
}

type KubernetesManagerDriverOption func(*KubernetesManagerDriverOptions)

func (o *KubernetesManagerDriverOptions) apply(opts ...KubernetesManagerDriverOption) {
	for _, op := range opts {
		op(o)
	}
}

func WithRestConfig(restConfig *rest.Config) KubernetesManagerDriverOption {
	return func(o *KubernetesManagerDriverOptions) {
		o.restConfig = restConfig
	}
}

func NewKubernetesManagerDriver(
	logger *zap.SugaredLogger,
	opts ...KubernetesManagerDriverOption,
) (*KubernetesManagerDriver, error) {
	options := KubernetesManagerDriverOptions{}
	options.apply(opts...)

	namespace, ok := os.LookupEnv("POD_NAMESPACE")
	if !ok {
		return nil, fmt.Errorf("POD_NAMESPACE environment variable not set")
	}

	if options.restConfig == nil {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create rest config: %w", err)
		}
		options.restConfig = restConfig
	}

	k8sClient, err := client.New(options.restConfig, client.Options{
		Scheme: apis.NewScheme(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8sClient: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(options.restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	provider, err := provider.DetectProvider(context.TODO(), clientset)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	return &KubernetesManagerDriver{
		KubernetesManagerDriverOptions: options,
		logger:                         logger,
		namespace:                      namespace,
		provider:                       provider,
		k8sClient:                      k8sClient,
	}, nil

}

func (m *KubernetesManagerDriver) Name() string {
	return "kubernetes-manager"
}

var _ drivers.LoggingNodeDriver = (*KubernetesManagerDriver)(nil)

func (m *KubernetesManagerDriver) ConfigureNode(config *node.LoggingCapabilityConfig) {
	m.state.Lock()
	if m.state.running {
		m.state.backoffCancel()
	}
	m.state.running = true
	ctx, ca := context.WithCancel(context.TODO())
	m.state.backoffCtx = ctx
	m.state.backoffCancel = ca
	m.state.Unlock()

	p := backoff.Exponential()
	b := p.Start(ctx)
	var success bool
BACKOFF:
	for backoff.Continue(b) {
		// First remove old objects
		for _, obj := range []client.Object{
			m.buildAuthSecret(),
			m.buildDataPrepper(),
			m.buildOpniClusterOutput(),
			m.buildOpniClusterFlow(),
			m.buildLogAdapter(),
		} {
			if err := m.reconcileObject(obj, false); err != nil {
				m.logger.With(
					"object", client.ObjectKeyFromObject(obj).String(),
					zap.Error(err),
				).Error("error reconciling object")
				continue BACKOFF
			}
		}
		// Now create new objects
		collectorConf := m.buildLoggingCollectorConfig()
		if err := m.reconcileObject(collectorConf, config.Enabled); err != nil {
			m.logger.With(
				"object", client.ObjectKeyFromObject(collectorConf).String(),
				zap.Error(err),
			).Error("error reconciling object")
			continue BACKOFF
		}
		if err := m.reconcileCollector(config.Enabled); err != nil {
			m.logger.With(
				"object", "opni collector",
				zap.Error(err),
			).Error("error reconciling object")
		}

		success = true
		break
	}

	if !success {
		m.logger.Error("timed out reconciling objects")
	} else {
		m.logger.Info("objects reconciled successfully")
	}
}

func (m *KubernetesManagerDriver) buildAuthSecret() *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: m.namespace,
		},
	}
	return secret
}

func (m *KubernetesManagerDriver) buildDataPrepper() *opniloggingv1beta1.DataPrepper {
	dataPrepper := &opniloggingv1beta1.DataPrepper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dataPrepperName,
			Namespace: m.namespace,
		},
	}
	return dataPrepper
}

func (m *KubernetesManagerDriver) buildOpniClusterOutput() *loggingv1beta1.ClusterOutput {
	return &loggingv1beta1.ClusterOutput{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterOutputName,
			Namespace: m.namespace,
		},
		Spec: loggingv1beta1.ClusterOutputSpec{
			OutputSpec: loggingv1beta1.OutputSpec{
				HTTPOutput: &output.HTTPOutputConfig{
					Endpoint:    fmt.Sprintf("http://%s.%s:2021/log/ingest", dataPrepperName, m.namespace),
					ContentType: "application/json",
					JsonArray:   true,
					Buffer: &output.Buffer{
						Tags:           lo.ToPtr("[]"),
						FlushInterval:  "2s",
						ChunkLimitSize: "1mb",
					},
				},
			},
		},
	}
}

func (m *KubernetesManagerDriver) buildOpniClusterFlow() *loggingv1beta1.ClusterFlow {
	return &loggingv1beta1.ClusterFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterFlowName,
			Namespace: m.namespace,
		},
		Spec: loggingv1beta1.ClusterFlowSpec{
			Filters: []loggingv1beta1.Filter{
				{
					Dedot: &filter.DedotFilterConfig{
						Separator: "-",
						Nested:    true,
					},
				},
				{
					Grep: &filter.GrepConfig{
						Exclude: []filter.ExcludeSection{
							{
								Key:     "log",
								Pattern: `^\n$`,
							},
						},
					},
				},
				{
					DetectExceptions: &filter.DetectExceptions{
						Languages: []string{
							"java",
							"python",
							"go",
							"ruby",
							"js",
							"csharp",
							"php",
						},
						MultilineFlushInterval: "0.1",
					},
				},
			},
			Match: []loggingv1beta1.ClusterMatch{
				{
					ClusterExclude: &loggingv1beta1.ClusterExclude{
						Namespaces: []string{
							m.namespace,
						},
					},
				},
				{
					ClusterSelect: &loggingv1beta1.ClusterSelect{},
				},
			},
			GlobalOutputRefs: []string{
				clusterOutputName,
			},
		},
	}
}

func (m *KubernetesManagerDriver) buildLogAdapter() *opniloggingv1beta1.LogAdapter {
	logAdapter := &opniloggingv1beta1.LogAdapter{
		ObjectMeta: metav1.ObjectMeta{
			Name: "opni-logging",
		},
		Spec: opniloggingv1beta1.LogAdapterSpec{
			Provider:         opniloggingv1beta1.LogProvider(m.provider),
			ControlNamespace: &m.namespace,
		},
	}
	logAdapter.Default()
	return logAdapter
}

func (m *KubernetesManagerDriver) buildLoggingCollectorConfig() *opniloggingv1beta1.CollectorConfig {
	collectorConfig := &opniloggingv1beta1.CollectorConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "opni-logging",
		},
		Spec: opniloggingv1beta1.CollectorConfigSpec{
			Provider: opniloggingv1beta1.LogProvider(m.provider),
		},
	}
	return collectorConfig
}

// TODO: make this generic
func (m *KubernetesManagerDriver) reconcileObject(desired client.Object, shouldExist bool) error {
	// get the object
	key := client.ObjectKeyFromObject(desired)
	lg := m.logger.With("object", key)
	lg.Info("reconciling object")

	// get the agent statefulset
	agentStatefulSet, err := m.getAgentSet()
	if err != nil {
		return err
	}

	current := desired.DeepCopyObject().(client.Object)
	err = m.k8sClient.Get(context.TODO(), key, current)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	// this can error if the object is cluster-scoped, but that's ok
	controllerutil.SetOwnerReference(agentStatefulSet, desired, m.k8sClient.Scheme())

	if k8serrors.IsNotFound(err) {
		if !shouldExist {
			lg.Info("object does not exist and should not exist, skipping")
			return nil
		}
		lg.Info("object does not exist, creating")
		// create the object
		return m.k8sClient.Create(context.TODO(), desired)
	} else if !shouldExist {
		// delete the object
		lg.Info("object exists and should not exist, deleting")
		return m.k8sClient.Delete(context.TODO(), current)
	}

	// update the object
	return m.patchObject(current, desired)
}

func (m *KubernetesManagerDriver) reconcileCollector(shouldExist bool) error {
	coll := &opnicorev1beta1.Collector{
		ObjectMeta: metav1.ObjectMeta{
			Name: otel.CollectorName,
		},
	}
	err := m.k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(coll), coll)
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	if k8serrors.IsNotFound(err) && shouldExist {
		coll = m.buildEmptyCollector()
		coll.Spec.LoggingConfig = &corev1.LocalObjectReference{
			Name: loggingConfigName,
		}
		return m.k8sClient.Create(context.TODO(), coll)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		err := m.k8sClient.Get(context.TODO(), client.ObjectKeyFromObject(coll), coll)
		if err != nil {
			return err
		}
		if shouldExist {
			coll.Spec.LoggingConfig = &corev1.LocalObjectReference{
				Name: loggingConfigName,
			}
		} else {
			coll.Spec.LoggingConfig = nil
		}

		return m.k8sClient.Update(context.TODO(), coll)
	})
	return err
}

func (m *KubernetesManagerDriver) getAgentSet() (*appsv1.StatefulSet, error) {
	list := &appsv1.StatefulSetList{}
	if err := m.k8sClient.List(context.TODO(), list,
		client.InNamespace(m.namespace),
		client.MatchingLabels{
			"opni.io/app": "agent",
		},
	); err != nil {
		return nil, err
	}

	if len(list.Items) != 1 {
		return nil, errors.New("statefulsets found not exactly 1")
	}
	return &list.Items[0], nil
}

func (m *KubernetesManagerDriver) getAgentService() (*corev1.Service, error) {
	list := &corev1.ServiceList{}
	if err := m.k8sClient.List(context.TODO(), list,
		client.InNamespace(m.namespace),
		client.MatchingLabels{
			"opni.io/app": "agent",
		},
	); err != nil {
		return nil, err
	}

	if len(list.Items) != 1 {
		return nil, errors.New("statefulsets found not exactly 1")
	}
	return &list.Items[0], nil
}

func (m *KubernetesManagerDriver) patchObject(current client.Object, desired client.Object) error {
	// update the object
	patchResult, err := patch.DefaultPatchMaker.Calculate(current, desired, patch.IgnoreStatusFields())
	if err != nil {
		m.logger.With(
			zap.Error(err),
		).Warn("could not match objects")
		return err
	}
	if patchResult.IsEmpty() {
		m.logger.Info("resource is in sync")
		return nil
	}
	m.logger.Info("resource diff")

	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
		m.logger.With(
			zap.Error(err),
		).Error("failed to set last applied annotation")
	}

	metaAccessor := meta.NewAccessor()

	currentResourceVersion, err := metaAccessor.ResourceVersion(current)
	if err != nil {
		return err
	}
	if err := metaAccessor.SetResourceVersion(desired, currentResourceVersion); err != nil {
		return err
	}

	m.logger.Info("updating resource")

	return m.k8sClient.Update(context.TODO(), desired)
}

func (m *KubernetesManagerDriver) buildEmptyCollector() *opnicorev1beta1.Collector {
	var serviceName string
	service, err := m.getAgentService()
	if err == nil {
		serviceName = service.Name
	}
	return &opnicorev1beta1.Collector{
		ObjectMeta: metav1.ObjectMeta{
			Name: otel.CollectorName,
		},
		Spec: opnicorev1beta1.CollectorSpec{
			ImageSpec: opnimeta.ImageSpec{
				ImagePullPolicy: lo.ToPtr(corev1.PullAlways),
			},
			SystemNamespace: m.namespace,
			AgentEndpoint:   otel.AgentEndpoint(serviceName),
		},
	}
}
