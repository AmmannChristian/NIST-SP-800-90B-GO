package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// UnaryRequestIDInterceptor adds a request ID to the context and response metadata.
func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		requestID := uuid.New().String()

		ctx = context.WithValue(ctx, requestIDKey, requestID)

		md := metadata.Pairs("x-request-id", requestID)
		_ = grpc.SetHeader(ctx, md) // best effort; do not fail the request

		return handler(ctx, req)
	}
}

// GetRequestID returns the request ID from context if available.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
