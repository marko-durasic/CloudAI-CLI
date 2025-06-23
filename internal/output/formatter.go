package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Formatter handles output formatting
type Formatter struct {
	jsonOutput bool
}

// NewFormatter creates a new formatter
func NewFormatter(jsonOutput bool) *Formatter {
	return &Formatter{jsonOutput: jsonOutput}
}

// Result represents a query result
type Result struct {
	Query   string      `json:"query"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// FormatResult formats and outputs the result
func (f *Formatter) FormatResult(result *Result) error {
	if f.jsonOutput {
		return f.formatJSON(result)
	}
	return f.formatTable(result)
}

// formatJSON outputs result in JSON format
func (f *Formatter) formatJSON(result *Result) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// formatTable outputs result in table format
func (f *Formatter) formatTable(result *Result) error {
	if !result.Success {
		fmt.Printf("❌ Error: %s\n", result.Error)
		return nil
	}

	// TODO: Implement table formatting based on data type
	fmt.Printf("✅ Query: %s\n", result.Query)
	fmt.Printf("📊 Data: %+v\n", result.Data)
	return nil
}
