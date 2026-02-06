package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/AmmannChristian/nist-800-90b/internal/entropy"
)

// runCLI parses command-line arguments, reads input data from a file or stdin,
// and performs an IID or Non-IID entropy assessment. It returns an exit code:
// 0 on success, 1 on assessment error, or 2 on argument validation failure.
func runCLI(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ea_tool", flag.ContinueOnError)
	fs.SetOutput(stderr)

	iid := fs.Bool("iid", false, "Run IID (Independent and Identically Distributed) test")
	nonIID := fs.Bool("non-iid", false, "Run Non-IID test")
	bits := fs.Int("bits", 0, "Bits per symbol (1-8), 0 for auto-detect")
	verbose := fs.Int("verbose", 1, "Verbosity level (0=quiet, 1=normal, 2=verbose, 3=very verbose)")
	outputFile := fs.String("output", "", "Output file for JSON results")
	showVersion := fs.Bool("version", false, "Show version information")

	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options] <file>\n\n", fs.Name())
		fmt.Fprintf(stderr, "Entropy Assessment Tool for NIST SP800-90B\n\n")
		fmt.Fprintf(stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExamples:\n")
		fmt.Fprintf(stderr, "  %s -non-iid -bits 8 data.bin\n", fs.Name())
		fmt.Fprintf(stderr, "  %s -iid -bits 1 data.bin -output result.json\n", fs.Name())
		fmt.Fprintf(stderr, "  cat data.bin | %s -non-iid -bits 8\n", fs.Name())
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "ea_tool version %s\n", version)
		return 0
	}

	if *iid == *nonIID {
		fmt.Fprintf(stderr, "Error: Must specify exactly one of -iid or -non-iid\n\n")
		fs.Usage()
		return 2
	}

	var testType entropy.TestType
	if *iid {
		testType = entropy.IID
	} else {
		testType = entropy.NonIID
	}

	if *bits < 0 || *bits > 8 {
		fmt.Fprintf(stderr, "Error: bits per symbol must be 0-8, got %d\n", *bits)
		return 2
	}

	var data []byte
	var filename string
	var err error

	if fs.NArg() == 0 {
		filename = "stdin"
		data, err = io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading from stdin: %v\n", err)
			return 1
		}
	} else {
		filename = fs.Arg(0)
		data, err = os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading file %s: %v\n", filename, err)
			return 1
		}
	}

	assessment := entropy.NewAssessment()
	assessment.SetVerbose(*verbose)

	var result *entropy.Result
	if testType == entropy.IID {
		result, err = assessment.AssessIID(data, *bits)
	} else {
		result, err = assessment.AssessNonIID(data, *bits)
	}

	jsonOut := JSONOutput{
		Version:       version,
		Filename:      filename,
		TestType:      testType.String(),
		BitsPerSymbol: *bits,
		DataSize:      len(data),
		ErrorCode:     0,
	}

	if err != nil {
		jsonOut.ErrorCode = 1
		jsonOut.ErrorMessage = err.Error()
		if *outputFile != "" {
			writeJSON(*outputFile, jsonOut)
		} else {
			fmt.Fprintf(stderr, "Error: %v\n", err)
		}
		return 1
	}

	jsonOut.MinEntropy = result.MinEntropy
	jsonOut.HOriginal = result.HOriginal
	jsonOut.HBitstring = result.HBitstring
	jsonOut.HAssessed = result.HAssessed

	if *outputFile != "" {
		writeJSON(*outputFile, jsonOut)
		if *verbose > 0 {
			fmt.Fprintf(stdout, "Results written to %s\n", *outputFile)
		}
	} else if *verbose >= 1 {
		fmt.Fprintf(stdout, "\nEntropy Assessment Results:\n")
		fmt.Fprintf(stdout, "  Test Type:       %s\n", testType)
		fmt.Fprintf(stdout, "  Bits/Symbol:     %d\n", result.DataWordSize)
		fmt.Fprintf(stdout, "  H_original:      %.6f\n", result.HOriginal)
		if result.HBitstring > 0 {
			fmt.Fprintf(stdout, "  H_bitstring:     %.6f\n", result.HBitstring)
		}
		fmt.Fprintf(stdout, "  H_assessed:      %.6f\n", result.HAssessed)
		fmt.Fprintf(stdout, "  Min Entropy:     %.6f\n", result.MinEntropy)
	}

	return 0
}
