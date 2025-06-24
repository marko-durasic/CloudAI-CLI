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

	fmt.Printf("✅ Query: %s\n", result.Query)

	// Special handling for scan results
	if result.Query == "scan ." || result.Query == "scan" {
		f.formatScanSummary(result.Data)
	} else {
		// For other queries, show a summary of the data
		fmt.Printf("📊 Data: %+v\n", result.Data)
	}
	return nil
}

// formatScanSummary creates a user-friendly summary of scan results
func (f *Formatter) formatScanSummary(data interface{}) {
	if infraData, ok := data.(map[string]interface{}); ok {
		fmt.Println("📋 Infrastructure Summary:")

		// Extract and display key resources
		if resources, ok := infraData["Resources"].(map[string]interface{}); ok {
			resourceCount := len(resources)
			fmt.Printf("   • Total Resources: %d\n", resourceCount)

			// Count by resource type
			resourceTypes := make(map[string]int)
			for _, resource := range resources {
				if resourceMap, ok := resource.(map[string]interface{}); ok {
					if resourceType, ok := resourceMap["Type"].(string); ok {
						resourceTypes[resourceType]++
					}
				}
			}

			// Display resource types
			for resourceType, count := range resourceTypes {
				fmt.Printf("   • %s: %d\n", resourceType, count)
			}

			// Show some key resources
			fmt.Println("\n🔍 Key Resources Found:")
			for resourceName, resource := range resources {
				if resourceMap, ok := resource.(map[string]interface{}); ok {
					if resourceType, ok := resourceMap["Type"].(string); ok {
						// Show user-friendly names for common resources
						switch resourceType {
						case "AWS::Lambda::Function":
							// Try to get the actual function name
							if properties, ok := resourceMap["Properties"].(map[string]interface{}); ok {
								if functionName, ok := properties["FunctionName"].(string); ok {
									fmt.Printf("   • Lambda: %s (%s)\n", functionName, resourceName)
								} else {
									fmt.Printf("   • Lambda: %s\n", resourceName)
								}
							} else {
								fmt.Printf("   • Lambda: %s\n", resourceName)
							}
						case "AWS::ApiGateway::RestApi":
							// Try to get the actual API name
							if properties, ok := resourceMap["Properties"].(map[string]interface{}); ok {
								if apiName, ok := properties["Name"].(string); ok {
									fmt.Printf("   • API Gateway: %s (%s)\n", apiName, resourceName)
								} else {
									fmt.Printf("   • API Gateway: %s\n", resourceName)
								}
							} else {
								fmt.Printf("   • API Gateway: %s\n", resourceName)
							}
						case "AWS::S3::Bucket":
							// Try to get the actual bucket name
							if properties, ok := resourceMap["Properties"].(map[string]interface{}); ok {
								if bucketName, ok := properties["BucketName"].(string); ok {
									fmt.Printf("   • S3 Bucket: %s (%s)\n", bucketName, resourceName)
								} else {
									fmt.Printf("   • S3 Bucket: %s\n", resourceName)
								}
							} else {
								fmt.Printf("   • S3 Bucket: %s\n", resourceName)
							}
						case "AWS::DynamoDB::Table":
							// Try to get the actual table name
							if properties, ok := resourceMap["Properties"].(map[string]interface{}); ok {
								if tableName, ok := properties["TableName"].(string); ok {
									fmt.Printf("   • DynamoDB Table: %s (%s)\n", tableName, resourceName)
								} else {
									fmt.Printf("   • DynamoDB Table: %s\n", resourceName)
								}
							} else {
								fmt.Printf("   • DynamoDB Table: %s\n", resourceName)
							}
						}
					}
				}
			}
		}

		// Show outputs if available
		if outputs, ok := infraData["Outputs"].(map[string]interface{}); ok && len(outputs) > 0 {
			fmt.Printf("\n📤 Outputs: %d\n", len(outputs))
			for outputName := range outputs {
				fmt.Printf("   • %s\n", outputName)
			}
		}

		fmt.Println("\n💡 You can now ask questions about your infrastructure!")
		fmt.Println("   Example: cloudai \"Which Lambda handles GET /hello?\"")
	} else {
		fmt.Printf("📊 Data: %+v\n", data)
	}
}
