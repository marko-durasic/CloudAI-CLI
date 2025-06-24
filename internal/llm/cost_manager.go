package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ModelCost represents the cost structure for different AWS models
type ModelCost struct {
	ModelID         string  `json:"model_id"`
	InputTokenCost  float64 `json:"input_token_cost"`  // Cost per 1000 input tokens
	OutputTokenCost float64 `json:"output_token_cost"` // Cost per 1000 output tokens
	Speed           int     `json:"speed"`             // Relative speed score (1-10)
	Quality         int     `json:"quality"`           // Relative quality score (1-10)
}

// CostTracker tracks daily usage and costs
type CostTracker struct {
	Date         string  `json:"date"`
	TotalCost    float64 `json:"total_cost"`
	RequestCount int     `json:"request_count"`
	TokensUsed   int     `json:"tokens_used"`
}

// CostManager manages cost tracking and limits
type CostManager struct {
	DailyLimit   float64     `json:"daily_limit"`
	CurrentUsage CostTracker `json:"current_usage"`
	configPath   string
}

// AWS Model costs (as of 2024 - approximate)
var ModelCosts = []ModelCost{
	{
		ModelID:         "anthropic.claude-3-haiku-20240307-v1:0",
		InputTokenCost:  0.25, // $0.25 per 1K tokens
		OutputTokenCost: 1.25, // $1.25 per 1K tokens
		Speed:           9,    // Very fast
		Quality:         7,    // Good quality
	},
	{
		ModelID:         "anthropic.claude-3-sonnet-20240229-v1:0",
		InputTokenCost:  3.0,  // $3.00 per 1K tokens
		OutputTokenCost: 15.0, // $15.00 per 1K tokens
		Speed:           7,    // Medium speed
		Quality:         9,    // Excellent quality
	},
	{
		ModelID:         "amazon.titan-text-express-v1",
		InputTokenCost:  0.13, // $0.13 per 1K tokens
		OutputTokenCost: 0.17, // $0.17 per 1K tokens
		Speed:           8,    // Fast
		Quality:         6,    // Decent quality
	},
	{
		ModelID:         "meta.llama3.2-70b-instruct-v1:0",
		InputTokenCost:  0.99, // $0.99 per 1K tokens
		OutputTokenCost: 0.99, // $0.99 per 1K tokens
		Speed:           6,    // Slower
		Quality:         8,    // High quality
	},
}

// NewCostManager creates a new cost manager
func NewCostManager(dailyLimit float64) *CostManager {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".cloudai-cost.json")

	cm := &CostManager{
		DailyLimit: dailyLimit,
		configPath: configPath,
	}

	cm.LoadUsage()
	return cm
}

// LoadUsage loads current usage from disk
func (cm *CostManager) LoadUsage() {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		// Initialize with today's date
		cm.CurrentUsage = CostTracker{
			Date:         time.Now().Format("2006-01-02"),
			TotalCost:    0,
			RequestCount: 0,
			TokensUsed:   0,
		}
		return
	}

	var usage CostTracker
	if err := json.Unmarshal(data, &usage); err != nil {
		cm.CurrentUsage = CostTracker{
			Date:         time.Now().Format("2006-01-02"),
			TotalCost:    0,
			RequestCount: 0,
			TokensUsed:   0,
		}
		return
	}

	// Reset if it's a new day
	today := time.Now().Format("2006-01-02")
	if usage.Date != today {
		cm.CurrentUsage = CostTracker{
			Date:         today,
			TotalCost:    0,
			RequestCount: 0,
			TokensUsed:   0,
		}
	} else {
		cm.CurrentUsage = usage
	}
}

// SaveUsage saves current usage to disk
func (cm *CostManager) SaveUsage() error {
	data, err := json.MarshalIndent(cm.CurrentUsage, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.configPath, data, 0644)
}

// CanMakeRequest checks if a request can be made within budget
func (cm *CostManager) CanMakeRequest(estimatedCost float64) bool {
	return cm.CurrentUsage.TotalCost+estimatedCost <= cm.DailyLimit
}

// TrackUsage records usage after a request
func (cm *CostManager) TrackUsage(inputTokens, outputTokens int, modelID string) error {
	cost := cm.CalculateCost(inputTokens, outputTokens, modelID)

	cm.CurrentUsage.TotalCost += cost
	cm.CurrentUsage.RequestCount++
	cm.CurrentUsage.TokensUsed += inputTokens + outputTokens

	return cm.SaveUsage()
}

// CalculateCost calculates the cost for a request
func (cm *CostManager) CalculateCost(inputTokens, outputTokens int, modelID string) float64 {
	for _, model := range ModelCosts {
		if model.ModelID == modelID {
			inputCost := float64(inputTokens) / 1000.0 * model.InputTokenCost
			outputCost := float64(outputTokens) / 1000.0 * model.OutputTokenCost
			return inputCost + outputCost
		}
	}
	return 0.0 // Unknown model
}

// GetRemainingBudget returns the remaining daily budget
func (cm *CostManager) GetRemainingBudget() float64 {
	return cm.DailyLimit - cm.CurrentUsage.TotalCost
}

// GetUsageStats returns current usage statistics
func (cm *CostManager) GetUsageStats() CostTracker {
	return cm.CurrentUsage
}

// SelectBestAWSModel selects the best AWS model based on budget and preferences
func SelectBestAWSModel(dailyBudget float64, prioritizeSpeed bool) ModelCost {
	// Filter models that fit within a reasonable per-request budget
	// Assume average request uses ~1000 input + 500 output tokens
	avgInputTokens := 1000.0
	avgOutputTokens := 500.0
	maxCostPerRequest := dailyBudget / 10.0 // Allow up to 10 requests per day

	var affordableModels []ModelCost
	for _, model := range ModelCosts {
		estimatedCost := (avgInputTokens/1000.0)*model.InputTokenCost + (avgOutputTokens/1000.0)*model.OutputTokenCost
		if estimatedCost <= maxCostPerRequest {
			affordableModels = append(affordableModels, model)
		}
	}

	if len(affordableModels) == 0 {
		// Return Claude Haiku as it's most commonly available and affordable
		for _, model := range ModelCosts {
			if model.ModelID == "anthropic.claude-3-haiku-20240307-v1:0" {
				return model
			}
		}
		// Fallback to cheapest model if Haiku not found
		cheapest := ModelCosts[0]
		for _, model := range ModelCosts {
			if model.InputTokenCost+model.OutputTokenCost < cheapest.InputTokenCost+cheapest.OutputTokenCost {
				cheapest = model
			}
		}
		return cheapest
	}

	// Prioritize Claude Haiku if it's affordable (most commonly available)
	for _, model := range affordableModels {
		if model.ModelID == "anthropic.claude-3-haiku-20240307-v1:0" {
			return model
		}
	}

	// Select best model based on priority
	best := affordableModels[0]
	for _, model := range affordableModels {
		if prioritizeSpeed {
			// Prioritize speed over quality
			if model.Speed > best.Speed || (model.Speed == best.Speed && model.Quality > best.Quality) {
				best = model
			}
		} else {
			// Prioritize quality over speed
			if model.Quality > best.Quality || (model.Quality == best.Quality && model.Speed > best.Speed) {
				best = model
			}
		}
	}

	return best
}

// GetModelCost returns cost information for a model
func GetModelCost(modelID string) *ModelCost {
	for _, model := range ModelCosts {
		if model.ModelID == modelID {
			return &model
		}
	}
	return nil
}
