package workers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const MetricsNamespace string = "go_worker"

var (
	TaskEnqueueMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "task_enqueue",
		Help:      "number of task enqueue, grouped by queue name",
	}, []string{"name"})

	TaskDequeueMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "task_dequeue",
		Help:      "number of task dequeue, grouped by queue name",
	}, []string{"name"})

	TaskProcessedMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricsNamespace,
		Name:      "task_processed",
		Help:      "Total number of tasks processed.",
	}, []string{"name", "error"})
)

func init() {
	prometheus.MustRegister(TaskEnqueueMetric, TaskDequeueMetric, TaskProcessedMetric)
}

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
