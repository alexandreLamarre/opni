package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/rancher/opni/pkg/alerting/drivers/backend"
	"github.com/rancher/opni/pkg/alerting/drivers/routing"
	"github.com/rancher/opni/pkg/alerting/shared"
	alertingv1 "github.com/rancher/opni/pkg/apis/alerting/v1"
	corev1 "github.com/rancher/opni/pkg/apis/core/v1"
	"github.com/rancher/opni/pkg/util"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// groups notifications by the duration "context" that is sending them
// resolveContext's value will depend on the values the notification routing tree holds
var resolveContext time.Duration

func init() {
	notificationSubtree := routing.NotificationSubTreeValues()
	maxDur := time.Duration(0)
	for _, node := range notificationSubtree {
		dur := node.B.InitialDelay + node.B.ThrottlingDuration
		if dur > maxDur {
			maxDur = dur
		}
	}
	resolveContext = time.Microsecond * time.Duration((math.Round(float64(maxDur.Microseconds()) * 1.2)))
}

// --- Trigger ---
func (p *Plugin) TriggerAlerts(ctx context.Context, req *alertingv1.TriggerAlertsRequest) (*alertingv1.TriggerAlertsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	lg := p.Logger.With("Handler", "TriggerAlerts")
	lg.Debugf("Received request to trigger alerts  on condition %s", req.GetConditionId())
	lg.Debugf("Received alert annotations : %s", req.Annotations)

	options, err := p.opsNode.GetRuntimeOptions(ctx)
	if err != nil {
		return nil, err
	}
	// dispatch with alert condition id to alert endpoint id, by obeying rate limiting from AM
	availableEndpoint, err := p.opsNode.GetAvailableEndpoint(ctx, &options)
	if err != nil {
		return nil, err
	}
	// This logic is intended to
	// 1) Provide a safeguard to ensure that external callers of the API will not cause nil pointer map dereferences in the AlertManager adapter logic
	// 2) Set the required information if it is not already embedded in the postable Alert's information.
	// 3) Ensure that callers of the API are efficiently & uniquely routed when posted to our construction
	// of the AlertManager routing tree, regardless if there are specific routes & receivers to handle this set of information
	if _, ok := req.Labels[req.Namespace]; !ok {
		req.Labels[req.Namespace] = req.ConditionId.Id
	}

	if _, ok := req.Labels[alertingv1.NotificationPropertyOpniUuid]; !ok {
		req.Labels[alertingv1.NotificationPropertyOpniUuid] = req.ConditionId.Id
	}

	if _, ok := req.Annotations[shared.BackendConditionNameLabel]; !ok {
		req.Annotations[shared.BackendConditionNameLabel] = req.ConditionName
	}

	apiNode := backend.NewAlertManagerPostAlertClient(
		ctx,
		availableEndpoint,
		backend.WithLogger(lg),
		backend.WithExpectClosure(backend.NewExpectStatusOk()),
		backend.WithPostAlertBody(req.ConditionId.Id, req.Labels, req.Annotations),
	)

	err = apiNode.DoRequest()
	if err != nil {
		return nil, err
	}
	return &alertingv1.TriggerAlertsResponse{}, nil
}

func (p *Plugin) ResolveAlerts(ctx context.Context, req *alertingv1.ResolveAlertsRequest) (*alertingv1.ResolveAlertsResponse, error) {
	lg := p.Logger.With("Handler", "ResolveAlerts")
	if err := req.Validate(); err != nil {
		return nil, err
	}
	options, err := p.opsNode.GetRuntimeOptions(ctx)
	if err != nil {
		return nil, err
	}
	// dispatch with alert condition id to alert endpoint id, by obeying rate limiting from AM
	availableEndpoint, err := p.opsNode.GetAvailableEndpoint(ctx, &options)
	if err != nil {
		return nil, err
	}
	// This logic is intended to
	// 1) Provide a safeguard to ensure that external callers of the API will not cause nil pointer map dereferences in the AlertManager adapter logic
	// 2) Set the required information if it is not already embedded in the postable Alert's information.
	// 3) Ensure that callers of the API are efficiently & uniquely routed when posted to our construction
	// of the AlertManager routing tree, regardless if there are specific routes & receivers to handle this set of information
	if _, ok := req.Labels[req.Namespace]; !ok {
		req.Labels[req.Namespace] = req.ConditionId.Id
	}

	if _, ok := req.Labels[alertingv1.NotificationPropertyOpniUuid]; !ok {
		req.Labels[alertingv1.NotificationPropertyOpniUuid] = req.ConditionId.Id
	}

	if _, ok := req.Annotations[shared.BackendConditionNameLabel]; !ok {
		req.Annotations[shared.BackendConditionNameLabel] = req.ConditionName
	}

	apiNode := backend.NewAlertManagerPostAlertClient(
		ctx,
		availableEndpoint,
		backend.WithLogger(lg),
		backend.WithExpectClosure(backend.NewExpectStatusOk()),
		backend.WithPostResolveAlertBody(req.ConditionId.Id, req.Labels, req.Annotations),
	)
	err = apiNode.DoRequest()
	if err != nil {
		return nil, err
	}
	return &alertingv1.ResolveAlertsResponse{}, nil
}

func (p *Plugin) PushNotification(ctx context.Context, req *alertingv1.Notification) (*emptypb.Empty, error) {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, ok := req.GetProperties()[alertingv1.NotificationPropertyDedupeKey]; !ok {
		req.Properties[alertingv1.NotificationPropertyDedupeKey] = util.HashStrings([]string{req.Title, req.Body})
	}

	options, err := p.opsNode.GetRuntimeOptions(ctx)
	if err != nil {
		return nil, err
	}
	// dispatch with alert condition id to alert endpoint id, by obeying rate limiting from AM
	availableEndpoint, err := p.opsNode.GetAvailableEndpoint(ctx, &options)
	if err != nil {
		return nil, err
	}
	routingLabels := req.GetRoutingLabels()
	apiNode := backend.NewAlertManagerPostAlertClient(
		ctx,
		availableEndpoint,
		backend.WithLogger(p.Logger),
		backend.WithExpectClosure(backend.NewExpectStatusOk()),
		backend.WithPostNotificationBody(
			routingLabels[alertingv1.NotificationPropertyOpniUuid],
			routingLabels,
			req.GetRoutingAnnotations(),
			resolveContext,
		),
	)
	return &emptypb.Empty{}, apiNode.DoRequest()
}

func (p *Plugin) ListNotifications(ctx context.Context, req *alertingv1.ListNotificationRequest) (*alertingv1.ListMessageResponse, error) {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	options, err := p.opsNode.GetRuntimeOptions(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME: the messages returned by this endpoint will not always be consistent
	// within the HA vanilla AlertManager,
	// move this logic to cortex AlertManager member set when applicable
	availableEndpoint, err := p.opsNode.GetAvailableCacheEndpoint(ctx, &options)
	if err != nil {
		return nil, err
	}

	resp := &alertingv1.ListMessageResponse{}
	apiNode := backend.NewListNotificationMessagesClient(
		ctx,
		availableEndpoint,
		backend.WithLogger(p.Logger),
		backend.WithExpectClosure(func(incoming *http.Response) error {
			defer incoming.Body.Close()
			if incoming.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code %d", incoming.StatusCode)
			}
			return json.NewDecoder(incoming.Body).Decode(resp)
		}),
		backend.WithPostProto(req),
		backend.WithDefaultRetrier(),
	)

	if err := apiNode.DoRequest(); err != nil {
		return nil, err
	}
	slices.SortFunc(resp.Items, func(a, b *alertingv1.MessageInstance) bool {
		return a.ReceivedAt.AsTime().Before(b.ReceivedAt.AsTime())
	})
	return resp, nil
}

func (p *Plugin) ListAlarmMessages(ctx context.Context, req *alertingv1.ListAlarmMessageRequest) (*alertingv1.ListMessageResponse, error) {
	req.Sanitize()
	if err := req.Validate(); err != nil {
		return nil, err
	}
	options, err := p.opsNode.GetRuntimeOptions(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME: the messages returned by this endpoint will not always be consistent
	// within the HA vanilla AlertManager,
	// move this logic to cortex AlertManager member set when applicable
	availableEndpoint, err := p.opsNode.GetAvailableCacheEndpoint(ctx, &options)
	if err != nil {
		return nil, err
	}

	cond, err := p.GetAlertCondition(ctx, &corev1.Reference{
		Id: req.ConditionId,
	})
	if err != nil {
		return nil, err
	}
	consistencyInterval := durationpb.New(time.Second * 30)
	if cond.AttachedEndpoints != nil {
		consistencyInterval = cond.AttachedEndpoints.InitialDelay
	}
	req.End = timestamppb.New(req.End.AsTime().Add(consistencyInterval.AsDuration()))

	resp := &alertingv1.ListMessageResponse{}
	apiNode := backend.NewListAlarmMessagesClient(
		ctx,
		availableEndpoint,
		backend.WithLogger(p.Logger),
		backend.WithExpectClosure(func(incoming *http.Response) error {
			defer incoming.Body.Close()
			if incoming.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected status code %d", incoming.StatusCode)
			}
			return json.NewDecoder(incoming.Body).Decode(resp)
		}),
		backend.WithPostProto(req),
		backend.WithDefaultRetrier(),
	)
	if err := apiNode.DoRequest(); err != nil {
		return nil, err
	}
	slices.SortFunc(resp.Items, func(a, b *alertingv1.MessageInstance) bool {
		return a.ReceivedAt.AsTime().Before(b.ReceivedAt.AsTime())
	})
	return resp, nil
}

func (p *Plugin) ListRoutingRelationships(ctx context.Context, _ *emptypb.Empty) (*alertingv1.ListRoutingRelationshipsResponse, error) {
	cond, err := p.ListAlertConditions(ctx, &alertingv1.ListAlertConditionRequest{})
	if err != nil {
		return nil, err
	}
	relationships := map[string]*corev1.ReferenceList{}
	for _, c := range cond.Items {
		if c.AlertCondition.AttachedEndpoints != nil && len(c.AlertCondition.AttachedEndpoints.Items) > 0 {
			refs := &corev1.ReferenceList{
				Items: lo.Map(
					c.AlertCondition.AttachedEndpoints.Items,
					func(endp *alertingv1.AttachedEndpoint, _ int) *corev1.Reference {
						return &corev1.Reference{
							Id: endp.EndpointId,
						}
					}),
			}
			relationships[c.Id.Id] = refs
		}
	}
	return &alertingv1.ListRoutingRelationshipsResponse{
		RoutingRelationships: relationships,
	}, nil
}
