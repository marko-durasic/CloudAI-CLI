package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/ddjura/cloudai/internal/llm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var interactiveSetupCmd = &cobra.Command{
	Use:   "setup-interactive",
	Short: "Interactive setup for CloudAI-CLI model selection and cost controls",
	Long: `Interactive setup that guides you through:
1. Choosing between local (Ollama) and remote (AWS) models
2. Setting up cost controls for remote models
3. Configuring your preferred model settings

This setup will create a configuration file to remember your preferences.`,
	RunE: runInteractiveSetup,
}

func runInteractiveSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 CloudAI-CLI Interactive Setup")
	fmt.Println("=" + strings.Repeat("=", 40))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Step 1: Choose model type
	fmt.Println("1. Choose your AI model preference:")
	fmt.Println("   [1] Local models (Ollama) - Private, free, slower")
	fmt.Println("   [2] Remote models (AWS) - Fast, paid, requires AWS account")
	fmt.Println()

	for {
		fmt.Print("Choose option (1 or 2): ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			return setupLocalModels(reader)
		case "2":
			return setupRemoteModels(reader)
		default:
			fmt.Println("❌ Please enter 1 or 2")
		}
	}
}

func setupLocalModels(reader *bufio.Reader) error {
	fmt.Println("\n🖥️  Setting up local models (Ollama)...")
	fmt.Println()

	// Check if Ollama is installed
	if !isOllamaAvailable("http://localhost:11434") {
		fmt.Println("❌ Ollama is not running. Please:")
		fmt.Println("   1. Install Ollama: https://ollama.com/")
		fmt.Println("   2. Start Ollama: ollama serve")
		fmt.Println("   3. Install a model: ollama pull llama3.2:3b")
		fmt.Println()
		fmt.Print("Press Enter when Ollama is ready...")
		reader.ReadString('\n')

		if !isOllamaAvailable("http://localhost:11434") {
			return fmt.Errorf("Ollama is still not available")
		}
	}

	fmt.Println("✅ Ollama detected!")

	// Auto-select best model
	bestModel, err := llm.SelectBestModel("http://localhost:11434")
	if err != nil {
		return fmt.Errorf("failed to select best model: %w", err)
	}

	fmt.Printf("✅ Selected model: %s\n", bestModel)

	// Save configuration
	viper.Set("model.type", "ollama")
	viper.Set("model.name", bestModel)
	viper.Set("model.url", "http://localhost:11434")

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n🎉 Local model setup complete!")
	fmt.Println("You can now use CloudAI-CLI with privacy-focused local models.")
	return nil
}

func setupRemoteModels(reader *bufio.Reader) error {
	fmt.Println("\n☁️  Setting up remote models (AWS)...")
	fmt.Println()

	// Check AWS credentials
	fmt.Println("Checking AWS credentials...")
	if err := checkAWSCredentials(); err != nil {
		fmt.Printf("❌ AWS credentials issue: %v\n", err)
		fmt.Println("\n📋 To fix this, set up your AWS credentials:")
		fmt.Println("   1. Install AWS CLI: https://aws.amazon.com/cli/")
		fmt.Println("   2. Run: aws configure")
		fmt.Println("   3. Or set environment variables:")
		fmt.Println("      export AWS_ACCESS_KEY_ID=your_access_key")
		fmt.Println("      export AWS_SECRET_ACCESS_KEY=your_secret_key")
		fmt.Println("      export AWS_DEFAULT_REGION=us-east-1")
		return fmt.Errorf("AWS credentials not configured")
	}
	fmt.Println("✅ AWS credentials found!")

	// Check Bedrock access
	fmt.Println("Checking Bedrock access...")
	if err := checkBedrockAccess(); err != nil {
		fmt.Printf("❌ Bedrock access issue: %v\n", err)
		fmt.Println("\n🔧 To enable Bedrock access:")
		fmt.Println("   1. Go to AWS Console → Amazon Bedrock")
		fmt.Println("   2. Navigate to 'Model access' in the left sidebar")
		fmt.Println("   3. Click 'Enable specific models' or 'Enable all models'")
		fmt.Println("   4. At minimum, enable: Anthropic Claude (recommended)")
		fmt.Println("   5. Submit the request and wait for approval (usually instant)")
		fmt.Println("\n📋 Required IAM permissions:")
		fmt.Println("   - bedrock:InvokeModel")
		fmt.Println("   - bedrock:ListFoundationModels")
		fmt.Println("   - bedrock:GetFoundationModel")
		fmt.Println()
		fmt.Print("Press Enter after enabling Bedrock access to continue...")
		reader.ReadString('\n')

		// Re-check after user action
		if err := checkBedrockAccess(); err != nil {
			return fmt.Errorf("Bedrock access still not available: %w", err)
		}
	}
	fmt.Println("✅ Bedrock access confirmed!")

	// Step 1: Set daily budget
	var dailyBudget float64
	for {
		fmt.Print("Set daily spending limit (USD, e.g., 5.00): $")
		budgetStr, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading budget: %w", err)
		}
		budgetStr = strings.TrimSpace(budgetStr)

		budget, err := strconv.ParseFloat(budgetStr, 64)
		if err != nil || budget <= 0 {
			fmt.Println("❌ Please enter a valid amount (e.g., 5.00)")
			continue
		}

		dailyBudget = budget
		break
	}

	// Step 2: Choose priority
	fmt.Println("\n2. Choose your priority:")
	fmt.Println("   [1] Speed - Faster responses, may cost more")
	fmt.Println("   [2] Cost - Cheaper models, may be slower")
	fmt.Println()

	var prioritizeSpeed bool
	for {
		fmt.Print("Choose priority (1 or 2): ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading priority: %w", err)
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			prioritizeSpeed = true
			break
		case "2":
			prioritizeSpeed = false
			break
		default:
			fmt.Println("❌ Please enter 1 or 2")
			continue
		}
		break
	}

	// Step 3: Select best model based on preferences
	bestModel := llm.SelectBestAWSModel(dailyBudget, prioritizeSpeed)

	// Verify the selected model is actually available
	fmt.Printf("\n🔍 Verifying model access: %s\n", bestModel.ModelID)
	if err := testModelAccess(bestModel.ModelID); err != nil {
		fmt.Printf("❌ Selected model not accessible: %v\n", err)
		fmt.Println("🔄 Falling back to most commonly available model...")

		// Try Claude Haiku as fallback
		fallbackModel := "anthropic.claude-3-haiku-20240307-v1:0"
		if err := testModelAccess(fallbackModel); err != nil {
			return fmt.Errorf("no accessible Bedrock models found. Please enable model access in AWS Console")
		}

		// Update to use the fallback model
		for _, model := range llm.ModelCosts {
			if model.ModelID == fallbackModel {
				bestModel = model
				break
			}
		}
	}

	fmt.Printf("✅ Selected model: %s\n", bestModel.ModelID)
	fmt.Printf("   Speed: %d/10, Quality: %d/10\n", bestModel.Speed, bestModel.Quality)
	fmt.Printf("   Estimated cost per request: $%.4f\n",
		(1000.0/1000.0)*bestModel.InputTokenCost+(500.0/1000.0)*bestModel.OutputTokenCost)

	// Step 4: Choose AWS region
	fmt.Println("\n3. Choose AWS region:")
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
	for i, region := range regions {
		fmt.Printf("   [%d] %s\n", i+1, region)
	}
	fmt.Println()

	var selectedRegion string
	for {
		fmt.Print("Choose region (1-4): ")
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading region: %w", err)
		}
		choice = strings.TrimSpace(choice)

		regionIndex, err := strconv.Atoi(choice)
		if err != nil || regionIndex < 1 || regionIndex > len(regions) {
			fmt.Println("❌ Please enter a number between 1 and 4")
			continue
		}

		selectedRegion = regions[regionIndex-1]
		break
	}

	// Save configuration
	viper.Set("model.type", "aws")
	viper.Set("model.aws_type", "bedrock")
	viper.Set("model.model_id", bestModel.ModelID)
	viper.Set("model.region", selectedRegion)
	viper.Set("cost.daily_limit", dailyBudget)
	viper.Set("cost.prioritize_speed", prioritizeSpeed)

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Show environment variables to set
	fmt.Println("\n📋 Configuration saved! You can also set these environment variables:")
	fmt.Printf("export AWS_MODEL_TYPE=bedrock\n")
	fmt.Printf("export AWS_MODEL_ID=%s\n", bestModel.ModelID)
	fmt.Printf("export AWS_REGION=%s\n", selectedRegion)
	fmt.Println()

	fmt.Println("🎉 Remote model setup complete!")
	fmt.Printf("Daily budget: $%.2f\n", dailyBudget)
	fmt.Printf("Selected model: %s\n", bestModel.ModelID)
	fmt.Println("You can now use CloudAI-CLI with fast AWS models.")

	return nil
}

// checkAWSCredentials verifies that AWS credentials are configured
func checkAWSCredentials() error {
	ctx := context.Background()
	_, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	return nil
}

// checkBedrockAccess verifies that Bedrock is accessible and models are enabled
func checkBedrockAccess() error {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Try to list foundation models to verify access
	bedrockClient := bedrock.NewFromConfig(cfg)
	_, err = bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return fmt.Errorf("cannot access Bedrock service: %w", err)
	}

	return nil
}

// testModelAccess tests if a specific model can be invoked
func testModelAccess(modelID string) error {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	bedrockClient := bedrockruntime.NewFromConfig(cfg)

	// Try a minimal test request
	testPrompt := "Hello"
	body := `{"prompt": "` + testPrompt + `", "max_tokens": 1, "temperature": 0.1, "anthropic_version": "bedrock-2023-05-31"}`

	_, err = bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        []byte(body),
	})

	if err != nil {
		return fmt.Errorf("model %s not accessible: %w", modelID, err)
	}

	return nil
}

func saveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := home + "/.cloudai.yaml"
	return viper.WriteConfigAs(configPath)
}

func init() {
	rootCmd.AddCommand(interactiveSetupCmd)
}
