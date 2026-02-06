// Package middleware provides gRPC server interceptors for cross-cutting
// concerns such as request identification.
package middleware

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// UnaryRequestIDInterceptor returns a gRPC unary interceptor that generates a
// UUID v4 request ID, stores it in the context, and sends it back to the client
// via the "x-request-id" response header.
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

// GetRequestID extracts the request ID from the context. It returns an empty
// string if no request ID has been set.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
