package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AmmannChristian/nist-800-90b/internal/metrics"
	"github.com/AmmannChristian/nist-800-90b/internal/middleware"
	"github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

// GRPCServer implements the EntropyService gRPC interface.
type GRPCServer struct {
	pb.UnimplementedEntropyServiceServer
	svc *EntropyService
}

// NewGRPCServer creates a new GRPCServer instance.
func NewGRPCServer(svc *EntropyService) *GRPCServer {
	return &GRPCServer{
		svc: svc,
	}
}

// AssessEntropy handles gRPC requests for entropy assessment.
func (s *GRPCServer) AssessEntropy(ctx context.Context, req *pb.EntropyAssessmentRequest) (*pb.EntropyAssessmentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	if len(req.Data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data cannot be empty")
	}

	if req.BitsPerSymbol > 8 {
		return nil, status.Errorf(codes.InvalidArgument, "bits_per_symbol must be between 0 and 8, got %d", req.BitsPerSymbol)
	}

	if !req.IidMode && !req.NonIidMode {
		return nil, status.Error(codes.InvalidArgument, "either iid_mode or non_iid_mode must be enabled")
	}

	requestID := middleware.GetRequestID(ctx)
	testType := "mixed"
	if req.IidMode && !req.NonIidMode {
		testType = "IID"
	} else if req.NonIidMode && !req.IidMode {
		testType = "Non-IID"
	}
	startTime := time.Now()
	metrics.RecordRequest(testType)
	metrics.RecordDataSize(testType, len(req.Data))

	bits := int(req.BitsPerSymbol)
	var iidResults []*pb.EstimatorResult
	var nonIIDResults []*pb.EstimatorResult
	minEntropy := math.Inf(1)
	var usedBits uint32

	// IID path
	if req.IidMode {
		res, err := s.svc.AssessIID(req.Data, bits)
		if err != nil {
			metrics.RecordError("IID", "IID assessment failed")
			metrics.RecordDuration(testType, time.Since(startTime).Seconds())
			return nil, status.Errorf(codes.InvalidArgument, "IID assessment failed: %v", err)
		}
		minEntropy = math.Min(minEntropy, res.MinEntropy)
		usedBits = uint32(res.DataWordSize)
		iidResults = append(iidResults, &pb.EstimatorResult{
			Name:            "IID",
			EntropyEstimate: res.MinEntropy,
			Passed:          true,
			Details: map[string]float64{
				"h_original":  res.HOriginal,
				"h_bitstring": res.HBitstring,
				"h_assessed":  res.HAssessed,
			},
			Description: fmt.Sprintf("IID minimum entropy estimate (request_id=%s)", requestID),
		})
	}

	// Non-IID path
	if req.NonIidMode {
		res, err := s.svc.AssessNonIID(req.Data, bits)
		if err != nil {
			metrics.RecordError("Non-IID", "Non-IID assessment failed")
			metrics.RecordDuration(testType, time.Since(startTime).Seconds())
			return nil, status.Errorf(codes.InvalidArgument, "Non-IID assessment failed: %v", err)
		}
		minEntropy = math.Min(minEntropy, res.MinEntropy)
		usedBits = uint32(res.DataWordSize)
		nonIIDResults = append(nonIIDResults, &pb.EstimatorResult{
			Name:            "Non-IID",
			EntropyEstimate: res.MinEntropy,
			Passed:          true,
			Details: map[string]float64{
				"h_original":  res.HOriginal,
				"h_bitstring": res.HBitstring,
				"h_assessed":  res.HAssessed,
			},
			Description: fmt.Sprintf("Non-IID minimum entropy estimate (request_id=%s)", requestID),
		})
	}

	if usedBits == 0 {
		usedBits = req.BitsPerSymbol
	}

	if !math.IsInf(minEntropy, 1) {
		metrics.RecordMinEntropy(testType, minEntropy)
	} else {
		minEntropy = 0
	}
	metrics.RecordDuration(testType, time.Since(startTime).Seconds())

	return &pb.EntropyAssessmentResponse{
		MinEntropy:        minEntropy,
		IidResults:        iidResults,
		NonIidResults:     nonIIDResults,
		Passed:            true,
		AssessmentSummary: "entropy assessment completed",
		SampleCount:       uint64(len(req.Data)),
		BitsPerSymbol:     usedBits,
	}, nil
}
