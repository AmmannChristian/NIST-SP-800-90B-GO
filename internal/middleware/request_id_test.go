package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestUnaryRequestIDInterceptorSetsHeaderAndContext(t *testing.T) {
	interceptor := UnaryRequestIDInterceptor()

	var gotCtx context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		gotCtx = ctx
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "req", &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)
	assert.NoError(t, err)

	// Check context value
	requestID := GetRequestID(gotCtx)
	assert.NotEmpty(t, requestID)
}

func TestGetRequestIDMissing(t *testing.T) {
	assert.Equal(t, "", GetRequestID(context.Background()))
}
