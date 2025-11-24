package entropy

// TestType represents the type of entropy test performed
type TestType int

const (
	// IID represents Independent and Identically Distributed test
	IID TestType = iota
	// NonIID represents Non-IID test
	NonIID
)

// String returns the string representation of TestType
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

// Result contains the entropy assessment results
type Result struct {
	MinEntropy   float64  // Minimum entropy estimate
	HOriginal    float64  // Entropy from original symbols
	HBitstring   float64  // Entropy from bitstring
	HAssessed    float64  // Assessed entropy value
	DataWordSize int      // Bits per symbol
	TestType     TestType // Type of test performed
}

// Assessment configuration
type Assessment struct {
	verbose int
}

// NewAssessment creates a new Assessment instance with default configuration
func NewAssessment() *Assessment {
	return &Assessment{
		verbose: 1, // Normal verbosity
	}
}

// SetVerbose sets the verbosity level
// 0 = quiet, 1 = normal, 2 = verbose, 3 = very verbose
func (a *Assessment) SetVerbose(level int) {
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	a.verbose = level
}

// GetVerbose returns the current verbosity level
func (a *Assessment) GetVerbose() int {
	return a.verbose
}
