package llm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/ddjura/cloudai/internal/sysinfo"
)

// ModelInfo represents information about a model
type ModelInfo struct {
	Name     string `json:"name"`
	MinRAMGB int    `json:"min_ram_gb"`
	MinCPUs  int    `json:"min_cpus"`
	NeedsGPU bool   `json:"needs_gpu"`
	Size     string `json:"size"`
	Priority int    `json:"priority"` // Higher number = better model
}

// ModelRequirements defines the requirements for different models
var ModelRequirements = []ModelInfo{
	{
		Name:     "llama3.2:3b",
		MinRAMGB: 8,
		MinCPUs:  4,
		NeedsGPU: false,
		Size:     "3B",
		Priority: 100,
	},
	{
		Name:     "llama3.2:1b",
		MinRAMGB: 4,
		MinCPUs:  2,
		NeedsGPU: false,
		Size:     "1B",
		Priority: 80,
	},
	{
		Name:     "phi3:mini",
		MinRAMGB: 4,
		MinCPUs:  2,
		NeedsGPU: false,
		Size:     "Mini",
		Priority: 70,
	},
	{
		Name:     "llama3.2:8b",
		MinRAMGB: 16,
		MinCPUs:  8,
		NeedsGPU: false,
		Size:     "8B",
		Priority: 120,
	},
}

// AvailableModel represents a model available in Ollama
type AvailableModel struct {
	Name    string `json:"name"`
	Model   string `json:"model"`
	Size    int64  `json:"size"`
	Details struct {
		ParameterSize string `json:"parameter_size"`
	} `json:"details"`
}

// SelectBestModel selects the best available model based on system specs
func SelectBestModel(ollamaURL string) (string, error) {
	// Get system specs
	specs, err := sysinfo.DetectSystemSpecs()
	if err != nil {
		return "", fmt.Errorf("failed to detect system specs: %w", err)
	}

	fmt.Fprintf(os.Stderr, "🔍 Detected system: %s\n", specs.String())

	// Get available models from Ollama
	availableModels, err := getAvailableModels(ollamaURL)
	if err != nil {
		return "", fmt.Errorf("failed to get available models: %w", err)
	}

	if len(availableModels) == 0 {
		return "", fmt.Errorf("no models available in Ollama. Please install a model first: ollama pull llama3.2:1b")
	}

	// Find the best model that fits the system and is available
	bestModel := selectBestAvailableModel(specs, availableModels)
	if bestModel == "" {
		return "", fmt.Errorf("no suitable model found for your system specs: %s", specs.String())
	}

	fmt.Fprintf(os.Stderr, "✅ Selected model: %s\n", bestModel)
	return bestModel, nil
}

// getAvailableModels fetches the list of available models from Ollama
func getAvailableModels(ollamaURL string) ([]AvailableModel, error) {
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []AvailableModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return result.Models, nil
}

// selectBestAvailableModel finds the best model that fits the system specs and is available
func selectBestAvailableModel(specs *sysinfo.SystemSpecs, availableModels []AvailableModel) string {
	// Create a map of available models for quick lookup
	availableMap := make(map[string]bool)
	for _, model := range availableModels {
		availableMap[model.Name] = true
	}

	// Sort model requirements by priority (highest first)
	sortedRequirements := make([]ModelInfo, len(ModelRequirements))
	copy(sortedRequirements, ModelRequirements)
	sort.Slice(sortedRequirements, func(i, j int) bool {
		return sortedRequirements[i].Priority > sortedRequirements[j].Priority
	})

	// Find the first model that fits the system and is available
	for _, req := range sortedRequirements {
		if !availableMap[req.Name] {
			continue // Model not available
		}

		if specs.RAMGB < req.MinRAMGB {
			continue // Not enough RAM
		}

		if specs.CPUCores < req.MinCPUs {
			continue // Not enough CPU cores
		}

		if req.NeedsGPU && !specs.HasGPU {
			continue // Needs GPU but none available
		}

		return req.Name
	}

	return ""
}

// GetModelDisplayName returns a user-friendly name for a model
func GetModelDisplayName(modelName string) string {
	for _, req := range ModelRequirements {
		if req.Name == modelName {
			return fmt.Sprintf("%s (%s)", req.Name, req.Size)
		}
	}
	return modelName
}
