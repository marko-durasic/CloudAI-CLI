package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/sagemakerruntime"
)

// AWSModelType represents different types of AWS-hosted models
type AWSModelType string

const (
	AWSModelBedrock   AWSModelType = "bedrock"
	AWSModelSageMaker AWSModelType = "sagemaker"
	AWSModelOpenAI    AWSModelType = "openai"
)

// AWSModelConfig holds configuration for AWS models
type AWSModelConfig struct {
	Type         AWSModelType `json:"type"`
	ModelID      string       `json:"model_id"`
	EndpointName string       `json:"endpoint_name,omitempty"` // For SageMaker
	Region       string       `json:"region"`
	MaxTokens    int          `json:"max_tokens"`
	Temperature  float64      `json:"temperature"`
}

// AWSClient handles AWS-hosted model interactions
type AWSClient struct {
	config          *AWSModelConfig
	bedrockClient   *bedrockruntime.Client
	sagemakerClient *sagemakerruntime.Client
	region          string
}

// NewAWSClient creates a new AWS model client
func NewAWSClient(modelConfig *AWSModelConfig) (*AWSClient, error) {
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(modelConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := &AWSClient{
		config: modelConfig,
		region: modelConfig.Region,
	}

	// Initialize appropriate client based on model type
	switch modelConfig.Type {
	case AWSModelBedrock:
		client.bedrockClient = bedrockruntime.NewFromConfig(cfg)
	case AWSModelSageMaker:
		client.sagemakerClient = sagemakerruntime.NewFromConfig(cfg)
	case AWSModelOpenAI:
		// OpenAI through AWS (if configured)
		client.bedrockClient = bedrockruntime.NewFromConfig(cfg)
	default:
		return nil, fmt.Errorf("unsupported AWS model type: %s", modelConfig.Type)
	}

	return client, nil
}

// Generate sends a prompt to the AWS model and returns the response
func (c *AWSClient) Generate(ctx context.Context, prompt string) (string, error) {
	switch c.config.Type {
	case AWSModelBedrock:
		return c.generateWithBedrock(ctx, prompt)
	case AWSModelSageMaker:
		return c.generateWithSageMaker(ctx, prompt)
	case AWSModelOpenAI:
		return c.generateWithBedrockOpenAI(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported model type: %s", c.config.Type)
	}
}

// generateWithBedrock sends request to AWS Bedrock
func (c *AWSClient) generateWithBedrock(ctx context.Context, prompt string) (string, error) {
	// Prepare the request body based on the model
	var body []byte
	var err error

	switch {
	case strings.Contains(c.config.ModelID, "anthropic"):
		body, err = json.Marshal(map[string]interface{}{
			"prompt":            prompt,
			"max_tokens":        c.config.MaxTokens,
			"temperature":       c.config.Temperature,
			"top_p":             1.0,
			"anthropic_version": "bedrock-2023-05-31",
		})
	case strings.Contains(c.config.ModelID, "amazon.titan"):
		body, err = json.Marshal(map[string]interface{}{
			"inputText": prompt,
			"textGenerationConfig": map[string]interface{}{
				"maxTokenCount": c.config.MaxTokens,
				"temperature":   c.config.Temperature,
				"topP":          1.0,
			},
		})
	case strings.Contains(c.config.ModelID, "meta.llama"):
		body, err = json.Marshal(map[string]interface{}{
			"prompt":      prompt,
			"max_gen_len": c.config.MaxTokens,
			"temperature": c.config.Temperature,
			"top_p":       1.0,
		})
	default:
		return "", fmt.Errorf("unsupported Bedrock model: %s", c.config.ModelID)
	}

	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request to Bedrock
	resp, err := c.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.config.ModelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return "", fmt.Errorf("bedrock request failed: %w", err)
	}

	// Parse response based on model type
	var responseText string
	switch {
	case strings.Contains(c.config.ModelID, "anthropic"):
		var result struct {
			Completion string `json:"completion"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return "", fmt.Errorf("failed to parse anthropic response: %w", err)
		}
		responseText = result.Completion
	case strings.Contains(c.config.ModelID, "amazon.titan"):
		var result struct {
			Results []struct {
				OutputText string `json:"outputText"`
			} `json:"results"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return "", fmt.Errorf("failed to parse titan response: %w", err)
		}
		if len(result.Results) > 0 {
			responseText = result.Results[0].OutputText
		}
	case strings.Contains(c.config.ModelID, "meta.llama"):
		var result struct {
			Generation string `json:"generation"`
		}
		if err := json.Unmarshal(resp.Body, &result); err != nil {
			return "", fmt.Errorf("failed to parse llama response: %w", err)
		}
		responseText = result.Generation
	}

	return strings.TrimSpace(responseText), nil
}

// generateWithSageMaker sends request to SageMaker endpoint
func (c *AWSClient) generateWithSageMaker(ctx context.Context, prompt string) (string, error) {
	// Prepare the request body (assuming a standard format)
	body := map[string]interface{}{
		"prompt":      prompt,
		"max_tokens":  c.config.MaxTokens,
		"temperature": c.config.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Send request to SageMaker endpoint
	resp, err := c.sagemakerClient.InvokeEndpoint(ctx, &sagemakerruntime.InvokeEndpointInput{
		EndpointName: aws.String(c.config.EndpointName),
		ContentType:  aws.String("application/json"),
		Body:         bodyBytes,
	})
	if err != nil {
		return "", fmt.Errorf("sagemaker request failed: %w", err)
	}

	// Parse response (assuming standard format)
	var result struct {
		Response string `json:"response"`
		Output   string `json:"output"`
		Text     string `json:"text"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", fmt.Errorf("failed to parse sagemaker response: %w", err)
	}

	// Try different response fields
	responseText := result.Response
	if responseText == "" {
		responseText = result.Output
	}
	if responseText == "" {
		responseText = result.Text
	}

	return strings.TrimSpace(responseText), nil
}

// generateWithBedrockOpenAI sends request to OpenAI through AWS Bedrock
func (c *AWSClient) generateWithBedrockOpenAI(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  c.config.MaxTokens,
		"temperature": c.config.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(c.config.ModelID),
		ContentType: aws.String("application/json"),
		Body:        bodyBytes,
	})
	if err != nil {
		return "", fmt.Errorf("bedrock openai request failed: %w", err)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", fmt.Errorf("failed to parse openai response: %w", err)
	}

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response from model")
}

// GetAvailableAWSModels returns a list of available AWS models
func GetAvailableAWSModels() []AWSModelConfig {
	return []AWSModelConfig{
		{
			Type:        AWSModelBedrock,
			ModelID:     "anthropic.claude-3-sonnet-20240229-v1:0",
			Region:      "us-east-1",
			MaxTokens:   4096,
			Temperature: 0.1,
		},
		{
			Type:        AWSModelBedrock,
			ModelID:     "anthropic.claude-3-haiku-20240307-v1:0",
			Region:      "us-east-1",
			MaxTokens:   4096,
			Temperature: 0.1,
		},
		{
			Type:        AWSModelBedrock,
			ModelID:     "amazon.titan-text-express-v1",
			Region:      "us-east-1",
			MaxTokens:   4096,
			Temperature: 0.1,
		},
		{
			Type:        AWSModelBedrock,
			ModelID:     "meta.llama3.2-70b-instruct-v1:0",
			Region:      "us-east-1",
			MaxTokens:   4096,
			Temperature: 0.1,
		},
		{
			Type:        AWSModelBedrock,
			ModelID:     "openai.gpt-4o",
			Region:      "us-east-1",
			MaxTokens:   4096,
			Temperature: 0.1,
		},
	}
}

// LoadAWSModelFromConfig loads AWS model configuration from environment or config file
func LoadAWSModelFromConfig() *AWSModelConfig {
	// Check environment variables first
	if modelType := os.Getenv("AWS_MODEL_TYPE"); modelType != "" {
		config := &AWSModelConfig{
			Type:         AWSModelType(modelType),
			ModelID:      os.Getenv("AWS_MODEL_ID"),
			EndpointName: os.Getenv("AWS_ENDPOINT_NAME"),
			Region:       os.Getenv("AWS_REGION"),
			MaxTokens:    4096,
			Temperature:  0.1,
		}

		// Set defaults
		if config.Region == "" {
			config.Region = "us-east-1"
		}

		return config
	}

	// TODO: Load from config file if no env vars
	return nil
}
