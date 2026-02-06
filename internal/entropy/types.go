// Package entropy provides the core types and assessment logic for NIST SP
// 800-90B entropy estimation. It defines the IID and Non-IID test modes,
// result structures, and the Assessment facade that delegates to the underlying
// C++ reference implementation via a CGO bridge.
package entropy

// TestType represents the type of entropy test performed.
type TestType int

const (
	// IID represents the Independent and Identically Distributed test mode.
	IID TestType = iota
	// NonIID represents the Non-IID test mode.
	NonIID
)

// String returns the string representation of TestType.
func (t TestType) String() string {
	switch t {
	case IID:
		return "IID"
	case NonIID:
		return "Non-IID"
	default:
		return "Unknown"
	}
}

// EstimatorResult contains the output of a single NIST SP 800-90B entropy
// estimator or statistical test. When IsEntropyValid is false, the
// EntropyEstimate field is set to -1.0 and should be disregarded.
type EstimatorResult struct {
	Name            string  // Estimator name (e.g., "Most Common Value")
	EntropyEstimate float64 // Entropy estimate in bits per sample, or -1.0 if not applicable
	Passed          bool    // Whether the test passed
	IsEntropyValid  bool    // Indicates whether EntropyEstimate holds a meaningful value
}

// Result contains the aggregate entropy assessment output. HOriginal is the
// per-sample entropy estimated from the original symbol alphabet, HBitstring
// is derived from the binary expansion, and HAssessed is the conservative
// minimum of both scaled to the word size. MinEntropy equals HAssessed.
type Result struct {
	MinEntropy   float64  // Minimum entropy estimate in bits per sample
	HOriginal    float64  // Entropy from original symbols
	HBitstring   float64  // Entropy from bitstring representation
	HAssessed    float64  // Final assessed entropy (min of original and bitstring)
	DataWordSize int      // Bits per symbol used in the assessment
	TestType     TestType // IID or NonIID

	Estimators []EstimatorResult // Individual estimator results
}

// Assessment holds configuration for entropy estimation and serves as the
// primary entry point for running IID and Non-IID assessments.
type Assessment struct {
	verbose int
}

// NewAssessment creates a new Assessment instance with default configuration.
func NewAssessment() *Assessment {
	return &Assessment{
		verbose: 1, // Normal verbosity
	}
}

// SetVerbose sets the verbosity level. Values are clamped to the range [0, 3]:
// 0 = quiet, 1 = normal, 2 = verbose, 3 = very verbose.
func (a *Assessment) SetVerbose(level int) {
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	a.verbose = level
}

// GetVerbose returns the current verbosity level.
func (a *Assessment) GetVerbose() int {
	return a.verbose
}
