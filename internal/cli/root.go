package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/ddjura/cloudai/internal/aws"
	"github.com/ddjura/cloudai/internal/llm"
	"github.com/ddjura/cloudai/internal/output"
	"github.com/ddjura/cloudai/internal/state"
	"github.com/ddjura/cloudai/internal/sysinfo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	jsonOutput bool
	planMode   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cloudai",
	Short: "Ask your AWS account a question – get the answer in seconds",
	Long: `CloudAI-CLI is a single-binary Go tool that turns plain-English prompts 
into AWS SDK calls, revealing live infrastructure topology and high-level cost drivers.

Examples:
  cloudai "Which Lambda handles GET /users on prod-api?"
  cloudai "What triggers the process-order Lambda?"
  cloudai "Top 3 services by cost last 7 days"`,
	Args: cobra.ExactArgs(1),
	RunE: runQuery,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Guide for setting up AWS credentials and permissions for CloudAI-CLI",
	Long: `This command helps you set up the required AWS IAM permissions and credentials for CloudAI-CLI.

1. Create an IAM user or role with the following policy:

{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "lambda:ListFunctions",
        "lambda:GetFunction",
        "lambda:ListEventSourceMappings",
        "apigateway:GET",
        "apigateway:GET RestApis",
        "apigateway:GET Resources",
        "apigateway:GET Methods",
        "costexplorer:GetCostAndUsage",
        "s3:ListBuckets",
        "s3:GetBucketLocation",
        "bedrock:InvokeModel",
        "bedrock:ListFoundationModels",
        "bedrock:GetFoundationModel"
      ],
      "Resource": "*"
    }
  ]
}

2. Configure your credentials using one of the following methods:
- AWS CLI profile: aws configure --profile cloudai
- Environment variables: export AWS_ACCESS_KEY_ID=...; export AWS_SECRET_ACCESS_KEY=...; export AWS_DEFAULT_REGION=us-east-1

3. Enable Bedrock model access (for AWS AI models):
- Go to AWS Console → Amazon Bedrock
- Navigate to 'Model access' in the left sidebar
- Click 'Enable specific models' or 'Enable all models'
- At minimum, enable: Anthropic Claude (recommended)
- Submit the request and wait for approval (usually instant)

4. (Optional) Set your default region in ~/.aws/config or via AWS_DEFAULT_REGION.

This command will now verify your credentials by listing your Lambda functions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n=== CloudAI-CLI AWS Setup Guide ===\n")
		fmt.Println("1. Create an IAM user or role with the following policy:\n")
		fmt.Println(`{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "lambda:ListFunctions",
        "lambda:GetFunction",
        "lambda:ListEventSourceMappings",
        "apigateway:GET",
        "apigateway:GET RestApis",
        "apigateway:GET Resources",
        "apigateway:GET Methods",
        "costexplorer:GetCostAndUsage",
        "s3:ListBuckets",
        "s3:GetBucketLocation"
      ],
      "Resource": "*"
    }
  ]
}`)
		fmt.Println("\n2. Configure your credentials using one of the following methods:")
		fmt.Println("- AWS CLI profile: aws configure --profile cloudai")
		fmt.Println("- Environment variables: export AWS_ACCESS_KEY_ID=...; export AWS_SECRET_ACCESS_KEY=...; export AWS_DEFAULT_REGION=us-east-1")
		fmt.Println("\n3. (Optional) Set your default region in ~/.aws/config or via AWS_DEFAULT_REGION.")
		fmt.Println("\nVerifying your AWS credentials by listing Lambda functions...\n")

		ctx := context.Background()
		awsClient, err := aws.NewClient(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ AWS client initialization failed: %v\n", err)
			return err
		}
		// Try to list Lambda functions
		resp, err := awsClient.Lambda.ListFunctions(ctx, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Unable to list Lambda functions: %v\n", err)
			fmt.Fprintf(os.Stderr, "Check your credentials and permissions.\n")
			return err
		}
		fmt.Printf("✅ Success! Found %d Lambda functions.\n", len(resp.Functions))
		fmt.Println("CloudAI-CLI is ready to use!\n")
		return nil
	},
}

var listModelsCmd = &cobra.Command{
	Use:   "list-models",
	Short: "List all available Bedrock models in your region",
	Long: `Lists all available Bedrock foundation models in your current AWS region.

This helps you see what models are available before enabling them.
Models marked as "Available" can be enabled for use.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n📋 Available Bedrock Models\n")

		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}

		region := cfg.Region
		if region == "" {
			region = "us-east-1"
		}
		fmt.Printf("Region: %s\n\n", region)

		bedrockClient := bedrock.NewFromConfig(cfg)
		resp, err := bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
		if err != nil {
			return fmt.Errorf("failed to list models: %w", err)
		}

		if len(resp.ModelSummaries) == 0 {
			fmt.Println("❌ No models found in this region")
			fmt.Println("💡 Try a different region like us-east-1 or us-west-2")
			return nil
		}

		// Group models by provider
		providers := make(map[string][]string)
		for _, model := range resp.ModelSummaries {
			provider := "Unknown"
			if model.ProviderName != nil {
				provider = *model.ProviderName
			}
			modelName := "Unknown"
			if model.ModelName != nil {
				modelName = *model.ModelName
			}
			modelID := "Unknown"
			if model.ModelId != nil {
				modelID = *model.ModelId
			}

			providers[provider] = append(providers[provider], fmt.Sprintf("%s (%s)", modelName, modelID))
		}

		// Display models by provider
		for provider, models := range providers {
			fmt.Printf("🏢 %s:\n", provider)
			for _, model := range models {
				fmt.Printf("   • %s\n", model)
			}
			fmt.Println()
		}

		fmt.Printf("📊 Total: %d models available\n", len(resp.ModelSummaries))
		fmt.Println("\n💡 To enable models:")
		fmt.Printf("   🌐 Console: https://%s.console.aws.amazon.com/bedrock/home?region=%s#/modelaccess\n", region, region)
		fmt.Println("   🔧 CLI: cloudai bedrock-setup")

		return nil
	},
}

var bedrockSetupCmd = &cobra.Command{
	Use:   "bedrock-setup",
	Short: "Enable and test AWS Bedrock model access",
	Long: `This command helps you enable AWS Bedrock model access and tests your setup.

It will:
1. Check your AWS credentials
2. Test Bedrock service access
3. List available models in your region
4. Automatically open AWS Console to enable model access
5. Guide you through the process step-by-step
6. Test models continuously until access is granted

This completely automates the Bedrock setup process.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n🔧 AWS Bedrock Setup Assistant\n")

		// Check AWS credentials
		fmt.Println("1. Checking AWS credentials...")
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			fmt.Printf("❌ AWS credentials issue: %v\n", err)
			fmt.Println("\n📋 To fix this:")
			fmt.Println("   - Install AWS CLI: https://aws.amazon.com/cli/")
			fmt.Println("   - Run: aws configure")
			fmt.Println("   - Or set environment variables")
			return err
		}
		fmt.Println("✅ AWS credentials found!")

		// Get current region
		region := cfg.Region
		if region == "" {
			region = "us-east-1"
		}
		fmt.Printf("   Using region: %s\n", region)

		// Check Bedrock service access
		fmt.Println("\n2. Checking Bedrock service access...")
		bedrockClient := bedrock.NewFromConfig(cfg)
		modelsResp, err := bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
		if err != nil {
			fmt.Printf("❌ Cannot access Bedrock service: %v\n", err)
			fmt.Println("\n📋 Required IAM permissions:")
			fmt.Println("   - bedrock:InvokeModel")
			fmt.Println("   - bedrock:ListFoundationModels")
			fmt.Println("   - bedrock:GetFoundationModel")
			return err
		}
		fmt.Println("✅ Bedrock service accessible!")

		// Show available models
		fmt.Printf("   Found %d foundation models in %s\n", len(modelsResp.ModelSummaries), region)

		// Test available models
		fmt.Println("\n3. Testing model access...")
		availableModel := findAvailableBedrockModel(ctx, cfg)
		if availableModel != "" {
			fmt.Printf("✅ Found working model: %s\n", availableModel)
			fmt.Println("\n🎉 Bedrock setup complete!")
			fmt.Println("You can now use CloudAI-CLI with AWS models.")
			return nil
		}

		// No models available - guide user through enabling them
		fmt.Println("❌ No models are currently accessible")
		fmt.Println("   This is normal - models need to be explicitly enabled")
		fmt.Println("\n🚀 Let's enable Bedrock models!")
		fmt.Println()

		// Show recommended models
		fmt.Println("📋 Recommended models to enable:")
		fmt.Println("   🥇 Anthropic Claude 3 Haiku (fast, affordable)")
		fmt.Println("   🥈 Amazon Titan Text Express (AWS native)")
		fmt.Println("   🥉 Anthropic Claude 3 Sonnet (high quality)")
		fmt.Println()

		// Automatically open AWS Console
		consoleURL := fmt.Sprintf("https://%s.console.aws.amazon.com/bedrock/home?region=%s#/modelaccess", region, region)
		fmt.Printf("📱 Opening AWS Console: %s\n", consoleURL)

		if err := openBrowser(consoleURL); err != nil {
			fmt.Printf("⚠️  Could not open browser automatically: %v\n", err)
			fmt.Printf("🔗 Please open this URL manually: %s\n", consoleURL)
		}

		fmt.Println("\n📋 In the AWS Console (follow these steps):")
		fmt.Println("   1. ✅ Click 'Enable all models' (orange button)")
		fmt.Println("      OR click 'Enable specific models' (blue button)")
		fmt.Println("   2. ✅ If specific: Select 'Anthropic Claude' models")
		fmt.Println("   3. ✅ Click 'Next' button")
		fmt.Println("   4. ✅ Review and click 'Submit' button")
		fmt.Println("   5. ✅ Wait for 'Access granted' status (10-30 seconds)")
		fmt.Println()

		// Ask user to confirm they've submitted
		fmt.Print("Press Enter after you've clicked 'Submit' in the AWS Console...")
		fmt.Scanln()

		// Continuously test until access is granted
		fmt.Println("\n⏳ Waiting for model access to be enabled...")
		fmt.Println("   (This tool will automatically detect when it's ready)")
		fmt.Println()

		return waitForModelAccess(ctx, cfg)
	},
}

var autoSetupCmd = &cobra.Command{
	Use:   "auto-setup",
	Short: "Completely automated setup - does everything for you",
	Long: `This command completely automates the CloudAI-CLI setup process:

1. Checks AWS credentials
2. Automatically enables Bedrock model access
3. Selects the best model for your budget
4. Configures cost controls
5. Tests everything works

Just run this one command and CloudAI-CLI will be ready to use!`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("\n🚀 CloudAI-CLI Auto Setup")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("This will set up everything automatically!")
		fmt.Println()

		// Step 1: Check AWS credentials
		fmt.Println("1️⃣  Checking AWS credentials...")
		ctx := context.Background()
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			fmt.Printf("❌ AWS credentials not found: %v\n", err)
			fmt.Println("\n📋 Quick fix:")
			fmt.Println("   aws configure")
			fmt.Println("   # Enter your AWS Access Key ID and Secret")
			return fmt.Errorf("AWS credentials required")
		}
		fmt.Println("✅ AWS credentials found!")

		// Step 2: Test Bedrock access
		fmt.Println("\n2️⃣  Testing Bedrock access...")
		availableModel := findAvailableBedrockModel(ctx, cfg)
		if availableModel != "" {
			fmt.Printf("✅ Bedrock already enabled! Found model: %s\n", availableModel)
		} else {
			fmt.Println("⚠️  Bedrock models need to be enabled")
			fmt.Println("\n🔧 Enabling Bedrock access automatically...")

			// Get region for console URL
			region := cfg.Region
			if region == "" {
				region = "us-east-1"
			}

			// Open console automatically
			consoleURL := fmt.Sprintf("https://%s.console.aws.amazon.com/bedrock/home?region=%s#/modelaccess", region, region)
			fmt.Printf("📱 Opening AWS Console: %s\n", consoleURL)

			if err := openBrowser(consoleURL); err != nil {
				fmt.Printf("🔗 Please open manually: %s\n", consoleURL)
			}

			fmt.Println("\n📋 In the AWS Console:")
			fmt.Println("   ✅ Click 'Enable specific models'")
			fmt.Println("   ✅ Select 'Anthropic Claude' (recommended)")
			fmt.Println("   ✅ Click 'Next' → 'Submit'")
			fmt.Println()
			fmt.Print("Press Enter when you've submitted the request...")
			fmt.Scanln()

			fmt.Println("\n⏳ Waiting for model access...")
			if err := waitForModelAccess(ctx, cfg); err != nil {
				return fmt.Errorf("failed to enable Bedrock access: %w", err)
			}
		}

		// Step 3: Configure model and budget
		fmt.Println("\n3️⃣  Configuring optimal settings...")

		// Set reasonable defaults
		dailyBudget := 5.0
		prioritizeSpeed := true

		// Get the best available model
		availableModel = findAvailableBedrockModel(ctx, cfg)
		if availableModel == "" {
			return fmt.Errorf("no Bedrock models available after setup")
		}

		// Find the model cost info
		var bestModel llm.ModelCost
		for _, model := range llm.ModelCosts {
			if model.ModelID == availableModel {
				bestModel = model
				break
			}
		}

		if bestModel.ModelID == "" {
			// Default to Claude Haiku if not found
			bestModel = llm.ModelCost{
				ModelID:         availableModel,
				InputTokenCost:  0.25,
				OutputTokenCost: 1.25,
				Speed:           9,
				Quality:         7,
			}
		}

		fmt.Printf("✅ Selected model: %s\n", bestModel.ModelID)
		fmt.Printf("   Speed: %d/10, Quality: %d/10\n", bestModel.Speed, bestModel.Quality)
		fmt.Printf("   Daily budget: $%.2f\n", dailyBudget)

		// Step 4: Save configuration
		fmt.Println("\n4️⃣  Saving configuration...")

		region := cfg.Region
		if region == "" {
			region = "us-east-1"
		}

		viper.Set("model.type", "aws")
		viper.Set("model.aws_type", "bedrock")
		viper.Set("model.model_id", bestModel.ModelID)
		viper.Set("model.region", region)
		viper.Set("cost.daily_limit", dailyBudget)
		viper.Set("cost.prioritize_speed", prioritizeSpeed)

		home, _ := os.UserHomeDir()
		configPath := home + "/.cloudai.yaml"
		if err := viper.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("✅ Configuration saved to: %s\n", configPath)

		// Step 5: Final test
		fmt.Println("\n5️⃣  Testing complete setup...")

		// Quick test with the LLM client
		llmClient, err := llm.NewClient()
		if err != nil {
			fmt.Printf("⚠️  Setup complete but test failed: %v\n", err)
		} else {
			// Test with a simple query
			_, err = llmClient.Answer(ctx, "Hello", `{"test": "data"}`)
			if err != nil {
				fmt.Printf("⚠️  Setup complete but test failed: %v\n", err)
			} else {
				fmt.Println("✅ End-to-end test successful!")
			}
		}

		// Success!
		fmt.Println("\n🎉 AUTO SETUP COMPLETE!")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("CloudAI-CLI is ready to use!")
		fmt.Println()
		fmt.Println("📋 What's configured:")
		fmt.Printf("   • Model: %s\n", bestModel.ModelID)
		fmt.Printf("   • Region: %s\n", region)
		fmt.Printf("   • Daily budget: $%.2f\n", dailyBudget)
		fmt.Println()
		fmt.Println("🚀 Try it now:")
		fmt.Println("   cd demo-cdk")
		fmt.Println("   cloudai scan")
		fmt.Println("   cloudai \"Which Lambda handles GET /hello?\"")
		fmt.Println()
		fmt.Println("💡 Commands:")
		fmt.Println("   cloudai cost      # Check usage")
		fmt.Println("   cloudai model     # See model info")
		fmt.Println("   cloudai --help    # All commands")

		return nil
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan an IaC project or AWS account to build a knowledge base",
	Long: `Scans a given directory for Infrastructure as Code (IaC) files (like CDK, Terraform)
or a live AWS account to create a cache of the infrastructure state.

This cached state is then used to answer general questions about your infrastructure.
If no path is provided, it scans the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		scanPath := "."
		if len(args) > 0 {
			scanPath = args[0]
		}
		absPath, err := filepath.Abs(scanPath)
		if err != nil {
			return fmt.Errorf("error getting absolute path: %w", err)
		}

		fmt.Printf("Scanning for infrastructure in: %s\n", absPath)

		iacProvider := &state.IaCProvider{}
		infraState, err := iacProvider.Scan(context.Background(), absPath)

		formatter := output.NewFormatter(jsonOutput)
		var result *output.Result

		if err != nil {
			result = &output.Result{
				Query:   fmt.Sprintf("scan %s", scanPath),
				Error:   err.Error(),
				Success: false,
			}
		} else {
			// Save the successful scan to cache
			cacheManager := state.NewCacheManager(absPath)
			if err := cacheManager.Save(infraState); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not save cache: %v\n", err)
			} else {
				fmt.Println("Successfully saved infrastructure state to .cloudai/cache.json")
			}

			result = &output.Result{
				Query:   fmt.Sprintf("scan %s", scanPath),
				Data:    infraState,
				Success: true,
			}
		}

		return formatter.FormatResult(result)
	},
}

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Show information about the current LLM model and available options",
	Long: `Shows information about the currently selected LLM model and available options.

This command will:
1. Detect your system specifications
2. Show what model is currently selected (AWS, Ollama, or OpenAI)
3. List available models in Ollama and AWS
4. Suggest the best model for your system`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🤖 CloudAI-CLI Model Information\n")

		// Detect system specs
		specs, err := sysinfo.DetectSystemSpecs()
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to detect system specs: %v\n", err)
			return err
		}
		fmt.Printf("💻 System: %s\n\n", specs.String())

		// Check AWS model configuration first
		awsConfig := llm.LoadAWSModelFromConfig()
		if awsConfig != nil {
			fmt.Printf("🚀 AWS Model Configured: %s (%s)\n", awsConfig.ModelID, awsConfig.Type)
			fmt.Printf("   Region: %s\n", awsConfig.Region)
			if awsConfig.EndpointName != "" {
				fmt.Printf("   Endpoint: %s\n", awsConfig.EndpointName)
			}
			fmt.Println()
		}

		// Check Ollama availability
		ollamaURL := os.Getenv("OLLAMA_URL")
		if ollamaURL == "" {
			ollamaURL = "http://localhost:11434"
		}

		if isOllamaAvailable(ollamaURL) {
			// Get current model
			currentModel := os.Getenv("OLLAMA_MODEL")
			if currentModel == "" {
				fmt.Println("🔍 Auto-selecting best model for your system...")
				bestModel, err := llm.SelectBestModel(ollamaURL)
				if err != nil {
					fmt.Fprintf(os.Stderr, "❌ Failed to auto-select model: %v\n", err)
					return err
				}
				currentModel = bestModel
			}

			fmt.Printf("✅ Current Ollama model: %s\n", llm.GetModelDisplayName(currentModel))

			// List available models
			fmt.Println("\n📋 Available models in Ollama:")
			availableModels, err := getAvailableModels(ollamaURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to get available models: %v\n", err)
				return err
			}

			for _, model := range availableModels {
				marker := " "
				if model.Name == currentModel {
					marker = "→"
				}
				fmt.Printf("   %s %s\n", marker, llm.GetModelDisplayName(model.Name))
			}

			// Show model requirements
			fmt.Println("\n📊 Ollama Model Requirements:")
			for _, req := range llm.ModelRequirements {
				marker := " "
				if req.Name == currentModel {
					marker = "→"
				}
				gpuReq := "No"
				if req.NeedsGPU {
					gpuReq = "Yes"
				}
				fmt.Printf("   %s %s: %d GB RAM, %d CPU cores, GPU: %s\n",
					marker, llm.GetModelDisplayName(req.Name), req.MinRAMGB, req.MinCPUs, gpuReq)
			}
		} else {
			fmt.Println("❌ Ollama is not running or not accessible")
			fmt.Println("   Start Ollama: ollama serve")
			fmt.Println("   Install a model: ollama pull llama3.2:1b")
		}

		// Show AWS models
		fmt.Println("\n☁️  Available AWS Models (for faster inference):")
		awsModels := llm.GetAvailableAWSModels()
		for _, model := range awsModels {
			fmt.Printf("   • %s (%s) - %s\n", model.ModelID, model.Type, model.Region)
		}

		fmt.Println("\n💡 Tips:")
		if awsConfig == nil {
			fmt.Println("   🚀 For faster inference, configure an AWS model:")
			fmt.Println("      export AWS_MODEL_TYPE=bedrock")
			fmt.Println("      export AWS_MODEL_ID=anthropic.claude-3-haiku-20240307-v1:0")
			fmt.Println("      export AWS_REGION=us-east-1")
		}
		if isOllamaAvailable(ollamaURL) {
			fmt.Println("   • Set OLLAMA_MODEL to override auto-selection")
			fmt.Println("   • Install more models: ollama pull llama3.2:3b")
			fmt.Println("   • Smaller models are faster but less accurate")
		}
		fmt.Println("   • AWS models are much faster but require AWS credentials")
		fmt.Println("   • Priority: AWS > Ollama > OpenAI")

		return nil
	},
}

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show current cost usage and budget information",
	Long: `Shows current daily cost usage and remaining budget for AWS models.

This command displays:
- Current daily spending
- Remaining budget
- Number of requests made today
- Cost per request statistics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("💰 CloudAI-CLI Cost Information\n")

		// Check if using AWS models
		modelType := getConfigString("model.type")
		if modelType != "aws" {
			fmt.Println("ℹ️  Cost tracking is only available for AWS models.")
			fmt.Println("   Local Ollama models are free to use.")
			fmt.Println("   Run 'cloudai setup-interactive' to configure AWS models.")
			return nil
		}

		// Load cost manager
		dailyLimit := getConfigFloat("cost.daily_limit")
		if dailyLimit == 0 {
			dailyLimit = 5.0 // Default
		}

		costManager := llm.NewCostManager(dailyLimit)
		usage := costManager.GetUsageStats()
		remaining := costManager.GetRemainingBudget()

		// Display current usage
		fmt.Printf("📊 Daily Usage (today: %s)\n", usage.Date)
		fmt.Printf("   Spent: $%.4f / $%.2f\n", usage.TotalCost, dailyLimit)
		fmt.Printf("   Remaining: $%.4f\n", remaining)
		fmt.Printf("   Requests: %d\n", usage.RequestCount)
		fmt.Printf("   Tokens used: %d\n", usage.TokensUsed)

		if usage.RequestCount > 0 {
			avgCost := usage.TotalCost / float64(usage.RequestCount)
			fmt.Printf("   Avg cost per request: $%.4f\n", avgCost)
		}

		// Show progress bar
		percentage := (usage.TotalCost / dailyLimit) * 100
		fmt.Printf("\n📈 Budget Usage: %.1f%%\n", percentage)

		barWidth := 30
		filled := int((percentage / 100) * float64(barWidth))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Printf("   [%s]\n", bar)

		// Show model information
		modelID := getConfigString("model.model_id")
		if modelCost := llm.GetModelCost(modelID); modelCost != nil {
			fmt.Printf("\n🤖 Current Model: %s\n", modelID)
			fmt.Printf("   Input cost: $%.4f per 1K tokens\n", modelCost.InputTokenCost)
			fmt.Printf("   Output cost: $%.4f per 1K tokens\n", modelCost.OutputTokenCost)
			fmt.Printf("   Speed: %d/10, Quality: %d/10\n", modelCost.Speed, modelCost.Quality)
		}

		// Warnings
		if percentage > 80 {
			fmt.Printf("\n⚠️  Warning: You've used %.1f%% of your daily budget\n", percentage)
		}

		if remaining < 0.01 {
			fmt.Println("\n🚫 Daily budget exceeded! No more requests allowed today.")
		}

		return nil
	},
}

// Helper functions for the model command
func isOllamaAvailable(url string) bool {
	resp, err := http.Get(url + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func getAvailableModels(ollamaURL string) ([]llm.AvailableModel, error) {
	resp, err := http.Get(ollamaURL + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var result struct {
		Models []llm.AvailableModel `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	return result.Models, nil
}

func getConfigString(key string) string {
	return viper.GetString(key)
}

func getConfigFloat(key string) float64 {
	return viper.GetFloat64(key)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cloudai.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format for automation")
	rootCmd.PersistentFlags().BoolVar(&planMode, "plan", false, "print remediation scripts (never executed)")

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(bedrockSetupCmd)
	rootCmd.AddCommand(autoSetupCmd)
	rootCmd.AddCommand(listModelsCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(modelCmd)
	rootCmd.AddCommand(costCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cloudai" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cloudai")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func runQuery(cmd *cobra.Command, args []string) error {
	userQuery := args[0]
	ctx := context.Background()

	// 1. Find and load the infrastructure context from cache
	// We assume the user is running the command from a path that contains the cache
	// A more robust solution would search parent directories
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}
	cacheManager := state.NewCacheManager(cwd)
	if !cacheManager.Exists() {
		return fmt.Errorf("no infrastructure cache found in this directory. Please run `cloudai scan` first")
	}

	infraState, err := cacheManager.Load()
	if err != nil {
		return fmt.Errorf("could not load infrastructure cache: %w", err)
	}

	// 2. Serialize the context for the LLM prompt
	contextBytes, err := json.Marshal(infraState)
	if err != nil {
		return fmt.Errorf("could not serialize infrastructure state for LLM: %w", err)
	}
	contextString := string(contextBytes)

	// 3. Initialize the LLM client
	llmClient, err := llm.NewClient()
	if err != nil {
		return fmt.Errorf("could not initialize LLM client: %w", err)
	}

	// 4. Ask the LLM to answer the question using the provided context
	fmt.Println("Asking AI to reason about your infrastructure...")
	answer, err := llmClient.Answer(ctx, userQuery, contextString)
	if err != nil {
		return fmt.Errorf("AI failed to answer the question: %w", err)
	}

	// 5. Print the answer in a cleaner format
	fmt.Println("\n🤖 AI Answer:")
	fmt.Println("─" + strings.Repeat("─", 50))
	fmt.Println(strings.TrimSpace(answer))
	fmt.Println("─" + strings.Repeat("─", 50))

	return nil
}

// findAvailableBedrockModel tests common models to find one that works
func findAvailableBedrockModel(ctx context.Context, cfg awssdk.Config) string {
	bedrockRuntimeClient := bedrockruntime.NewFromConfig(cfg)

	// Test models in order of preference
	testModels := []string{
		"anthropic.claude-3-haiku-20240307-v1:0",
		"anthropic.claude-3-sonnet-20240229-v1:0",
		"amazon.titan-text-express-v1",
		"meta.llama3.2-70b-instruct-v1:0",
	}

	for _, modelID := range testModels {
		if testModelQuietly(ctx, bedrockRuntimeClient, modelID) {
			return modelID
		}
	}

	return ""
}

// testModelQuietly tests a model without printing errors
func testModelQuietly(ctx context.Context, client *bedrockruntime.Client, modelID string) bool {
	testBody := `{"prompt": "Hi", "max_tokens": 1, "temperature": 0.1, "anthropic_version": "bedrock-2023-05-31"}`

	_, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     awssdk.String(modelID),
		ContentType: awssdk.String("application/json"),
		Body:        []byte(testBody),
	})

	return err == nil
}

// waitForModelAccess continuously tests until a model becomes available
func waitForModelAccess(ctx context.Context, cfg awssdk.Config) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	attempts := 0
	maxAttempts := 60 // 5 minutes max

	for {
		select {
		case <-ticker.C:
			attempts++

			// Test for available models
			availableModel := findAvailableBedrockModel(ctx, cfg)
			if availableModel != "" {
				fmt.Printf("\n✅ Success! Model access enabled: %s\n", availableModel)
				fmt.Println("\n🎉 Bedrock setup complete!")
				fmt.Println("You can now use CloudAI-CLI with AWS models.")
				fmt.Println("\nNext steps:")
				fmt.Println("   - Run: cloudai setup-interactive")
				fmt.Println("   - Choose option 2 (Remote models)")
				return nil
			}

			// Show progress
			if attempts%6 == 0 { // Every 30 seconds
				fmt.Printf("⏳ Still waiting... (%d minutes elapsed)\n", attempts/12)
				fmt.Println("   💡 Make sure you clicked 'Submit' in the AWS Console")
			} else {
				fmt.Print(".")
			}

			if attempts >= maxAttempts {
				fmt.Println("\n⏰ Timeout reached. Model access may take longer than expected.")
				fmt.Println("\n📋 Manual verification:")
				fmt.Println("   1. Check the AWS Console for any pending requests")
				fmt.Println("   2. Some models may require manual approval")
				fmt.Println("   3. Try running 'cloudai bedrock-setup' again in a few minutes")
				return fmt.Errorf("timeout waiting for model access")
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// openBrowser attempts to open a URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
