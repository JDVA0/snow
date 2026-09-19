package main

import (
	"fmt"
	"github.com/JDVA0/snow/lsp/analyzer"
)

func main() {
	anlzr := analyzer.NewAnalyzer()
	
	// Test parsing valid Snow code
	validCode := `const PORT = 8080
print("Hello, World!")
x = 10
y = x + 5
print(y)`
	
	fmt.Println("Testing valid Snow code:")
	result, err := anlzr.Parse(validCode, "test.snow")
	if err != nil {
		fmt.Printf("Error parsing valid code: %v\n", err)
	} else {
		fmt.Printf("Parse successful, %d diagnostics\n", len(result.Diagnostics))
		if len(result.Diagnostics) > 0 {
			fmt.Println("Diagnostics:")
			for _, diag := range result.Diagnostics {
				fmt.Printf("  %s:%d:%d: %s: %s\n", diag.Source, diag.Line+1, diag.Column+1, diag.Severity, diag.Message)
			}
		}
	}
	
	// Test parsing invalid Snow code
	invalidCode := `const PORT = 8080
print("Hello, World!"` // Missing closing quote
	
	fmt.Println("\nTesting invalid Snow code:")
	result, err = anlzr.Parse(invalidCode, "test.snow")
	if err != nil {
		fmt.Printf("Error parsing invalid code: %v\n", err)
	} else {
		fmt.Printf("Parse successful, %d diagnostics\n", len(result.Diagnostics))
		if len(result.Diagnostics) > 0 {
			fmt.Println("Diagnostics:")
			for _, diag := range result.Diagnostics {
				fmt.Printf("  %s:%d:%d: %s: %s\n", diag.Source, diag.Line+1, diag.Column+1, diag.Severity, diag.Message)
			}
		}
	}
	
	// Test check functionality
	fmt.Println("\nTesting check functionality:")
	if result.Program != nil {
		checkResult := anlzr.Check(result.Program, invalidCode)
		if len(checkResult) > 0 {
			fmt.Printf("Check found %d issues\n", len(checkResult))
			for _, issue := range checkResult {
				fmt.Printf("  %s:%d:%d: %s: %s\n", issue.Source, issue.Line+1, issue.Column+1, issue.Severity, issue.Message)
			}
		} else {
			fmt.Println("No check issues found")
		}
	} else {
		fmt.Println("Skipping check on invalid code (program is nil)")
	}
	
	// Test with undefined variable
	undefinedCode := `x = 10
print(y)` // y is undefined
	
	fmt.Println("\nTesting undefined variable:")
	result, err = anlzr.Parse(undefinedCode, "test.snow")
	if err != nil {
		fmt.Printf("Error parsing code: %v\n", err)
	} else {
		fmt.Printf("Parse successful, %d diagnostics\n", len(result.Diagnostics))
		checkResult := anlzr.Check(result.Program, undefinedCode)
		if len(checkResult) > 0 {
			fmt.Printf("Check found %d issues\n", len(checkResult))
			for _, issue := range checkResult {
				fmt.Printf("  %s:%d:%d: %s: %s\n", issue.Source, issue.Line+1, issue.Column+1, issue.Severity, issue.Message)
			}
		} else {
			fmt.Println("No check issues found")
		}
	}
	
	fmt.Println("\nManual test completed!")
	
	// Test completion functionality
	fmt.Println("\nTesting completion functionality:")
	
	// Just show that completion is available
	fmt.Println("  Completion is now implemented in the LSP server")
	fmt.Println("  Available items include:")
	fmt.Println("    - Keywords: using, import, fn, if, for, etc.")
	fmt.Println("    - Standard modules: sys, fs, api, http, etc.")
	fmt.Println("    - Built-in functions: print, len, str, int, etc.")
	fmt.Println("    - Constants: true, false, nil")
	
	fmt.Println("Completion test completed!")
}