package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts the total number of entropy assessment requests
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "entropy_requests_total",
			Help: "Total number of entropy assessment requests",
		},
		[]string{"test_type"}, // iid or non_iid
	)

	// DurationSeconds measures the duration of entropy assessments
	DurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "entropy_duration_seconds",
			Help:    "Duration of entropy assessment in seconds",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
		},
		[]string{"test_type"}, // iid or non_iid
	)

	// ErrorsTotal counts the total number of errors
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "entropy_errors_total",
			Help: "Total number of entropy assessment errors",
		},
		[]string{"test_type", "error_type"}, // test_type and error classification
	)

	// DataSizeBytes tracks the size of data being processed
	DataSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "entropy_data_size_bytes",
			Help:    "Size of data being assessed in bytes",
			Buckets: prometheus.ExponentialBuckets(1024, 10, 6), // 1KB to ~1MB
		},
		[]string{"test_type"},
	)

	// MinEntropyValue tracks the min entropy values calculated
	MinEntropyValue = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "entropy_min_entropy_value",
			Help:    "Minimum entropy values calculated",
			Buckets: prometheus.LinearBuckets(0, 0.5, 17), // 0 to 8 in 0.5 increments
		},
		[]string{"test_type"},
	)
)

// RecordRequest increments the request counter for a given test type
func RecordRequest(testType string) {
	RequestsTotal.WithLabelValues(testType).Inc()
}

// RecordDuration records the duration of an entropy assessment
func RecordDuration(testType string, duration float64) {
	DurationSeconds.WithLabelValues(testType).Observe(duration)
}

// RecordError increments the error counter
func RecordError(testType, errorType string) {
	ErrorsTotal.WithLabelValues(testType, errorType).Inc()
}

// RecordDataSize records the size of data being processed
func RecordDataSize(testType string, sizeBytes int) {
	DataSizeBytes.WithLabelValues(testType).Observe(float64(sizeBytes))
}

// RecordMinEntropy records a minimum entropy value
func RecordMinEntropy(testType string, value float64) {
	MinEntropyValue.WithLabelValues(testType).Observe(value)
}
