package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

func TestAssessEntropyValidation(t *testing.T) {
	server := NewGRPCServer(NewService())

	tests := []struct {
		name string
		req  *pb.Sp80090BAssessmentRequest
		code codes.Code
	}{
		{
			name: "nil request",
			req:  nil,
			code: codes.InvalidArgument,
		},
		{
			name: "empty data",
			req: &pb.Sp80090BAssessmentRequest{
				Data:          []byte{},
				IidMode:       true,
				NonIidMode:    false,
				BitsPerSymbol: 8,
			},
			code: codes.InvalidArgument,
		},
		{
			name: "bits too high",
			req: &pb.Sp80090BAssessmentRequest{
				Data:          []byte{1, 2, 3},
				IidMode:       true,
				NonIidMode:    false,
				BitsPerSymbol: nineBits(),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "no mode selected",
			req: &pb.Sp80090BAssessmentRequest{
				Data:          []byte{1, 2, 3},
				BitsPerSymbol: 8,
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.AssessEntropy(context.Background(), tt.req)
			if assert.Error(t, err) {
				st, _ := status.FromError(err)
				assert.Equal(t, tt.code, st.Code())
			}
		})
	}
}

// nineBits returns a uint32 value exceeding the valid bits-per-symbol range,
// used to avoid a compile-time constant overflow warning in test literals.
func nineBits() uint32 {
	return 9
}
