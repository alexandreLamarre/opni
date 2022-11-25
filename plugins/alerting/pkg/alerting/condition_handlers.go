/*
- Functions that handle each endpoint implementation update case
- Functions that handle each alert condition case
*/
package alerting

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/common/model"
	"github.com/rancher/opni/pkg/health"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rancher/opni/pkg/alerting/metrics"
	"github.com/rancher/opni/plugins/metrics/pkg/apis/cortexadmin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/rancher/opni/pkg/alerting/shared"
	alertingv1 "github.com/rancher/opni/pkg/apis/alerting/v1"
	corev1 "github.com/rancher/opni/pkg/apis/core/v1"
	natsutil "github.com/rancher/opni/pkg/util/nats"
)

func setupCondition(
	p *Plugin,
	lg *zap.SugaredLogger,
	ctx context.Context,
	req *alertingv1.AlertCondition,
	newConditionId string) (*corev1.Reference, error) {
	if s := req.GetAlertType().GetSystem(); s != nil {
		err := p.handleSystemAlertCreation(ctx, s, newConditionId, req.GetName())
		if err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if dc := req.GetAlertType().GetDownstreamCapability(); dc != nil {
		err := p.handleDownstreamCapabilityAlertCreation(ctx, dc, newConditionId, req.GetName())
		if err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if k := req.GetAlertType().GetKubeState(); k != nil {
		err := p.handleKubeAlertCreation(ctx, k, newConditionId, req.Name)
		if err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if c := req.GetAlertType().GetCpu(); c != nil {
		err := p.handleCpuSaturationAlertCreation(ctx, c, newConditionId, req.Name)
		if err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if m := req.AlertType.GetMemory(); m != nil {
		err := p.handleMemorySaturationAlertCreation(ctx, m, newConditionId, req.Name)
		if err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if fs := req.AlertType.GetFs(); fs != nil {
		if err := p.handleFsSaturationAlertCreation(ctx, fs, newConditionId, req.Name); err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	if q := req.AlertType.GetPrometheusQuery(); q != nil {
		if err := p.handlePrometheusQueryAlertCreation(ctx, q, newConditionId, req.Name); err != nil {
			return nil, err
		}
		return &corev1.Reference{Id: newConditionId}, nil
	}
	return nil, shared.AlertingErrNotImplemented
}

func deleteCondition(p *Plugin, lg *zap.SugaredLogger, ctx context.Context, req *alertingv1.AlertCondition, id string) error {
	if r := req.GetAlertType().GetSystem(); r != nil {
		p.msgNode.RemoveConfigListener(id)
		p.storageNode.DeleteIncidentTracker(ctx, id)
		p.storageNode.DeleteConditionStatusTracker(ctx, id)
		return nil
	}
	if r := req.AlertType.GetDownstreamCapability(); r != nil {
		p.msgNode.RemoveConfigListener(id)
		p.storageNode.DeleteIncidentTracker(ctx, id)
		p.storageNode.DeleteConditionStatusTracker(ctx, id)
		return nil
	}
	if r, _ := handleSwitchCortexRules(req.AlertType); r != nil {
		_, err := p.adminClient.Get().DeleteRule(ctx, &cortexadmin.RuleRequest{
			ClusterId: r.Id,
			GroupName: CortexRuleIdFromUuid(id),
		})
		return err
	}
	return shared.AlertingErrNotImplemented
}

func handleSwitchCortexRules(t *alertingv1.AlertTypeDetails) (*corev1.Reference, alertingv1.IndexableMetric) {
	if k := t.GetKubeState(); k != nil {
		return &corev1.Reference{Id: k.ClusterId}, k
	}
	if c := t.GetCpu(); c != nil {
		return c.ClusterId, c
	}
	if m := t.GetMemory(); m != nil {
		return m.ClusterId, m
	}
	if f := t.GetFs(); f != nil {
		return f.ClusterId, f
	}
	if q := t.GetPrometheusQuery(); q != nil {
		return q.ClusterId, q
	}

	return nil, nil
}

func (p *Plugin) handleSystemAlertCreation(
	ctx context.Context,
	k *alertingv1.AlertConditionSystem,
	newConditionId string,
	conditionName string,
) error {
	err := p.onSystemConditionCreate(newConditionId, conditionName, k)
	if err != nil {
		p.Logger.Errorf("failed to create agent condition %s", err)
	}
	return nil
}

func (p *Plugin) handleDownstreamCapabilityAlertCreation(
	ctx context.Context,
	k *alertingv1.AlertConditionDownstreamCapability,
	newConditionId string,
	conditionName string,
) error {
	err := p.onDownstreamCapabilityConditionCreate(newConditionId, conditionName, k)
	if err != nil {
		p.Logger.Errorf("failed to create agent condition %s", err)
	}
	return nil
}

func (p *Plugin) handleKubeAlertCreation(ctx context.Context, k *alertingv1.AlertConditionKubeState, newId, alertName string) error {
	baseKubeRule, err := metrics.NewKubeStateRule(
		k.GetObjectType(),
		k.GetObjectName(),
		k.GetNamespace(),
		k.GetState(),
		timeDurationToPromStr(k.GetFor().AsDuration()),
		metrics.KubeStateAnnotations,
	)
	if err != nil {
		return err
	}
	kubeRuleContent, err := NewCortexAlertingRule(newId, alertName, k, nil, baseKubeRule)
	p.Logger.With("handler", "kubeStateAlertCreate").Debugf("kube state alert created %v", kubeRuleContent)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(kubeRuleContent)
	if err != nil {
		return err
	}
	p.Logger.With("Expr", "kube-state").Debugf("%s", string(out))
	_, err = p.adminClient.Get().LoadRules(ctx, &cortexadmin.PostRuleRequest{
		ClusterId:   k.GetClusterId(),
		YamlContent: string(out),
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Plugin) handleCpuSaturationAlertCreation(
	ctx context.Context,
	c *alertingv1.AlertConditionCPUSaturation,
	conditionId,
	alertName string,
) error {
	baseCpuRule, err := metrics.NewCpuRule(
		c.GetNodeCoreFilters(),
		c.GetCpuStates(),
		c.GetOperation(),
		float64(c.GetExpectedRatio()),
		c.GetFor(),
		metrics.CpuRuleAnnotations,
	)
	if err != nil {
		return err
	}
	cpuRuleContent, err := NewCortexAlertingRule(conditionId, alertName, c, nil, baseCpuRule)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(cpuRuleContent)
	if err != nil {
		return err
	}
	p.Logger.With("Expr", "cpu").Debugf("%s", string(out))

	_, err = p.adminClient.Get().LoadRules(ctx, &cortexadmin.PostRuleRequest{
		ClusterId:   c.ClusterId.GetId(),
		YamlContent: string(out),
	})
	return err
}

func (p *Plugin) handleMemorySaturationAlertCreation(ctx context.Context, m *alertingv1.AlertConditionMemorySaturation, conditionId, alertName string) error {
	baseMemRule, err := metrics.NewMemRule(
		m.GetNodeMemoryFilters(),
		m.UsageTypes,
		m.GetOperation(),
		float64(m.GetExpectedRatio()),
		m.GetFor(),
		metrics.MemRuleAnnotations,
	)
	if err != nil {
		return err
	}
	memRuleContent, err := NewCortexAlertingRule(conditionId, alertName, m, nil, baseMemRule)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(memRuleContent)
	if err != nil {
		return err
	}
	p.Logger.With("Expr", "mem").Debugf("%s", string(out))
	_, err = p.adminClient.Get().LoadRules(ctx, &cortexadmin.PostRuleRequest{
		ClusterId:   m.ClusterId.GetId(),
		YamlContent: string(out),
	})
	return err
}

func (p *Plugin) handleFsSaturationAlertCreation(ctx context.Context, fs *alertingv1.AlertConditionFilesystemSaturation, conditionId, alertName string) error {
	baseFsRule, err := metrics.NewFsRule(
		fs.GetNodeFilters(),
		fs.GetOperation(),
		float64(fs.GetExpectedRatio()),
		fs.GetFor(),
		metrics.MemRuleAnnotations,
	)
	if err != nil {
		return err
	}
	fsRuleContent, err := NewCortexAlertingRule(conditionId, alertName, fs, nil, baseFsRule)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(fsRuleContent)
	if err != nil {
		return err
	}
	p.Logger.With("Expr", "fs").Debugf("%s", string(out))
	_, err = p.adminClient.Get().LoadRules(ctx, &cortexadmin.PostRuleRequest{
		ClusterId:   fs.ClusterId.GetId(),
		YamlContent: string(out),
	})
	return err
}

func (p *Plugin) handlePrometheusQueryAlertCreation(ctx context.Context, q *alertingv1.AlertConditionPrometheusQuery, conditionId, alertName string) error {
	dur := model.Duration(q.GetFor().AsDuration())
	baseRule := &metrics.AlertingRule{
		Alert:       "",
		Expr:        metrics.PostProcessRuleString(q.GetQuery()),
		For:         dur,
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}

	baseRuleContent, err := NewCortexAlertingRule(conditionId, alertName, q, nil, baseRule)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(baseRuleContent)
	if err != nil {
		return err
	}
	p.Logger.With("Expr", "user-query").Debugf("%s", string(out))
	_, err = p.adminClient.Get().LoadRules(ctx, &cortexadmin.PostRuleRequest{
		ClusterId:   q.ClusterId.GetId(),
		YamlContent: string(out),
	})

	return err
}

func (p *Plugin) onSystemConditionCreate(conditionId, conditionName string, condition *alertingv1.AlertConditionSystem) error {
	lg := p.Logger.With("onSystemConditionCreate", conditionId)
	lg.Debugf("received condition update: %v", condition)
	jsCtx, cancel := context.WithCancel(p.Ctx)
	nc := p.natsConn.Get()
	js, err := nc.JetStream()
	if err != nil {
		cancel()
		return err
	}
	lg.Debugf("Creating agent disconnect with timeout %s", condition.GetTimeout().AsDuration())
	evaluator := &InternalConditionEvaluator[health.StatusUpdate]{
		lg:                 lg,
		conditionId:        conditionId,
		conditionName:      conditionName,
		storageNode:        p.storageNode,
		evaluationCtx:      jsCtx,
		parentCtx:          p.Ctx,
		cancelEvaluation:   cancel,
		evaluateDuration:   condition.GetTimeout().AsDuration(),
		alertmanagerlabels: map[string]string{},
		healthOnMessage: func(h health.StatusUpdate) (health bool, ts *timestamppb.Timestamp) {
			return h.Status.Connected, h.Status.Timestamp
		},
		triggerHook: func(ctx context.Context, conditionId string, labels map[string]string) {
			p.TriggerAlerts(ctx, &alertingv1.TriggerAlertsRequest{
				ConditionId: &corev1.Reference{Id: conditionId},
				Annotations: labels,
			})
		},
	}
	// handles re-entrant conditions
	evaluator.CalculateInitialState()
	go func() {
		defer cancel() // cancel parent context, if we return (non-recoverable)
		for {
			err = natsutil.NewPersistentStream(js, shared.NewAlertingDisconnectStream())
			if err != nil {
				lg.Errorf("alerting disconnect stream does not exist and cannot be created %s", err)
				continue
			}
			agentId := condition.GetClusterId().Id
			msgCh := make(chan *nats.Msg, 32)
			sub, err := js.ChanSubscribe(shared.NewAgentDisconnectSubject(agentId), msgCh)
			defer sub.Unsubscribe()
			if err != nil {
				lg.Errorf("failed  to subscribe to %s : %s", shared.NewAgentDisconnectSubject(agentId), err)
				continue
			}
			evaluator.msgCh = msgCh
			break
		}
		evaluator.SubscriberLoop()
	}()
	// spawn a watcher for the incidents
	go func() {
		evaluator.EvaluateLoop()
	}()
	p.msgNode.AddSystemConfigListener(conditionId, cancel)
	return nil
}

func (p *Plugin) onDownstreamCapabilityConditionCreate(conditionId, conditionName string, condition *alertingv1.AlertConditionDownstreamCapability) error {
	lg := p.Logger.With("onCapabilityStatusCreate", conditionId)
	lg.Debugf("received condition update: %v", condition)
	jsCtx, cancel := context.WithCancel(p.Ctx)
	nc := p.natsConn.Get()
	js, err := nc.JetStream()
	if err != nil {
		cancel()
		return err
	}
	lg.Debugf("Creating agent disconnect with timeout %s", condition.GetFor().AsDuration())
	evaluator := &InternalConditionEvaluator[*corev1.ClusterHealthStatus]{
		lg:                 lg,
		conditionId:        conditionId,
		conditionName:      conditionName,
		storageNode:        p.storageNode,
		evaluationCtx:      jsCtx,
		parentCtx:          p.Ctx,
		cancelEvaluation:   cancel,
		evaluateDuration:   condition.GetFor().AsDuration(),
		alertmanagerlabels: map[string]string{},
		healthOnMessage: func(h *corev1.ClusterHealthStatus) (healthy bool, ts *timestamppb.Timestamp) {
			healthy = true
			lg.Debugf("found health conditions %v", h.HealthStatus.Health.Conditions)
			for _, s := range h.HealthStatus.Health.Conditions {
				for _, badState := range condition.GetCapabilityState() {
					if strings.Contains(s, badState) {
						healthy = false
						break
					}
				}

			}
			return healthy, h.HealthStatus.Status.Timestamp
		},
		triggerHook: func(ctx context.Context, conditionId string, labels map[string]string) {
			_, _ = p.TriggerAlerts(ctx, &alertingv1.TriggerAlertsRequest{
				ConditionId: &corev1.Reference{Id: conditionId},
				Annotations: labels,
			})
		},
	}
	// handles re-entrant conditions
	evaluator.CalculateInitialState()
	go func() {
		defer cancel() // cancel parent context, if we return (non-recoverable)
		for {
			err = natsutil.NewPersistentStream(js, shared.NewAlertingHealthStream())
			if err != nil {
				lg.Errorf("alerting disconnect stream does not exist and cannot be created %s", err)
				continue
			}
			agentId := condition.GetClusterId().Id
			msgCh := make(chan *nats.Msg, 32)
			sub, err := js.ChanSubscribe(shared.NewHealthStatusSubject(agentId), msgCh)
			defer sub.Unsubscribe()
			if err != nil {
				lg.Errorf("failed  to subscribe to %s : %s", shared.NewAgentDisconnectSubject(agentId), err)
				continue
			}
			evaluator.msgCh = msgCh
			break
		}
		evaluator.SubscriberLoop()
	}()
	// spawn a watcher for the incidents
	go func() {
		evaluator.EvaluateLoop()
	}()
	p.msgNode.AddSystemConfigListener(conditionId, cancel)
	return nil
}

func (p *Plugin) onCortexClusterStatusCreate(conditionId string, cond interface{}) context.CancelFunc {
	//lg := p.Logger.With("onCortexClusterStatusCreate", conditionId)
	/*msgCtx*/
	_, cancel := context.WithCancel(p.Ctx)
	var firingLock sync.RWMutex
	//currentlyFiring := false

	// spawn subscription/ aggregation stream
	go func() {
		defer cancel()
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker.C:
				select {
				default:
					for el := range p.msgNode.GetWatcher(conditionId) {
						if el == nil {
							continue
						}
						firingLock.RLock()
						// TODO
						//err := p.storageNode.AddToCortexClusterStatusIncidentTracker(msgCtx, conditionId, alertstorage.CortexClusterStatusIncidentStep{
						//	AlertFiring:   currentlyFiring,
						//	ClusterStatus: el.ClusterStatus,
						//})
						//if err != nil {
						//	lg.Error(err)
						//}
						firingLock.RUnlock()
					}
				}
			}
		}
	}()

	// spawn a watcher for triggering alerts
	go func() {
		defer cancel()
		ticker := time.NewTicker(time.Second * 10)
		defer ticker.Stop()
		for {
			select {
			case <-p.Ctx.Done():
				return
			case <-ticker.C:
				// get storage tracker
			}
		}
	}()

	return cancel
}
