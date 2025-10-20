package settings

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	reloadFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "soyplane",
			Subsystem: "settings",
			Name:      "reload_failures_total",
			Help:      "Number of configuration load failures partitioned by stage.",
		},
		[]string{"stage"},
	)
)

func init() {
	ctrmetrics.Registry.MustRegister(reloadFailures)
}

func recordReloadFailure(stage string, err error) {
	reloadFailures.WithLabelValues(stage).Inc()
	emitEvent(stage, err)
}

func emitEvent(stage string, err error) {
	eventMu.RLock()
	rec := eventRecorder
	ref := eventTarget
	eventMu.RUnlock()

	if rec == nil || ref == nil {
		return
	}

	reason := "Settings" + strings.Title(stage) + "Failed"
	rec.Eventf(ref, corev1.EventTypeWarning, reason, "settings %s failed: %v", stage, err)
}
