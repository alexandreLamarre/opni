package slo

import (
	"fmt"
	"strconv"

	"github.com/OpenSLO/oslo/pkg/manifest"
	oslo "github.com/OpenSLO/oslo/pkg/manifest/v1"
	"github.com/google/uuid"
	corev1 "github.com/rancher/opni/pkg/apis/core/v1"
)

// TODO: calendar, time windows & budgeting are not responsive to user input

const (
	// Metric Enum
	MetricLatency      = "latency"
	MetricAvailability = "availability"

	// Datasource Enum
	LoggingDatasource    = "logging"
	MonitoringDatasource = "monitoring"

	// Alert Enum
	AlertingBurnRate = "burnrate"
	AlertingBudget   = "budget"

	// Notification Enum
	NotifSlack = "slack"
	NotifMail  = "email"
	NotifPager = "pager"

	osloVersion = "openslo/v1"
)

// func (slo *ServiceLevelObjective) GetFormulaReference() *corev1.Reference {
// 	return &corev1.Reference{
// 		Id: slo.GetFormulaId(),
// 	}
// }

func (slo *ServiceLevelObjective) GetServiceReference() *corev1.Reference {
	return &corev1.Reference{
		Id: slo.GetServiceId(),
	}
}

/// Returns a list of all the components passed in the protobuf we need to translate to specs
/// @errors: slo must have an id
func (slo *ServiceLevelObjective) ParseToOpenSLO() ([]manifest.OpenSLOKind, error) {
	res := make([]manifest.OpenSLOKind, 0)
	// Parse to inline SLO/SLIs
	newSLOI := oslo.SLOSpec{
		Description:     slo.GetDescription(),
		Service:         slo.GetServiceId(),
		BudgetingMethod: "30d", //FIXME: user should set this?
	}
	// actual SLO/SLI query
	indicator, err := slo.ParseToIndicator()
	if err != nil {
		return res, err
	}
	newSLOI.Indicator = indicator

	// targets
	newSLOI.Objectives = slo.ParseToObjectives()
	wrapSLOI := oslo.SLO{
		ObjectHeader: manifest.ObjectHeader{APIVersion: osloVersion}, //FIXME: subject to change in near future API
		Spec:         newSLOI,
	}
	res = append(res, wrapSLOI)

	// Parse inline Alert Policies and Alert Notifications
	// TODO
	return res, nil
}

/// @note : for now only one indicator per SLO is supported
/// Indicator is OpenSLO's inline indicator
func (slo *ServiceLevelObjective) ParseToIndicator() (*oslo.SLIInline, error) {
	metadata := oslo.Metadata{
		Name: fmt.Sprintf("sli-%s-%s", slo.GetMetricType(), slo.GetName()),
	}
	metric_type := "placeholder-string"                                    // TODO
	metric_query_bad, metric_query_total, err := slo.fetchPreconfQueries() // TODO

	bad_metric := oslo.MetricSource{
		Type:             metric_type,
		MetricSourceSpec: map[string]string{"query": metric_query_bad}, // this is flimsy
	}
	total_metric := oslo.MetricSource{
		Type:             metric_type,
		MetricSourceSpec: map[string]string{"query": metric_query_total}, // this is flimsy
	}
	spec := oslo.SLISpec{
		RatioMetric: &oslo.RatioMetric{
			Counter: true, //MAYBE : should not always be true
			Bad:     &oslo.MetricSourceHolder{MetricSource: bad_metric},
			Total:   oslo.MetricSourceHolder{MetricSource: total_metric},
		},
	}
	SLI := oslo.SLIInline{
		Metadata: metadata,
		Spec:     spec,
	}
	return &SLI, err
}

func (slo *ServiceLevelObjective) ParseToObjectives() []oslo.Objective {
	objectives := make([]oslo.Objective, 0)
	for i, target := range slo.GetTargets() {
		newObjective := oslo.Objective{
			DisplayName:     slo.GetName() + "-target" + strconv.Itoa(i),
			Target:          float64(target.GetValueX100() / 100),
			TimeSliceWindow: "5m", // FIXME: User should set this?
		}
		objectives = append(objectives, newObjective)
	}
	return objectives
}

func (slo *ServiceLevelObjective) ParseToAlerts() ([]oslo.AlertPolicy, []oslo.AlertNotificationTarget, error) {
	policies := make([]oslo.AlertPolicy, 0)
	notifs := make([]oslo.AlertNotificationTarget, 0)

	for _, alert := range slo.Alerts {
		// Create noticiation targets, and then store their refs in policy specs
		target_spec := oslo.AlertNotificationTargetSpec{
			Target:      alert.GetNotificationTarget(),
			Description: alert.GetDescription(),
		}
		target := oslo.AlertNotificationTarget{
			ObjectHeader: manifest.ObjectHeader{APIVersion: osloVersion}, //FIXME: subject to change in near future API
			Spec:         target_spec,
		}
		// internal ref used by policy to find the notification target
		gen_uuid := uuid.New().String() //TODO add this to the target metadata
		print(target, gen_uuid)

		// Create policy with inline condition
		policy_spec := oslo.AlertPolicySpec{
			Description:         alert.GetDescription(),
			AlertWhenNoData:     alert.GetOnNoData(),
			AlertWhenBreaching:  alert.GetOnBreanch(),
			AlertWhenResolved:   alert.GetOnResolved(),
			Conditions:          []oslo.AlertPolicyCondition{},
			NotificationTargets: []oslo.AlertPolicyNotificationTarget{{TargetRef: gen_uuid}},
		}
		wrapPolicy := oslo.AlertPolicy{
			ObjectHeader: manifest.ObjectHeader{APIVersion: osloVersion}, //FIXME: subject to change in near future API
			Spec:         policy_spec,
		}
		policies = append(policies, wrapPolicy)

	}

	return policies, notifs, nil

}

func (slo *ServiceLevelObjective) fetchPreconfQueries() (string, string, error) {
	// TODO : implement
	return "placeholder-bad", "placeholder-good", nil
}
