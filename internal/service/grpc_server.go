package service

import (
	"context"
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/AmmannChristian/nist-800-90b/internal/entropy"
	"github.com/AmmannChristian/nist-800-90b/internal/metrics"
	pb "github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

const Version = "2.0.0"

// GRPCServer implements the Sp80090BAssessmentService gRPC interface.
type GRPCServer struct {
	pb.UnimplementedSp80090BAssessmentServiceServer
	svc *EntropyService
}

// NewGRPCServer creates a new GRPCServer instance.
func NewGRPCServer(svc *EntropyService) *GRPCServer {
	return &GRPCServer{
		svc: svc,
	}
}

// AssessEntropy handles gRPC requests for NIST SP 800-90B entropy assessment.
func (s *GRPCServer) AssessEntropy(ctx context.Context, req *pb.Sp80090BAssessmentRequest) (*pb.Sp80090BAssessmentResponse, error) {
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

	_ = ctx // requestID no longer needed in response
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
	var iidResults []*pb.Sp80090BEstimatorResult
	var nonIIDResults []*pb.Sp80090BEstimatorResult
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
		iidResults = convertEstimatorsToProto(res.Estimators)
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
		nonIIDResults = convertEstimatorsToProto(res.Estimators)
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

	return &pb.Sp80090BAssessmentResponse{
		MinEntropy:        minEntropy,
		IidResults:        iidResults,
		NonIidResults:     nonIIDResults,
		Passed:            true,
		AssessmentSummary: "NIST SP 800-90B entropy assessment completed",
		SampleCount:       uint64(len(req.Data)),
		BitsPerSymbol:     usedBits,
	}, nil
}

// convertEstimatorsToProto converts Go estimator results to protobuf messages
func convertEstimatorsToProto(estimators []entropy.EstimatorResult) []*pb.Sp80090BEstimatorResult {
	if len(estimators) == 0 {
		return nil
	}

	results := make([]*pb.Sp80090BEstimatorResult, len(estimators))
	for i, est := range estimators {
		details := make(map[string]float64)
		if est.IsEntropyValid {
			details["entropy_estimate"] = est.EntropyEstimate
		}

		description := est.Name
		if est.IsEntropyValid {
			description += " entropy estimator"
		} else {
			description += " statistical test"
		}

		results[i] = &pb.Sp80090BEstimatorResult{
			Name:            est.Name,
			EntropyEstimate: est.EntropyEstimate,
			Passed:          est.Passed,
			Details:         details,
			Description:     description,
		}
	}
	return results
}
