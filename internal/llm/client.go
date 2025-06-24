package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

// Query represents a parsed query with intent and parameters
type Query struct {
	Intent   string            `json:"intent"`
	Service  string            `json:"service"`
	Action   string            `json:"action"`
	Params   map[string]string `json:"params"`
	RawQuery string            `json:"raw_query"`
}

// Client supports local (Ollama), remote (OpenAI), and AWS-hosted models
type Client struct {
	useOllama   bool
	useAWS      bool
	ollamaModel string
	ollamaURL   string
	openai      *openai.Client
	awsClient   *AWSClient
	costManager *CostManager
}

// NewClient creates a new LLM client, preferring config file settings, then env vars, then auto-detection
func NewClient() (*Client, error) {
	// Check configuration file first
	if modelType := getConfigString("model.type"); modelType != "" {
		switch modelType {
		case "aws":
			return newAWSClientFromConfig()
		case "ollama":
			return newOllamaClientFromConfig()
		}
	}

	// Fallback to environment variables and auto-detection
	return newClientFromEnvAndAutoDetect()
}

// newAWSClientFromConfig creates AWS client from configuration
func newAWSClientFromConfig() (*Client, error) {
	awsConfig := &AWSModelConfig{
		Type:        AWSModelType(getConfigString("model.aws_type")),
		ModelID:     getConfigString("model.model_id"),
		Region:      getConfigString("model.region"),
		MaxTokens:   4096,
		Temperature: 0.1,
	}

	awsClient, err := NewAWSClient(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client from config: %w", err)
	}

	// Initialize cost manager
	dailyLimit := getConfigFloat("cost.daily_limit")
	if dailyLimit == 0 {
		dailyLimit = 5.0 // Default $5/day
	}
	costManager := NewCostManager(dailyLimit)

	fmt.Fprintf(os.Stderr, "🚀 Using AWS model from config: %s (%s)\n", awsConfig.ModelID, awsConfig.Type)
	fmt.Fprintf(os.Stderr, "💰 Daily budget: $%.2f (remaining: $%.2f)\n",
		dailyLimit, costManager.GetRemainingBudget())

	return &Client{
		useAWS:      true,
		awsClient:   awsClient,
		costManager: costManager,
	}, nil
}

// newOllamaClientFromConfig creates Ollama client from configuration
func newOllamaClientFromConfig() (*Client, error) {
	ollamaURL := getConfigString("model.url")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	ollamaModel := getConfigString("model.name")
	if ollamaModel == "" {
		return nil, fmt.Errorf("no Ollama model specified in config")
	}

	if !isOllamaAvailable(ollamaURL) {
		return nil, fmt.Errorf("Ollama is not available at %s", ollamaURL)
	}

	fmt.Fprintf(os.Stderr, "🖥️  Using local Ollama model from config: %s\n", ollamaModel)
	return &Client{
		useOllama:   true,
		ollamaModel: ollamaModel,
		ollamaURL:   ollamaURL,
	}, nil
}

// newClientFromEnvAndAutoDetect creates client from environment variables and auto-detection
func newClientFromEnvAndAutoDetect() (*Client, error) {
	// First, check if AWS model is configured via environment
	if awsConfig := LoadAWSModelFromConfig(); awsConfig != nil {
		awsClient, err := NewAWSClient(awsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
		}

		// Use default daily limit for env-configured AWS models
		costManager := NewCostManager(5.0) // $5/day default

		fmt.Fprintf(os.Stderr, "🚀 Using AWS model: %s (%s)\n", awsConfig.ModelID, awsConfig.Type)
		return &Client{
			useAWS:      true,
			awsClient:   awsClient,
			costManager: costManager,
		}, nil
	}

	// Check for Ollama
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaModel := os.Getenv("OLLAMA_MODEL")

	// Check if Ollama is running
	if isOllamaAvailable(ollamaURL) {
		// If no model is specified, try to load from config or auto-select
		if ollamaModel == "" {
			ollamaModel = loadModelFromConfig()
			if ollamaModel == "" {
				var err error
				ollamaModel, err = SelectBestModel(ollamaURL)
				if err != nil {
					return nil, fmt.Errorf("failed to auto-select model: %w", err)
				}
				// Save the selected model to config for future use
				saveModelToConfig(ollamaModel)
			}
		}

		fmt.Fprintf(os.Stderr, "🖥️  Using local Ollama model: %s\n", ollamaModel)
		return &Client{
			useOllama:   true,
			ollamaModel: ollamaModel,
			ollamaURL:   ollamaURL,
		}, nil
	}

	// Fallback to OpenAI
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("No model configured. Please run 'cloudai setup-interactive' to configure your AI model")
	}

	fmt.Fprintf(os.Stderr, "☁️  Using OpenAI model\n")
	return &Client{
		useOllama: false,
		openai:    openai.NewClient(apiKey),
	}, nil
}

// isOllamaAvailable checks if Ollama API is reachable
func isOllamaAvailable(url string) bool {
	resp, err := http.Get(url + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// ParseQuery uses LLM to parse natural language into structured query
func (c *Client) ParseQuery(ctx context.Context, rawQuery string) (*Query, error) {
	prompt := buildPrompt(rawQuery)

	if c.useAWS {
		return c.parseWithAWS(ctx, prompt, rawQuery)
	} else if c.useOllama {
		return c.parseWithOllama(ctx, prompt, rawQuery)
	} else {
		return c.parseWithOpenAI(ctx, prompt, rawQuery)
	}
}

// buildPrompt creates a system prompt for intent extraction
func buildPrompt(raw string) string {
	return `You are an AWS CLI assistant. Parse the following user query into a JSON object with fields: intent, service, action, params (map), and raw_query.

Common intents:
- "api_gateway_lambda" for queries about which Lambda handles API Gateway requests
- "lambda_triggers" for queries about what triggers a Lambda function
- "cost_top" for queries about top cost services

Examples:
Query: "Which Lambda handles GET /users on prod-api?"
Response: {"intent": "api_gateway_lambda", "service": "apigateway", "action": "get_integration", "params": {"api": "prod-api", "method": "GET", "path": "/users"}, "raw_query": "Which Lambda handles GET /users on prod-api?"}

Query: "What triggers the process-order Lambda?"
Response: {"intent": "lambda_triggers", "service": "lambda", "action": "list_triggers", "params": {"lambda": "process-order"}, "raw_query": "What triggers the process-order Lambda?"}

Query: "Top 3 services by cost last 7 days"
Response: {"intent": "cost_top", "service": "costexplorer", "action": "get_cost", "params": {"limit": "3", "period": "7 days"}, "raw_query": "Top 3 services by cost last 7 days"}

Now parse this query: ` + raw
}

// parseWithAWS sends the prompt to the AWS model
func (c *Client) parseWithAWS(ctx context.Context, prompt, rawQuery string) (*Query, error) {
	response, err := c.awsClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("aws model request failed: %w", err)
	}

	// Try to parse JSON response
	var q Query
	if err := json.Unmarshal([]byte(response), &q); err == nil {
		q.RawQuery = rawQuery
		return &q, nil
	}

	// Fallback to unknown intent
	return &Query{Intent: "unknown", RawQuery: rawQuery, Params: map[string]string{}}, nil
}

// parseWithOllama sends the prompt to the local Ollama model
func (c *Client) parseWithOllama(ctx context.Context, prompt, rawQuery string) (*Query, error) {
	body := map[string]interface{}{
		"model":  c.ollamaModel,
		"prompt": prompt,
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(c.ollamaURL+"/api/generate", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Response string `json:"response"`
	}
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		if err := dec.Decode(&result); err == nil && strings.Contains(result.Response, "intent") {
			var q Query
			if err := json.Unmarshal([]byte(result.Response), &q); err == nil {
				q.RawQuery = rawQuery
				return &q, nil
			}
		}
	}
	return &Query{Intent: "unknown", RawQuery: rawQuery, Params: map[string]string{}}, nil
}

// parseWithOpenAI sends the prompt to OpenAI
func (c *Client) parseWithOpenAI(ctx context.Context, prompt, rawQuery string) (*Query, error) {
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{{Role: "system", Content: prompt}},
	}
	resp, err := c.openai.CreateChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return &Query{Intent: "unknown", RawQuery: rawQuery, Params: map[string]string{}}, nil
	}
	var q Query
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &q); err == nil {
		q.RawQuery = rawQuery
		return &q, nil
	}
	return &Query{Intent: "unknown", RawQuery: rawQuery, Params: map[string]string{}}, nil
}

// Answer uses the LLM to answer a question based on provided context.
func (c *Client) Answer(ctx context.Context, question, context string) (string, error) {
	prompt := buildRAGPrompt(question, context)

	var response string
	var err error

	if c.useAWS {
		// Check budget before making request
		if c.costManager != nil {
			estimatedCost := c.estimateRequestCost(prompt)
			if !c.costManager.CanMakeRequest(estimatedCost) {
				remaining := c.costManager.GetRemainingBudget()
				return "", fmt.Errorf("daily budget exceeded. Remaining: $%.2f, Estimated cost: $%.2f", remaining, estimatedCost)
			}
		}

		response, err = c.awsClient.Generate(ctx, prompt)

		// Track actual usage after successful request
		if err == nil && c.costManager != nil {
			// Estimate token usage (rough approximation)
			inputTokens := len(prompt) / 4 // ~4 chars per token
			outputTokens := len(response) / 4
			c.costManager.TrackUsage(inputTokens, outputTokens, c.awsClient.config.ModelID)
		}
	} else if c.useOllama {
		response, err = c.answerWithOllama(ctx, prompt)
	} else {
		response, err = c.answerWithOpenAI(ctx, prompt)
	}

	if err != nil {
		return "", err
	}

	// Post-process the response to make it more user-friendly
	cleanedResponse := cleanAIResponse(response, context)
	return cleanedResponse, nil
}

// estimateRequestCost estimates the cost of a request
func (c *Client) estimateRequestCost(prompt string) float64 {
	if c.awsClient == nil {
		return 0.0
	}

	// Rough estimation: 4 characters per token
	inputTokens := len(prompt) / 4
	outputTokens := 500 // Assume average output length

	modelCost := GetModelCost(c.awsClient.config.ModelID)
	if modelCost == nil {
		return 0.01 // Default small cost
	}

	inputCost := float64(inputTokens) / 1000.0 * modelCost.InputTokenCost
	outputCost := float64(outputTokens) / 1000.0 * modelCost.OutputTokenCost
	return inputCost + outputCost
}

// buildRAGPrompt creates a prompt for Retrieval-Augmented Generation.
func buildRAGPrompt(question, context string) string {
	// Truly non-deterministic, cloud-agnostic prompt
	return fmt.Sprintf(`You are an expert cloud infrastructure assistant.
Your task is to answer a user's question about their infrastructure based *only* on the provided context.

IMPORTANT GUIDELINES:
1. Always use the most human-friendly, descriptive property available for each resource.
2. Look for properties that appear to be names, IDs, or descriptions (such as "Name", "ID", "Description", etc.), but do not limit yourself to these—use your best judgment based on the data provided.
3. If no such property exists, use the most descriptive identifier available.
4. Never rely on internal logical IDs unless there is no better option.
5. Be specific and actionable in your responses.
6. If you can't find the answer in the context, say "I cannot answer this based on the provided infrastructure information."
7. Keep responses concise but informative—aim for 1-3 sentences.
8. Use bullet points or numbered lists when appropriate for clarity.
9. Focus on answering the user's question directly—don't over-explain technical details unless specifically asked.
10. Avoid listing all available resources unless the question specifically asks for them.

RESPONSE STYLE:
- Be direct and to the point
- Use simple, clear language
- Focus on what the user asked, not every detail in the infrastructure
- If the answer is simple, keep it simple
- ALWAYS use friendly resource names or descriptions instead of logical IDs

--- INFRASTRUCTURE CONTEXT ---
%s
--- END CONTEXT ---

QUESTION: %s

Please provide a clear, concise answer using the most human-friendly resource names or descriptions:`, context, question)
}

func (c *Client) answerWithOllama(ctx context.Context, prompt string) (string, error) {
	body := map[string]interface{}{
		"model":  c.ollamaModel,
		"prompt": prompt,
		"stream": false, // We want the full answer at once
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(c.ollamaURL+"/api/generate", "application/json", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Response, nil
}

func (c *Client) answerWithOpenAI(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4o,
		Messages: []openai.ChatCompletionMessage{{Role: "system", Content: prompt}},
	}
	resp, err := c.openai.CreateChatCompletion(ctx, req)
	if err != nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai request failed or returned no choices: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

// loadModelFromConfig loads the selected model from config file
func loadModelFromConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	configPath := home + "/.cloudai.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	// Simple YAML parsing for the model field
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "model:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

// saveModelToConfig saves the selected model to config file
func saveModelToConfig(model string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	configPath := home + "/.cloudai.yaml"
	configDir := home + "/.cloudai"

	// Create config directory if it doesn't exist
	os.MkdirAll(configDir, 0755)

	// Read existing config or create new
	var configData string
	if data, err := os.ReadFile(configPath); err == nil {
		configData = string(data)
	}

	// Update or add model setting
	lines := strings.Split(configData, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "model:") {
			lines[i] = "model: " + model
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, "model: "+model)
	}

	// Write back to file
	os.WriteFile(configPath, []byte(strings.Join(lines, "\n")), 0644)
}

// cleanAIResponse post-processes the AI response to make it more user-friendly
func cleanAIResponse(response string, context string) string {
	// Remove common verbose patterns
	response = strings.TrimSpace(response)

	// Remove overly technical explanations
	patternsToRemove := []string{
		"Based on the provided infrastructure context,",
		"From the context, we know that",
		"If you have any further questions or if there's anything else I can help you with, please let me know!",
		"To find out which",
		"Now, let's analyze",
		"So, in summary:",
		"However, since",
		"It's reasonable to conclude that",
	}

	for _, pattern := range patternsToRemove {
		response = strings.ReplaceAll(response, pattern, "")
	}

	// Clean up multiple newlines
	response = strings.ReplaceAll(response, "\n\n\n", "\n\n")

	// Remove bullet points that are just technical details
	lines := strings.Split(response, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip overly technical lines
		if strings.Contains(line, "DemoApi") && strings.Contains(line, "Permission") {
			continue
		}
		if strings.Contains(line, "SourceArn") {
			continue
		}
		if strings.Contains(line, "Action:") && strings.Contains(line, "lambda:InvokeFunction") {
			continue
		}

		cleanedLines = append(cleanedLines, line)
	}

	response = strings.Join(cleanedLines, "\n")

	// If the response is still too long, try to extract the key answer
	if len(response) > 500 {
		// Look for the main answer in the first few sentences
		sentences := strings.Split(response, ".")
		if len(sentences) > 0 {
			// Take the first 2-3 sentences that seem to contain the answer
			var keySentences []string
			for i, sentence := range sentences {
				if i >= 3 {
					break
				}
				sentence = strings.TrimSpace(sentence)
				if sentence != "" && !strings.Contains(sentence, "Based on") && !strings.Contains(sentence, "From the context") {
					keySentences = append(keySentences, sentence)
				}
			}
			if len(keySentences) > 0 {
				response = strings.Join(keySentences, ". ") + "."
			}
		}
	}

	return strings.TrimSpace(response)
}

// Helper functions for configuration
func getConfigString(key string) string {
	return viper.GetString(key)
}

func getConfigFloat(key string) float64 {
	return viper.GetFloat64(key)
}
