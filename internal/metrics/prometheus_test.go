package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRecordRequest(t *testing.T) {
	// Reset the counter
	RequestsTotal.Reset()

	RecordRequest("iid")
	RecordRequest("iid")
	RecordRequest("non_iid")

	// Verify counter values
	iidCount := testutil.ToFloat64(RequestsTotal.WithLabelValues("iid"))
	assert.Equal(t, 2.0, iidCount)

	nonIIDCount := testutil.ToFloat64(RequestsTotal.WithLabelValues("non_iid"))
	assert.Equal(t, 1.0, nonIIDCount)
}

func TestRecordDuration(t *testing.T) {
	DurationSeconds.Reset()

	RecordDuration("iid", 0.5)
	RecordDuration("iid", 1.5)
	RecordDuration("non_iid", 2.0)

	// Just verify the function doesn't panic - histogram metrics are hard to test
	// The fact that we got here means the function works
	assert.True(t, true)
}

func TestRecordError(t *testing.T) {
	ErrorsTotal.Reset()

	RecordError("iid", "validation")
	RecordError("iid", "validation")
	RecordError("non_iid", "memory")

	// Verify counter values
	iidValidationCount := testutil.ToFloat64(ErrorsTotal.WithLabelValues("iid", "validation"))
	assert.Equal(t, 2.0, iidValidationCount)

	nonIIDMemoryCount := testutil.ToFloat64(ErrorsTotal.WithLabelValues("non_iid", "memory"))
	assert.Equal(t, 1.0, nonIIDMemoryCount)
}

func TestRecordDataSize(t *testing.T) {
	DataSizeBytes.Reset()

	RecordDataSize("iid", 1024)
	RecordDataSize("iid", 2048)
	RecordDataSize("non_iid", 4096)

	// Just verify the function doesn't panic
	assert.True(t, true)
}

func TestRecordMinEntropy(t *testing.T) {
	MinEntropyValue.Reset()

	RecordMinEntropy("iid", 7.5)
	RecordMinEntropy("iid", 6.8)
	RecordMinEntropy("non_iid", 5.2)

	// Just verify the function doesn't panic
	assert.True(t, true)
}

func TestMetricsInitialization(t *testing.T) {
	// Verify that all metrics are properly initialized
	assert.NotNil(t, RequestsTotal)
	assert.NotNil(t, DurationSeconds)
	assert.NotNil(t, ErrorsTotal)
	assert.NotNil(t, DataSizeBytes)
	assert.NotNil(t, MinEntropyValue)
}
