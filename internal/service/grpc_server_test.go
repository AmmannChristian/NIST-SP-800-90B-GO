package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

func TestAssessEntropyValidation(t *testing.T) {
	server := NewGRPCServer(NewService())

	tests := []struct {
		name string
		req  *pb.EntropyAssessmentRequest
		code codes.Code
	}{
		{
			name: "nil request",
			req:  nil,
			code: codes.InvalidArgument,
		},
		{
			name: "empty data",
			req: &pb.EntropyAssessmentRequest{
				Data:          []byte{},
				IidMode:       true,
				NonIidMode:    false,
				BitsPerSymbol: 8,
			},
			code: codes.InvalidArgument,
		},
		{
			name: "bits too high",
			req: &pb.EntropyAssessmentRequest{
				Data:          []byte{1, 2, 3},
				IidMode:       true,
				NonIidMode:    false,
				BitsPerSymbol: nineBits(),
			},
			code: codes.InvalidArgument,
		},
		{
			name: "no mode selected",
			req: &pb.EntropyAssessmentRequest{
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

func nineBits() uint32 {
	return 9
}

// Success paths rely on the teststub build tag to avoid CGO.
func TestAssessEntropySuccessModes(t *testing.T) {
	server := NewGRPCServer(NewService())
	data := []byte{1, 2, 3, 4}

	// IID only
	resp, err := server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          data,
		BitsPerSymbol: 8,
		IidMode:       true,
		NonIidMode:    false,
	})
	require.NoError(t, err)
	assert.Len(t, resp.IidResults, 1)
	assert.Len(t, resp.NonIidResults, 0)
	assert.Equal(t, uint32(8), resp.BitsPerSymbol)
	assert.Greater(t, resp.MinEntropy, 0.0)

	// Non-IID only
	resp, err = server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          data,
		BitsPerSymbol: 8,
		IidMode:       false,
		NonIidMode:    true,
	})
	require.NoError(t, err)
	assert.Len(t, resp.IidResults, 0)
	assert.Len(t, resp.NonIidResults, 1)

	// Mixed mode
	resp, err = server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          data,
		BitsPerSymbol: 8,
		IidMode:       true,
		NonIidMode:    true,
	})
	require.NoError(t, err)
	assert.Len(t, resp.IidResults, 1)
	assert.Len(t, resp.NonIidResults, 1)
	assert.True(t, resp.Passed)
}

func TestAssessEntropyUsedBitsFallback(t *testing.T) {
	server := NewGRPCServer(NewService())
	data := []byte{1, 2, 3, 4}

	resp, err := server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          data,
		BitsPerSymbol: 0,
		IidMode:       true,
		NonIidMode:    false,
	})
	require.NoError(t, err)
	assert.Len(t, resp.IidResults, 1)
	assert.Equal(t, uint32(0), resp.BitsPerSymbol)
}

func TestAssessEntropyIIDError(t *testing.T) {
	server := NewGRPCServer(NewService())

	_, err := server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          []byte{0xFF, 1, 2},
		BitsPerSymbol: 8,
		IidMode:       true,
		NonIidMode:    false,
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "IID assessment failed")
}

func TestAssessEntropyNonIIDError(t *testing.T) {
	server := NewGRPCServer(NewService())

	_, err := server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          []byte{0xFF, 1, 2},
		BitsPerSymbol: 8,
		IidMode:       false,
		NonIidMode:    true,
	})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "Non-IID assessment failed")
}

func TestAssessEntropyInfinityFallback(t *testing.T) {
	server := NewGRPCServer(NewService())

	resp, err := server.AssessEntropy(context.Background(), &pb.EntropyAssessmentRequest{
		Data:          []byte{0xEE, 1, 2},
		BitsPerSymbol: 8,
		IidMode:       false,
		NonIidMode:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(0), resp.MinEntropy)
	assert.Equal(t, uint32(8), resp.BitsPerSymbol)
}
