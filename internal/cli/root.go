package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ddjura/cloudai/internal/aws"
	"github.com/ddjura/cloudai/internal/llm"
	"github.com/ddjura/cloudai/internal/output"
	"github.com/ddjura/cloudai/internal/state"
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
        "s3:GetBucketLocation"
      ],
      "Resource": "*"
    }
  ]
}

2. Configure your credentials using one of the following methods:
- AWS CLI profile: aws configure --profile cloudai
- Environment variables: export AWS_ACCESS_KEY_ID=...; export AWS_SECRET_ACCESS_KEY=...; export AWS_DEFAULT_REGION=us-east-1

3. (Optional) Set your default region in ~/.aws/config or via AWS_DEFAULT_REGION.

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

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cloudai.yaml)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format for automation")
	rootCmd.PersistentFlags().BoolVar(&planMode, "plan", false, "print remediation scripts (never executed)")

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(scanCmd)
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

	// 5. Print the answer
	fmt.Println("\n--- AI Answer ---")
	fmt.Println(answer)
	fmt.Println("-----------------")

	return nil
}
