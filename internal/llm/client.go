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
)

// Query represents a parsed query with intent and parameters
type Query struct {
	Intent   string            `json:"intent"`
	Service  string            `json:"service"`
	Action   string            `json:"action"`
	Params   map[string]string `json:"params"`
	RawQuery string            `json:"raw_query"`
}

// Client supports both local (Ollama) and remote (OpenAI) LLMs
type Client struct {
	useOllama   bool
	ollamaModel string
	ollamaURL   string
	openai      *openai.Client
}

// NewClient creates a new LLM client, preferring Ollama if available
func NewClient() (*Client, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama3.2:3b"
	}

	// Check if Ollama is running
	if isOllamaAvailable(ollamaURL) {
		return &Client{
			useOllama:   true,
			ollamaModel: ollamaModel,
			ollamaURL:   ollamaURL,
		}, nil
	}

	// Fallback to OpenAI
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("No local Ollama detected and OPENAI_API_KEY not set")
	}
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
	if c.useOllama {
		return c.parseWithOllama(ctx, prompt, rawQuery)
	}
	return c.parseWithOpenAI(ctx, prompt, rawQuery)
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

	if c.useOllama {
		return c.answerWithOllama(ctx, prompt)
	}
	return c.answerWithOpenAI(ctx, prompt)
}

// buildRAGPrompt creates a prompt for Retrieval-Augmented Generation.
func buildRAGPrompt(question, context string) string {
	// Basic prompt, can be improved significantly.
	return fmt.Sprintf(`You are an expert AWS cloud architect.
Your task is to answer a user's question based *only* on the provided infrastructure context.
Do not use any prior knowledge. If the answer is not in the context, say "I cannot answer this based on the provided information."

--- CONTEXT ---
%s
--- END CONTEXT ---

QUESTION: %s
`, context, question)
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
