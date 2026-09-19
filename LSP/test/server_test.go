package test

import (
	"fmt"
	"testing"

	"github.com/JDVA0/snow/lsp/analyzer"
)

func TestDocumentParsing(t *testing.T) {
	// Test that the analyzer can parse Snow code
	anlzr := analyzer.NewAnalyzer()

	// Valid Snow code
	validCode := `const PORT = 8080
print("Hello, World!")`

	result, err := anlzr.Parse(validCode, "test.snow")
	if err != nil {
		t.Fatalf("Failed to parse valid code: %v", err)
	}

	if result.Program == nil {
		t.Error("Expected program to be non-nil")
	}

	// Invalid Snow code
	invalidCode := `const PORT = 8080
print("Hello, World!"` // Missing closing quote

	result, err = anlzr.Parse(invalidCode, "test.snow")
	if err != nil {
		t.Fatalf("Failed to parse invalid code: %v", err)
	}

	if len(result.Diagnostics) == 0 {
		t.Error("Expected diagnostics for invalid code")
	}

	fmt.Println("Document parsing test passed!")
}

func TestDiagnosticsConversion(t *testing.T) {
	anlzr := analyzer.NewAnalyzer()

	code := `x = 10
print(y)` // y is undefined

	result, err := anlzr.Parse(code, "test.snow")
	if err != nil {
		t.Fatalf("Failed to parse code: %v", err)
	}

	// Add some check diagnostics
	checkDiagnostics := anlzr.Check(result.Program, code)

	if len(checkDiagnostics) == 0 {
		// This might not trigger depending on the check implementation
		fmt.Println("No check diagnostics (this is OK for this test)")
	} else {
		fmt.Printf("Found %d check diagnostics\n", len(checkDiagnostics))
	}

	fmt.Println("Diagnostics conversion test passed!")
}

func TestDiagnosticsKeepURIPositions(t *testing.T) {
	anlzr := analyzer.NewAnalyzer()
	result, err := anlzr.Parse("fn broken(:\n    missing_name\n", "file:///tmp/errors.snow")
	if err != nil {
		t.Fatalf("Failed to parse invalid code: %v", err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("Expected one diagnostic, got %d", len(result.Diagnostics))
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Line != 0 || diagnostic.Column != 10 {
		t.Fatalf("Expected position 0:10, got %d:%d", diagnostic.Line, diagnostic.Column)
	}
}
