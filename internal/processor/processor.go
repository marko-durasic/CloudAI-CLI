package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/ddjura/cloudai/internal/aws"
	"github.com/ddjura/cloudai/internal/llm"
	"github.com/ddjura/cloudai/internal/output"
)

// Processor handles query processing
type Processor struct {
	llmClient *llm.Client
	awsClient *aws.Client
	formatter *output.Formatter
}

// NewProcessor creates a new processor
func NewProcessor(llmClient *llm.Client, awsClient *aws.Client, formatter *output.Formatter) *Processor {
	return &Processor{
		llmClient: llmClient,
		awsClient: awsClient,
		formatter: formatter,
	}
}

// ProcessQuery processes a natural language query
func (p *Processor) ProcessQuery(ctx context.Context, rawQuery string) error {
	// Parse the query using LLM
	query, err := p.llmClient.ParseQuery(ctx, rawQuery)
	if err != nil {
		return fmt.Errorf("failed to parse query: %w", err)
	}

	// Fallback parser if LLM fails to determine intent
	if query.Intent == "unknown" {
		query = p.fallbackParse(rawQuery)
	}

	// Execute the query based on intent
	var data interface{}
	switch query.Intent {
	case "lambda_triggers":
		data, err = p.handleLambdaTriggers(ctx, query)
	case "api_gateway_lambda":
		data, err = p.handleAPIGatewayLambda(ctx, query)
	case "cost_top":
		data, err = p.handleCostTop(ctx, query)
	default:
		data = map[string]string{
			"message": "Query intent not yet implemented",
			"intent":  query.Intent,
		}
	}

	if err != nil {
		result := &output.Result{
			Query:   rawQuery,
			Error:   err.Error(),
			Success: false,
		}
		return p.formatter.FormatResult(result)
	}

	result := &output.Result{
		Query:   rawQuery,
		Data:    data,
		Success: true,
	}

	return p.formatter.FormatResult(result)
}

// handleLambdaTriggers handles Lambda trigger queries
func (p *Processor) handleLambdaTriggers(ctx context.Context, query *llm.Query) (interface{}, error) {
	// TODO: Implement Lambda trigger lookup
	return map[string]string{
		"message": "Lambda trigger lookup not yet implemented",
		"lambda":  query.Params["lambda"],
	}, nil
}

// handleAPIGatewayLambda handles API Gateway to Lambda queries
func (p *Processor) handleAPIGatewayLambda(ctx context.Context, query *llm.Query) (interface{}, error) {
	// Extract parameters from query
	apiName := query.Params["api"]
	httpMethod := query.Params["method"]
	path := query.Params["path"]

	// List all REST APIs
	apis, err := p.awsClient.APIGateway.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list API Gateways: %w", err)
	}

	// Find the target API
	var targetAPI *types.RestApi
	for _, api := range apis.Items {
		if apiName == "" || *api.Name == apiName {
			targetAPI = &api
			break
		}
	}

	if targetAPI == nil {
		// Return available APIs
		apiNames := make([]string, len(apis.Items))
		for i, api := range apis.Items {
			apiNames[i] = *api.Name
		}
		return map[string]interface{}{
			"message":        fmt.Sprintf("API Gateway '%s' not found", apiName),
			"available_apis": apiNames,
		}, nil
	}

	// Get resources for the API
	resources, err := p.awsClient.APIGateway.GetResources(ctx, &apigateway.GetResourcesInput{
		RestApiId: targetAPI.Id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get API resources: %w", err)
	}

	// Find the resource matching the path
	var targetResource *types.Resource
	for _, resource := range resources.Items {
		if resource.ResourceMethods != nil {
			if _, ok := resource.ResourceMethods[httpMethod]; ok && *resource.Path == path {
				targetResource = &resource
				break
			}
		}
	}

	if targetResource == nil {
		return map[string]interface{}{
			"message":  fmt.Sprintf("Path '%s' with method '%s' not found in API '%s'", path, httpMethod, *targetAPI.Name),
			"api_name": *targetAPI.Name,
			"api_id":   *targetAPI.Id,
		}, nil
	}

	// Get the method integration
	method, err := p.awsClient.APIGateway.GetMethod(ctx, &apigateway.GetMethodInput{
		RestApiId:  targetAPI.Id,
		ResourceId: targetResource.Id,
		HttpMethod: awssdk.String(httpMethod),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get method: %w", err)
	}

	// Extract Lambda function name from integration URI
	var lambdaName string
	if method.MethodIntegration != nil && method.MethodIntegration.Uri != nil {
		uri := *method.MethodIntegration.Uri
		if strings.Contains(uri, ":lambda:path") {
			parts := strings.Split(uri, ":function:")
			if len(parts) > 1 {
				lambdaName = strings.Split(parts[1], "/")[0]
			}
		}
	}

	return map[string]interface{}{
		"api_name":    *targetAPI.Name,
		"api_id":      *targetAPI.Id,
		"path":        *targetResource.Path,
		"method":      httpMethod,
		"lambda_name": lambdaName,
	}, nil
}

// handleCostTop handles cost top queries
func (p *Processor) handleCostTop(ctx context.Context, query *llm.Query) (interface{}, error) {
	// TODO: Implement cost top lookup
	return map[string]string{
		"message": "Cost top lookup not yet implemented",
		"period":  query.Params["period"],
		"limit":   query.Params["limit"],
	}, nil
}

// fallbackParse is a simple keyword-based parser
func (p *Processor) fallbackParse(rawQuery string) *llm.Query {
	lowerQuery := strings.ToLower(rawQuery)
	query := &llm.Query{RawQuery: rawQuery, Params: make(map[string]string)}

	// API Gateway -> Lambda intent
	if strings.Contains(lowerQuery, "lambda") && (strings.Contains(lowerQuery, "api") || strings.Contains(lowerQuery, "gateway")) {
		query.Intent = "api_gateway_lambda"
		query.Service = "apigateway"
		query.Action = "get_integration"

		// Regex to extract METHOD /path on api-name
		r := regexp.MustCompile(`(?i)(GET|POST|PUT|DELETE|PATCH)\s+([/\w-]+)\s+(?:on|in)\s+([\w-]+)`)
		matches := r.FindStringSubmatch(rawQuery)
		if len(matches) == 4 {
			query.Params["method"] = strings.ToUpper(matches[1])
			query.Params["path"] = matches[2]
			query.Params["api"] = matches[3]
		}
		return query
	}

	// Default to unknown
	query.Intent = "unknown"
	return query
}
