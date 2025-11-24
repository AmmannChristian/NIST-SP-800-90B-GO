package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	version = "1.0.0"
)

// JSONOutput represents the JSON output format
type JSONOutput struct {
	Version       string  `json:"version"`
	Filename      string  `json:"filename"`
	TestType      string  `json:"test_type"`
	BitsPerSymbol int     `json:"bits_per_symbol"`
	DataSize      int     `json:"data_size"`
	MinEntropy    float64 `json:"min_entropy"`
	HOriginal     float64 `json:"h_original,omitempty"`
	HBitstring    float64 `json:"h_bitstring,omitempty"`
	HAssessed     float64 `json:"h_assessed"`
	ErrorCode     int     `json:"error_code"`
	ErrorMessage  string  `json:"error_message,omitempty"`
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func writeJSON(filename string, data interface{}) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
		os.Exit(1)
	}
}
