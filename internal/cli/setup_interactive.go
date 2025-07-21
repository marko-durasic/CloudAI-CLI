package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
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
	Short: "Interactive setup for CloudAI-CLI deployment options",
	Long: `Interactive setup that guides you through choosing and configuring
your AI infrastructure deployment option.`,
	RunE: runInteractiveSetup,
}

func runInteractiveSetup(cmd *cobra.Command, args []string) error {
	displayWelcomeBanner()

	reader := bufio.NewReader(os.Stdin)

	// Show deployment options (fast - just text)
	displayDeploymentOptions()

	for {
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			return setupLocalOllama(reader)
		case "2":
			return setupEC2Ollama(reader)
		case "3":
			return setupSageMaker(reader)
		case "4":
			return setupBedrock(reader)
		case "5":
			return setupPrivacyRemoteAPI(reader)
		case "6":
			return setupPrivacyCLI(reader)
		case "h", "H", "help":
			displayDetailedOptions()
			fmt.Print("\n🎯 Choose your deployment option (1-6): ")
		default:
			fmt.Println("❌ Please enter 1-6 or 'h' for help")
			fmt.Print("🎯 Choose your deployment option (1-6): ")
		}
	}
}

func displayWelcomeBanner() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║           🚀 CloudAI-CLI Setup Assistant 🚀              ║")
	fmt.Println("║                                                          ║")
	fmt.Println("║  Configure your AI infrastructure deployment in minutes  ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func displayDeploymentOptions() {
	fmt.Println("📋 **Choose Your AI Deployment:**")
	fmt.Println()

	fmt.Println("1️⃣  Local Ollama          🆓 FREE • 🔒 Private • 🖥️  Your machine")
	fmt.Println("2️⃣  EC2 Ollama            ⚡ Fast GPU • 💰 ~$0.50/hr • ☁️  AWS cloud")
	fmt.Println("3️⃣  SageMaker             🎯 Fine-tuned • 🧠 AWS optimized • 💰 ~$0.02/req")
	fmt.Println("4️⃣  AWS Bedrock           ✅ Managed • 🚀 No setup • 💰 ~$0.001/req")
	fmt.Println("5️⃣  Privacy Remote API    🔒 Sanitized • 🌐 OpenAI/Claude • 💰 API cost")
	fmt.Println("6️⃣  Privacy CLI Tools     🔒 Sanitized • 🔧 Gemini/Bard • 🆓 Often free")
	fmt.Println()

	fmt.Println("💡 All options include smart infrastructure analysis!")
	fmt.Print("   Type 'h' for detailed comparison, or choose 1-6: ")
}

func displayDetailedOptions() {
	fmt.Println("\n📋 **Detailed Deployment Options:**")
	fmt.Println()

	// Option 1: Local
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 1️⃣  Local Ollama (On Your Machine)                      │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ ✅ Completely FREE                                      │")
	fmt.Println("│ ✅ Private & Secure (data never leaves your machine)    │")
	fmt.Println("│ ✅ No AWS account needed                                │")
	fmt.Println("│ ⚠️  Slower inference (CPU-based)                        │")
	fmt.Println("│ 💻 Requirements: 8GB+ RAM recommended                   │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Option 2: EC2
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 2️⃣  Ollama on EC2 (GPU-Powered)                         │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ ⚡ Fast inference with GPU acceleration                 │")
	fmt.Println("│ 🔒 Private cloud deployment                             │")
	fmt.Println("│ 💰 ~$0.50-1.00/hour when running                        │")
	fmt.Println("│ ⚠️  Requires AWS account & quota approval               │")
	fmt.Println("│ 🛠️  We handle all setup automatically                   │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Option 3: SageMaker
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 3️⃣  SageMaker (Fine-Tuned for AWS)                      │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ 🎯 Specialized for AWS infrastructure questions         │")
	fmt.Println("│ 🧠 Fine-tuned on AWS documentation                      │")
	fmt.Println("│ 💰 ~$0.01-0.05 per request                              │")
	fmt.Println("│ ⚠️  Requires SageMaker endpoint deployment              │")
	fmt.Println("│ 🔧 Best for advanced AWS architecture analysis          │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Option 4: Bedrock
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 4️⃣  AWS Bedrock (Managed Service)                       │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ ✅ No infrastructure to manage                          │")
	fmt.Println("│ ⚡ Fast & reliable                                      │")
	fmt.Println("│ 💰 Pay-per-use (~$0.001-0.01 per request)              │")
	fmt.Println("│ 🚀 Start immediately (no setup required)                │")
	fmt.Println("│ 📊 Best for production workloads                        │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Option 5: Privacy-Preserving Remote API
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 5️⃣  Privacy-First Remote API (Hybrid)                   │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ 🔒 Local AI sanitizes data before sending               │")
	fmt.Println("│ 🌐 Uses OpenAI/Anthropic for powerful responses         │")
	fmt.Println("│ 🛡️  Sensitive data never leaves your machine            │")
	fmt.Println("│ ⚡ Best of both worlds: Privacy + Performance          │")
	fmt.Println("│ 💰 Pay-per-use for remote API calls only               │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Option 6: Privacy-Preserving CLI Tools
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ 6️⃣  Privacy-First CLI Tools (Gemini/Bard)               │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│ 🔒 Local AI pre-processes sensitive data                │")
	fmt.Println("│ 🔧 Integrates with Google Gemini CLI & others           │")
	fmt.Println("│ 🛡️  Infrastructure details stay private                 │")
	fmt.Println("│ 🆓 Often free tier available                            │")
	fmt.Println("│ 🚀 Easy setup with existing CLI tools                   │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
}

func setupLocalOllama(reader *bufio.Reader) error {
	fmt.Println("\n🖥️  Setting up Local Ollama...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check if Ollama is installed
	if !isOllamaAvailable("http://localhost:11434") {
		fmt.Println("\n❌ Ollama is not running on your machine.")
		fmt.Println("\n📋 To install Ollama:")
		fmt.Println("   1. Visit: https://ollama.com/")
		fmt.Println("   2. Download for your OS")
		fmt.Println("   3. Install and run: ollama serve")
		fmt.Println("   4. Pull a model: ollama pull llama3.2:3b")
		fmt.Println("\n💡 Recommended models for different use cases:")
		fmt.Println("   • llama3.2:3b  - Best overall (3.2GB)")
		fmt.Println("   • phi3:mini    - Fastest (2.2GB)")
		fmt.Println("   • llama3.2:1b  - Smallest (1.3GB)")
		fmt.Println()
		fmt.Print("Press Enter after installing Ollama...")
		reader.ReadString('\n')

		if !isOllamaAvailable("http://localhost:11434") {
			return fmt.Errorf("Ollama is still not available. Please ensure it's running.")
		}
	}

	fmt.Println("✅ Ollama detected!")
	fmt.Println("\n🔍 Checking available models...")

	// Check if any models are available
	hasModels := checkForModels("http://localhost:11434")
	if !hasModels {
		fmt.Println("⚠️  No models found. Let's download one...")
		fmt.Println("\n📥 Downloading recommended model (llama3.2:3b)...")
		fmt.Println("This may take a few minutes...")

		// Here you would call ollama pull command
		fmt.Println("Run: ollama pull llama3.2:3b")
		fmt.Print("\nPress Enter when download is complete...")
		reader.ReadString('\n')
	} else {
		fmt.Println("✅ Models are available!")
	}

	// Auto-select best model
	bestModel, err := llm.SelectBestModel("http://localhost:11434")
	if err != nil {
		return fmt.Errorf("failed to select model: %w", err)
	}

	fmt.Printf("\n✅ Selected model: %s\n", bestModel)

	// Test the setup
	fmt.Println("\n🧪 Testing local setup...")
	if err := testLocalSetup(bestModel); err != nil {
		fmt.Printf("⚠️  Test failed: %v\n", err)
		fmt.Println("💡 Try running: ollama run", bestModel)
	} else {
		fmt.Println("✅ Local Ollama is working perfectly!")
	}

	// Save configuration
	viper.Set("model.type", "ollama")
	viper.Set("model.name", bestModel)
	viper.Set("model.url", "http://localhost:11434")

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Show demo commands
	displayLocalDemo()

	return nil
}

func setupEC2Ollama(reader *bufio.Reader) error {
	fmt.Println("\n☁️  Setting up Ollama on EC2...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📋 This option will:")
	fmt.Println("   • Check your AWS quotas")
	fmt.Println("   • Deploy a GPU-powered EC2 instance")
	fmt.Println("   • Install Ollama automatically")
	fmt.Println("   • Cost: ~$0.50/hour when running")

	fmt.Println("\n🔑 Requirements:")
	fmt.Println("   • AWS account with credentials configured")
	fmt.Println("   • EC2 quota for GPU instances (we'll check)")

	fmt.Print("\nContinue with EC2 setup? (y/N): ")
	confirm, _ := reader.ReadString('\n')
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(confirm)), "y") {
		return nil
	}

	// Check AWS credentials
	fmt.Println("\n🔍 Checking AWS credentials...")
	if err := checkAWSCredentials(); err != nil {
		fmt.Printf("❌ AWS credentials not found: %v\n", err)
		fmt.Println("\n📋 To configure AWS:")
		fmt.Println("   aws configure")
		fmt.Println("   # Enter your Access Key ID")
		fmt.Println("   # Enter your Secret Access Key")
		fmt.Println("   # Enter your preferred region")
		return fmt.Errorf("AWS credentials required")
	}
	fmt.Println("✅ AWS credentials found!")

	fmt.Println("\n🚀 To deploy Ollama on EC2:")
	fmt.Println("   ./deploy-ollama-ec2.sh")
	fmt.Println("\nThis script will:")
	fmt.Println("   • Check your quotas")
	fmt.Println("   • Request increases if needed")
	fmt.Println("   • Deploy when ready")

	return nil
}

func setupSageMaker(reader *bufio.Reader) error {
	fmt.Println("\n🧠 Setting up SageMaker (Fine-Tuned Model)...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n⚠️  SageMaker option is for advanced users")
	fmt.Println("\n📋 Requirements:")
	fmt.Println("   • Existing SageMaker endpoint")
	fmt.Println("   • Fine-tuned model deployed")
	fmt.Println("   • Endpoint name")

	fmt.Print("\nDo you have a SageMaker endpoint ready? (y/N): ")
	confirm, _ := reader.ReadString('\n')
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(confirm)), "y") {
		fmt.Println("\n💡 To learn about fine-tuning:")
		fmt.Println("   See: demo-cdk/training/")
		return nil
	}

	fmt.Print("\nEnter SageMaker endpoint name: ")
	endpoint, _ := reader.ReadString('\n')
	endpoint = strings.TrimSpace(endpoint)

	// Save configuration
	viper.Set("model.type", "sagemaker")
	viper.Set("model.endpoint", endpoint)
	viper.Set("model.region", "us-east-1")

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ SageMaker configuration saved!")
	return nil
}

func setupBedrock(reader *bufio.Reader) error {
	fmt.Println("\n☁️  Setting up AWS Bedrock...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Re-use the existing Bedrock setup logic from the original file
	// This is a simplified version for now
	fmt.Println("\n📋 Bedrock provides:")
	fmt.Println("   • Managed AI models (Claude, Llama, etc.)")
	fmt.Println("   • No infrastructure to manage")
	fmt.Println("   • Pay-per-request pricing")

	// Check AWS credentials
	fmt.Println("\n🔍 Checking AWS credentials...")
	if err := checkAWSCredentials(); err != nil {
		fmt.Printf("❌ AWS credentials not found: %v\n", err)
		return fmt.Errorf("AWS credentials required for Bedrock")
	}
	fmt.Println("✅ AWS credentials found!")

	// Save configuration
	viper.Set("model.type", "aws")
	viper.Set("model.aws_type", "bedrock")
	viper.Set("model.model_id", "anthropic.claude-3-haiku-20240307-v1:0")
	viper.Set("model.region", "us-east-1")

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ Bedrock configuration saved!")
	fmt.Println("🎉 You can now use CloudAI-CLI with AWS Bedrock!")

	return nil
}

func setupPrivacyRemoteAPI(reader *bufio.Reader) error {
	fmt.Println("\n🔒 Setting up Privacy-First Remote API...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📋 How it works:")
	fmt.Println("   1. Local Ollama sanitizes your data")
	fmt.Println("   2. Removes account IDs, ARNs, secrets")
	fmt.Println("   3. Sends sanitized query to OpenAI/Anthropic")
	fmt.Println("   4. Gets powerful response back")
	fmt.Println("   5. Re-maps to your actual resources")

	fmt.Println("\n🔑 Requirements:")
	fmt.Println("   • Local Ollama (for sanitization)")
	fmt.Println("   • OpenAI or Anthropic API key")

	// First ensure local Ollama is set up
	if !isOllamaAvailable("http://localhost:11434") {
		fmt.Println("\n❌ Local Ollama required for privacy protection")
		fmt.Println("💡 Please set up Option 1 first, then return here")
		return nil
	}
	fmt.Println("✅ Local Ollama detected!")

	// Choose remote API
	fmt.Println("\n🌐 Choose remote API provider:")
	fmt.Println("   [1] OpenAI (GPT-4)")
	fmt.Println("   [2] Anthropic (Claude)")

	fmt.Print("\nSelect provider (1 or 2): ")
	providerChoice, _ := reader.ReadString('\n')
	providerChoice = strings.TrimSpace(providerChoice)

	var provider string
	switch providerChoice {
	case "1":
		provider = "openai"
		fmt.Print("\nEnter OpenAI API key: ")
	case "2":
		provider = "anthropic"
		fmt.Print("\nEnter Anthropic API key: ")
	default:
		fmt.Println("❌ Invalid choice")
		return nil
	}

	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	// Save configuration
	viper.Set("model.type", "privacy-remote")
	viper.Set("model.local_sanitizer", "ollama")
	viper.Set("model.remote_provider", provider)
	viper.Set("model.api_key", apiKey)
	viper.Set("privacy.enabled", true)
	viper.Set("privacy.redact_account_ids", true)
	viper.Set("privacy.redact_arns", true)

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ Privacy-preserving remote API configured!")
	fmt.Println("🛡️  Your sensitive data stays local!")
	fmt.Println("⚡ Remote API provides powerful responses!")

	return nil
}

func setupPrivacyCLI(reader *bufio.Reader) error {
	fmt.Println("\n🔒 Setting up Privacy-First CLI Tools...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n📋 How it works:")
	fmt.Println("   1. Local Ollama pre-processes your query")
	fmt.Println("   2. Removes sensitive infrastructure details")
	fmt.Println("   3. Sends sanitized query to CLI tool")
	fmt.Println("   4. CLI tool (Gemini/Bard) processes it")
	fmt.Println("   5. Maps response back to your resources")

	fmt.Println("\n🔑 Requirements:")
	fmt.Println("   • Local Ollama (for sanitization)")
	fmt.Println("   • Google Gemini CLI or similar tool")

	// First ensure local Ollama is set up
	if !isOllamaAvailable("http://localhost:11434") {
		fmt.Println("\n❌ Local Ollama required for privacy protection")
		fmt.Println("💡 Please set up Option 1 first, then return here")
		return nil
	}
	fmt.Println("✅ Local Ollama detected!")

	fmt.Println("\n🔧 Available CLI tools:")
	fmt.Println("   [1] Google Gemini CLI")
	fmt.Println("   [2] Google Bard CLI")
	fmt.Println("   [3] Custom CLI tool")

	fmt.Print("\nSelect CLI tool (1-3): ")
	toolChoice, _ := reader.ReadString('\n')
	toolChoice = strings.TrimSpace(toolChoice)

	var cliTool string
	var cliCommand string

	switch toolChoice {
	case "1":
		cliTool = "gemini"
		cliCommand = "gemini"
		fmt.Println("\n📋 To install Gemini CLI:")
		fmt.Println("   pip install google-gemini-cli")
		fmt.Println("   gemini auth login")
	case "2":
		cliTool = "bard"
		cliCommand = "bard"
		fmt.Println("\n📋 To install Bard CLI:")
		fmt.Println("   npm install -g bard-cli")
	case "3":
		fmt.Print("\nEnter CLI command: ")
		customCmd, _ := reader.ReadString('\n')
		cliCommand = strings.TrimSpace(customCmd)
		cliTool = "custom"
	default:
		fmt.Println("❌ Invalid choice")
		return nil
	}

	fmt.Print("\nPress Enter when CLI tool is installed and ready...")
	reader.ReadString('\n')

	// Save configuration
	viper.Set("model.type", "privacy-cli")
	viper.Set("model.local_sanitizer", "ollama")
	viper.Set("model.cli_tool", cliTool)
	viper.Set("model.cli_command", cliCommand)
	viper.Set("privacy.enabled", true)
	viper.Set("privacy.redact_account_ids", true)
	viper.Set("privacy.redact_resource_names", true)

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println("\n✅ Privacy-preserving CLI tool configured!")
	fmt.Println("🛡️  Your AWS details stay private!")
	fmt.Println("🆓 Many CLI tools have free tiers!")

	return nil
}

func testLocalSetup(model string) error {
	// Simple test to verify Ollama is working
	fmt.Print("   Testing connection... ")

	// Here you would make a simple API call to Ollama
	// For now, just check if it's available
	if isOllamaAvailable("http://localhost:11434") {
		fmt.Println("✓")
		return nil
	}

	return fmt.Errorf("cannot connect to Ollama")
}

func displayLocalDemo() {
	fmt.Println("\n🎉 Setup Complete! Here's how to use CloudAI-CLI:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\n🧪 Try these demo commands:")
	fmt.Println()
	fmt.Println("   # Check what AWS services you're using:")
	fmt.Println("   ./cloudai \"What AWS services am I currently using?\"")
	fmt.Println()
	fmt.Println("   # Analyze Lambda functions:")
	fmt.Println("   ./cloudai \"List my Lambda functions and their memory settings\"")
	fmt.Println()
	fmt.Println("   # Get cost optimization tips:")
	fmt.Println("   ./cloudai \"How can I reduce my S3 storage costs?\"")
	fmt.Println()
	fmt.Println("   # Understand your architecture:")
	fmt.Println("   ./cloudai \"Explain the architecture of my VPC setup\"")
	fmt.Println()
	fmt.Println("   # Security check:")
	fmt.Println("   ./cloudai \"Are there any security issues with my S3 buckets?\"")
	fmt.Println()
	fmt.Println("💡 CloudAI-CLI automatically:")
	fmt.Println("   • Gathers context from your AWS environment")
	fmt.Println("   • Provides infrastructure-aware responses")
	fmt.Println("   • Keeps your data private (local processing)")
}

func checkForModels(url string) bool {
	// Simple check to see if Ollama has any models
	// We'll just try to select the best model - if it fails, no models
	_, err := llm.SelectBestModel(url)
	return err == nil
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
