# Shell Command Enhancement Implementation Guide

## Overview

This document outlines the implementation of enhanced shell command capabilities for CloudAI-CLI, focusing on intelligent routing, SageMaker integration, and architecture-specific learning.

## Core Components

### 1. Command Intelligence Router

```go
// internal/routing/command_router.go
package routing

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/ddjura/cloudai/internal/llm"
    "github.com/ddjura/cloudai/internal/privacy"
    "github.com/ddjura/cloudai/internal/training"
)

type CommandRouter struct {
    intentClassifier    *IntentClassifier
    privacyAnalyzer     *privacy.SensitivityAnalyzer
    modelManager        *llm.ModelManager
    trainingCollector   *training.DataCollector
    architectureContext *ArchitectureContext
}

type CommandContext struct {
    Command           string
    User              string
    Timestamp         time.Time
    InfrastructureCtx *InfrastructureContext
    PrivacyLevel      privacy.Level
    PreferredModel    ModelTier
}

type ModelTier int
const (
    TierCustomSageMaker ModelTier = iota
    TierLocalOllama
    TierBedrock
    TierExternalAPI
)

type CommandIntent struct {
    Type                 IntentType
    ArchitectureSpecific bool
    PrivacySensitive     bool
    ResourcesReferenced  []string
    Confidence           float64
    OptimalTier          ModelTier
}

func (r *CommandRouter) RouteCommand(ctx context.Context, cmdCtx *CommandContext) (*CommandResponse, error) {
    // Step 1: Analyze command intent and sensitivity
    intent, err := r.intentClassifier.ClassifyCommand(cmdCtx.Command)
    if err != nil {
        return nil, fmt.Errorf("failed to classify command: %w", err)
    }

    // Step 2: Determine privacy requirements
    privacyLevel := r.privacyAnalyzer.AnalyzeCommand(cmdCtx.Command)
    
    // Step 3: Select optimal model tier
    tier := r.selectModelTier(intent, privacyLevel)
    
    // Step 4: Route to appropriate model
    response, err := r.routeToModel(ctx, cmdCtx, intent, tier)
    if err != nil {
        return nil, fmt.Errorf("failed to route command: %w", err)
    }

    // Step 5: Collect training data
    r.trainingCollector.CollectInteraction(&training.Interaction{
        Command:     cmdCtx.Command,
        Intent:      intent,
        Response:    response,
        ModelTier:   tier,
        Timestamp:   time.Now(),
        Successful:  true,
    })

    return response, nil
}

func (r *CommandRouter) selectModelTier(intent *CommandIntent, privacyLevel privacy.Level) ModelTier {
    // Priority 1: Privacy requirements
    if privacyLevel == privacy.HighSensitivity {
        return TierLocalOllama
    }
    
    // Priority 2: Architecture-specific knowledge
    if intent.ArchitectureSpecific && intent.Confidence > 0.8 {
        return TierCustomSageMaker
    }
    
    // Priority 3: General AWS questions
    if intent.Type == IntentAWSGeneral {
        return TierBedrock
    }
    
    // Priority 4: Complex analysis
    if intent.Type == IntentComplexAnalysis {
        return TierExternalAPI
    }
    
    // Default to custom model
    return TierCustomSageMaker
}
```

### 2. Architecture-Specific SageMaker Integration

```go
// internal/training/sagemaker_trainer.go
package training

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/sagemaker"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type SageMakerTrainer struct {
    sagemakerClient *sagemaker.Client
    s3Client        *s3.Client
    config          *TrainingConfig
}

type TrainingConfig struct {
    ModelName              string
    TrainingJobName        string
    RoleArn               string
    TrainingImage         string
    TrainingInstanceType  string
    TrainingInstanceCount int
    VolumeSize           int
    MaxRuntimeInSeconds  int
    OutputPath           string  // Full S3 URI for SageMaker output (e.g., "s3://my-bucket/training-output/")
    TrainingDataBucket   string  // S3 bucket name for training data storage
    HyperParameters      map[string]string
}

type ArchitectureTrainingData struct {
    InfrastructureSnapshot  *InfrastructureState      `json:"infrastructure"`
    ShellCommandPatterns    []ShellCommandPattern     `json:"shell_patterns"`
    ResourceRelationships   map[string][]string       `json:"relationships"`
    NamingConventions      map[string]string         `json:"naming_conventions"`
    CostPatterns           []CostPattern             `json:"cost_patterns"`
    TroubleshootingCases   []TroubleshootingCase     `json:"troubleshooting"`
    UserPreferences        map[string]interface{}    `json:"user_preferences"`
}

type ShellCommandPattern struct {
    Command              string    `json:"command"`
    Intent               string    `json:"intent"`
    Context              string    `json:"context"`
    SuccessfulResponse   string    `json:"response"`
    ResourcesReferenced  []string  `json:"resources"`
    UserSatisfaction     int       `json:"satisfaction"`
    Timestamp           time.Time `json:"timestamp"`
}

func (t *SageMakerTrainer) TrainCustomModel(ctx context.Context, trainingData *ArchitectureTrainingData) error {
    // Step 1: Prepare training data
    trainingDataPath, err := t.prepareTrainingData(ctx, trainingData)
    if err != nil {
        return fmt.Errorf("failed to prepare training data: %w", err)
    }

    // Step 2: Create training job
    trainingJob := &sagemaker.CreateTrainingJobInput{
        TrainingJobName: &t.config.TrainingJobName,
        RoleArn:        &t.config.RoleArn,
        AlgorithmSpecification: &sagemaker.AlgorithmSpecification{
            TrainingImage:     &t.config.TrainingImage,
            TrainingInputMode: sagemaker.TrainingInputMode("File"),
        },
        InputDataConfig: []sagemaker.Channel{
            {
                ChannelName: aws.String("training"),
                DataSource: &sagemaker.DataSource{
                    S3DataSource: &sagemaker.S3DataSource{
                        S3DataType:         sagemaker.S3DataType("S3Prefix"),
                        S3Uri:             &trainingDataPath,
                        S3DataDistributionType: sagemaker.S3DataDistribution("FullyReplicated"),
                    },
                },
                ContentType: aws.String("application/json"),
            },
        },
        OutputDataConfig: &sagemaker.OutputDataConfig{
            S3OutputPath: &t.config.OutputPath,
        },
        ResourceConfig: &sagemaker.ResourceConfig{
            InstanceType:   sagemaker.TrainingInstanceType(t.config.TrainingInstanceType),
            InstanceCount:  &t.config.TrainingInstanceCount,
            VolumeSizeInGB: &t.config.VolumeSize,
        },
        StoppingCondition: &sagemaker.StoppingCondition{
            MaxRuntimeInSeconds: &t.config.MaxRuntimeInSeconds,
        },
        HyperParameters: t.config.HyperParameters,
    }

    // Step 3: Start training
    result, err := t.sagemakerClient.CreateTrainingJob(ctx, trainingJob)
    if err != nil {
        return fmt.Errorf("failed to create training job: %w", err)
    }

    fmt.Printf("Training job started: %s\n", *result.TrainingJobArn)

    // Step 4: Monitor training progress
    return t.monitorTrainingJob(ctx, t.config.TrainingJobName)
}

func (t *SageMakerTrainer) prepareTrainingData(ctx context.Context, data *ArchitectureTrainingData) (string, error) {
    // Convert to training format
    trainingExamples := t.convertToTrainingExamples(data)
    
    // Upload to S3
    s3Key := fmt.Sprintf("training-data/%s/data.json", time.Now().Format("2006-01-02"))
    
    jsonData, err := json.Marshal(trainingExamples)
    if err != nil {
        return "", fmt.Errorf("failed to marshal training data: %w", err)
    }

    _, err = t.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(t.config.TrainingDataBucket),
        Key:    aws.String(s3Key),
        Body:   bytes.NewReader(jsonData),
    })
    if err != nil {
        return "", fmt.Errorf("failed to upload training data: %w", err)
    }

    return fmt.Sprintf("s3://%s/%s", t.config.TrainingDataBucket, s3Key), nil
}

func (t *SageMakerTrainer) convertToTrainingExamples(data *ArchitectureTrainingData) []TrainingExample {
    var examples []TrainingExample
    
    // Convert shell command patterns to training examples
    for _, pattern := range data.ShellCommandPatterns {
        examples = append(examples, TrainingExample{
            Input:  t.formatCommandInput(pattern.Command, pattern.Context),
            Output: pattern.SuccessfulResponse,
            Metadata: map[string]interface{}{
                "resources": pattern.ResourcesReferenced,
                "intent":    pattern.Intent,
                "satisfaction": pattern.UserSatisfaction,
            },
        })
    }
    
    // Add architecture-specific examples
    for resource, relationships := range data.ResourceRelationships {
        examples = append(examples, TrainingExample{
            Input:  fmt.Sprintf("What connects to %s?", resource),
            Output: fmt.Sprintf("The following resources connect to %s: %s", resource, strings.Join(relationships, ", ")),
            Metadata: map[string]interface{}{
                "type": "relationship",
                "resource": resource,
            },
        })
    }
    
    return examples
}

type TrainingExample struct {
    Input    string                 `json:"input"`
    Output   string                 `json:"output"`
    Metadata map[string]interface{} `json:"metadata"`
}

// NewSageMakerTrainer creates a new SageMakerTrainer with proper configuration
func NewSageMakerTrainer() *SageMakerTrainer {
    // This would be called from the actual implementation
    return &SageMakerTrainer{
        // Initialize AWS clients
        // Set up configuration
    }
}

// SetConfig sets the training configuration
func (t *SageMakerTrainer) SetConfig(config *TrainingConfig) {
    t.config = config
}

// NewTrainingConfig creates a properly configured TrainingConfig
// 
// IMPORTANT: This fixes a previous bug where OutputPath was used inconsistently:
// - OutputPath should be a full S3 URI for SageMaker's output configuration
// - TrainingDataBucket should be just the bucket name for uploading training data
// 
// This separation ensures that:
// 1. SageMaker training jobs receive the correct full S3 URI for model output
// 2. Training data uploads use the correct bucket name
// 3. The configuration is consistent and won't cause job failures
func NewTrainingConfig(trainingDataBucket, outputPath string) *TrainingConfig {
    return &TrainingConfig{
        TrainingDataBucket: trainingDataBucket,  // Just the bucket name (e.g., "my-training-bucket")
        OutputPath:         outputPath,          // Full S3 URI (e.g., "s3://my-training-bucket/model-output/")
        // Other fields would be set based on requirements
    }
}
```

### 3. Enhanced Command Processing

```go
// internal/cli/enhanced_commands.go
package cli

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/ddjura/cloudai/internal/routing"
    "github.com/ddjura/cloudai/internal/privacy"
    "github.com/spf13/cobra"
)

func enhancedQueryCommand() *cobra.Command {
    var (
        localOnly    bool
        sanitize     bool
        explainModel bool
        privacyMode  string
    )

    cmd := &cobra.Command{
        Use:   "query [question]",
        Short: "Ask intelligent questions about your AWS architecture",
        Long: `Ask natural language questions about your AWS infrastructure.
The system will intelligently route your query to the best model tier
based on privacy requirements and architectural specificity.

Examples:
  cloudai query "what triggers the user lambda"
  cloudai query "database connections in staging"
  cloudai query "expensive resources this month"
  cloudai query "production secrets" --local-only
  cloudai query "cost optimization" --explain`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            question := args[0]
            
            // Create command context
            cmdCtx := &routing.CommandContext{
                Command:           question,
                User:              getCurrentUser(),
                Timestamp:         time.Now(),
                InfrastructureCtx: loadInfrastructureContext(),
                PrivacyLevel:      determinePrivacyLevel(privacyMode, localOnly),
                PreferredModel:    determinePreferredModel(localOnly, sanitize),
            }

            // Route command
            router := routing.NewCommandRouter()
            response, err := router.RouteCommand(context.Background(), cmdCtx)
            if err != nil {
                return fmt.Errorf("command routing failed: %w", err)
            }

            // Display response
            fmt.Printf("🤖 %s\n", response.Answer)
            
            if explainModel {
                fmt.Printf("\n📊 Model Info:\n")
                fmt.Printf("   Tier: %s\n", response.ModelTier)
                fmt.Printf("   Confidence: %.2f\n", response.Confidence)
                fmt.Printf("   Response Time: %v\n", response.ResponseTime)
                fmt.Printf("   Cost: $%.4f\n", response.Cost)
            }

            return nil
        },
    }

    cmd.Flags().BoolVar(&localOnly, "local-only", false, "Force processing with local models only")
    cmd.Flags().BoolVar(&sanitize, "sanitize", false, "Enable aggressive data sanitization")
    cmd.Flags().BoolVar(&explainModel, "explain", false, "Show which model tier was used and why")
    cmd.Flags().StringVar(&privacyMode, "privacy", "auto", "Privacy mode: auto, basic, strict")

    return cmd
}

func trainCommand() *cobra.Command {
    var (
        force      bool
        background bool
        dataPath   string
    )

    cmd := &cobra.Command{
        Use:   "train",
        Short: "Train custom SageMaker model on your architecture",
        Long: `Train a custom SageMaker model that understands your specific
AWS architecture patterns, naming conventions, and shell command preferences.

The training process collects:
- Infrastructure topology and relationships
- Shell command patterns and successful responses
- Resource naming conventions
- Cost patterns and optimization opportunities
- Troubleshooting cases and resolutions

Training typically takes 30-60 minutes and costs $10-50 depending on data size.`,
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Println("🚀 Starting custom model training...")
            
            // Collect training data
            fmt.Println("📊 Collecting training data...")
            trainingData, err := collectTrainingData(dataPath)
            if err != nil {
                return fmt.Errorf("failed to collect training data: %w", err)
            }

            fmt.Printf("   Infrastructure resources: %d\n", len(trainingData.InfrastructureSnapshot.Resources))
            fmt.Printf("   Shell command patterns: %d\n", len(trainingData.ShellCommandPatterns))
            fmt.Printf("   Resource relationships: %d\n", len(trainingData.ResourceRelationships))

            // Start training
            trainer := training.NewSageMakerTrainer()
            
            // Configure training with proper S3 paths
            // Note: TrainingDataBucket is just the bucket name, OutputPath is the full S3 URI
            trainingConfig := training.NewTrainingConfig(
                "my-training-bucket",                           // Training data bucket name
                "s3://my-training-bucket/model-output/",       // Full S3 URI for model output
            )
            trainer.SetConfig(trainingConfig)
            
            if background {
                go func() {
                    if err := trainer.TrainCustomModel(context.Background(), trainingData); err != nil {
                        fmt.Printf("❌ Training failed: %v\n", err)
                    } else {
                        fmt.Println("✅ Training completed successfully!")
                    }
                }()
                fmt.Println("🔄 Training started in background")
            } else {
                if err := trainer.TrainCustomModel(context.Background(), trainingData); err != nil {
                    return fmt.Errorf("training failed: %w", err)
                }
                fmt.Println("✅ Training completed successfully!")
            }

            return nil
        },
    }

    cmd.Flags().BoolVar(&force, "force", false, "Force retraining even if recent model exists")
    cmd.Flags().BoolVar(&background, "background", false, "Run training in background")
    cmd.Flags().StringVar(&dataPath, "data", "", "Path to additional training data")

    return cmd
}

func knowledgeCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "knowledge",
        Short: "Manage architecture knowledge base",
        Long: `Manage the architecture knowledge base that powers intelligent
shell command routing and responses.`,
    }

    cmd.AddCommand(
        &cobra.Command{
            Use:   "sync",
            Short: "Synchronize architecture knowledge base",
            RunE: func(cmd *cobra.Command, args []string) error {
                fmt.Println("🔄 Synchronizing architecture knowledge base...")
                // Implementation for knowledge sync
                return nil
            },
        },
        &cobra.Command{
            Use:   "status",
            Short: "Show knowledge base status",
            RunE: func(cmd *cobra.Command, args []string) error {
                fmt.Println("📊 Architecture Knowledge Base Status:")
                // Implementation for knowledge status
                return nil
            },
        },
    )

    return cmd
}
```

### 4. Privacy-Aware Data Processing

```go
// internal/privacy/data_sanitizer.go
package privacy

import (
    "regexp"
    "strings"
)

type Level int
const (
    LowSensitivity Level = iota
    MediumSensitivity
    HighSensitivity
)

type DataSanitizer struct {
    config *SanitizerConfig
}

type SanitizerConfig struct {
    MaskAccountIDs      bool
    MaskResourceNames   bool
    MaskIPAddresses     bool
    MaskAccessKeys      bool
    MaskDomainNames     bool
    PreserveStructure   bool
    ReplacementStyle    string // "hash", "generic", "placeholder"
}

func (s *DataSanitizer) SanitizeForExternalAPI(data string, level Level) string {
    switch level {
    case LowSensitivity:
        return s.basicSanitization(data)
    case MediumSensitivity:
        return s.aggressiveSanitization(data)
    case HighSensitivity:
        return s.fullAnonymization(data)
    default:
        return data
    }
}

func (s *DataSanitizer) basicSanitization(data string) string {
    // Mask AWS account IDs
    accountIDRegex := regexp.MustCompile(`\b\d{12}\b`)
    data = accountIDRegex.ReplaceAllString(data, "ACCOUNT_ID")
    
    // Mask access keys
    accessKeyRegex := regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
    data = accessKeyRegex.ReplaceAllString(data, "ACCESS_KEY")
    
    // Mask secret keys
    secretKeyRegex := regexp.MustCompile(`[A-Za-z0-9/+=]{40}`)
    data = secretKeyRegex.ReplaceAllString(data, "SECRET_KEY")
    
    return data
}

func (s *DataSanitizer) aggressiveSanitization(data string) string {
    data = s.basicSanitization(data)
    
    // Mask IP addresses
    ipRegex := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
    data = ipRegex.ReplaceAllString(data, "IP_ADDRESS")
    
    // Mask domain names
    domainRegex := regexp.MustCompile(`\b[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*\.[a-zA-Z]{2,}\b`)
    data = domainRegex.ReplaceAllString(data, "DOMAIN")
    
    return data
}

func (s *DataSanitizer) fullAnonymization(data string) string {
    data = s.aggressiveSanitization(data)
    
    // Replace resource names with generic identifiers
    resourcePrefixes := []string{"prod-", "staging-", "dev-", "test-"}
    for _, prefix := range resourcePrefixes {
        data = strings.ReplaceAll(data, prefix, "ENV_")
    }
    
    return data
}
```

## Configuration Example

```yaml
# ~/.cloudai.yaml
model:
  primary_tier: "sagemaker"
  fallback_tiers: ["ollama", "bedrock", "openai"]
  
sagemaker:
  custom_endpoint: "cloudai-architecture-model-endpoint"
  training_schedule: "weekly"
  model_version: "v1.0.0"
  training_job_role: "arn:aws:iam::123456789012:role/SageMakerTrainingRole"
  training_data_bucket: "my-training-bucket"
  output_path: "s3://my-training-bucket/model-output/"
  
privacy:
  default_sanitization: "medium"
  sensitive_commands_local_only: true
  audit_logging: true
  
training:
  auto_training_enabled: true
  training_data_retention: "90d"
  incremental_updates: true
  collect_shell_patterns: true
  
shell_commands:
  context_aware: true
  learn_from_usage: true
  personalized_responses: true
  command_history_size: 1000
```

## Usage Examples

```bash
# Architecture-specific queries (routed to custom SageMaker model)
cloudai "what triggers the user lambda"
cloudai "database connections in staging"
cloudai "cost breakdown for microservices"

# Privacy-sensitive queries (routed to local Ollama)
cloudai "production database credentials" --local-only
cloudai "security group rules" --privacy strict

# Training and knowledge management
cloudai train --background
cloudai knowledge sync
cloudai knowledge status

# Model comparison and analysis
cloudai "lambda performance issues" --explain
cloudai model compare "cost optimization suggestions"
```

This implementation provides a comprehensive shell command enhancement system that learns from your specific AWS architecture and provides intelligent, context-aware responses while maintaining privacy and security.