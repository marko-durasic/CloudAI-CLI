package training

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
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

type InfrastructureState struct {
	Resources map[string]interface{} `json:"resources"`
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

type CostPattern struct {
	Service     string  `json:"service"`
	Resource    string  `json:"resource"`
	Cost        float64 `json:"cost"`
	Timestamp   time.Time `json:"timestamp"`
}

type TroubleshootingCase struct {
	Issue       string `json:"issue"`
	Resolution  string `json:"resolution"`
	Resources   []string `json:"resources"`
	Timestamp   time.Time `json:"timestamp"`
}

type TrainingExample struct {
	Input    string                 `json:"input"`
	Output   string                 `json:"output"`
	Metadata map[string]interface{} `json:"metadata"`
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
		AlgorithmSpecification: &types.AlgorithmSpecification{
			TrainingImage:     &t.config.TrainingImage,
			TrainingInputMode: types.TrainingInputModeFile,
		},
		InputDataConfig: []types.Channel{
			{
				ChannelName: aws.String("training"),
				DataSource: &types.DataSource{
					S3DataSource: &types.S3DataSource{
						S3DataType:         types.S3DataTypeS3Prefix,
						S3Uri:             &trainingDataPath,
						S3DataDistributionType: types.S3DataDistributionFullyReplicated,
					},
				},
				ContentType: aws.String("application/json"),
			},
		},
		OutputDataConfig: &types.OutputDataConfig{
			S3OutputPath: &t.config.OutputPath,
		},
		ResourceConfig: &types.ResourceConfig{
			InstanceType:   types.TrainingInstanceType(t.config.TrainingInstanceType),
			InstanceCount:  aws.Int32(int32(t.config.TrainingInstanceCount)),
			VolumeSizeInGB: aws.Int32(int32(t.config.VolumeSize)),
		},
		StoppingCondition: &types.StoppingCondition{
			MaxRuntimeInSeconds: aws.Int32(int32(t.config.MaxRuntimeInSeconds)),
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

func (t *SageMakerTrainer) formatCommandInput(command, context string) string {
	return fmt.Sprintf("Command: %s\nContext: %s", command, context)
}

func (t *SageMakerTrainer) monitorTrainingJob(ctx context.Context, jobName string) error {
	// Implementation for monitoring training job progress
	for {
		describeResult, err := t.sagemakerClient.DescribeTrainingJob(ctx, &sagemaker.DescribeTrainingJobInput{
			TrainingJobName: &jobName,
		})
		if err != nil {
			return fmt.Errorf("failed to describe training job: %w", err)
		}

		status := describeResult.TrainingJobStatus
		fmt.Printf("Training job status: %s\n", status)

		if status == types.TrainingJobStatusCompleted {
			fmt.Println("Training job completed successfully!")
			return nil
		} else if status == types.TrainingJobStatusFailed {
			return fmt.Errorf("training job failed: %s", *describeResult.FailureReason)
		} else if status == types.TrainingJobStatusStopped {
			return fmt.Errorf("training job was stopped")
		}

		// Wait before checking again
		time.Sleep(30 * time.Second)
	}
}

// NewSageMakerTrainer creates a new SageMakerTrainer with proper configuration
func NewSageMakerTrainer(sagemakerClient *sagemaker.Client, s3Client *s3.Client) *SageMakerTrainer {
	return &SageMakerTrainer{
		sagemakerClient: sagemakerClient,
		s3Client:        s3Client,
	}
}

// SetConfig sets the training configuration
func (t *SageMakerTrainer) SetConfig(config *TrainingConfig) {
	t.config = config
}

// NewTrainingConfig creates a properly configured TrainingConfig with all required fields initialized
// 
// IMPORTANT: This fixes the bug where only TrainingDataBucket and OutputPath were initialized.
// All other required fields are now properly initialized with reasonable defaults.
func NewTrainingConfig(trainingDataBucket, outputPath string) *TrainingConfig {
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	
	return &TrainingConfig{
		ModelName:              fmt.Sprintf("cloudai-architecture-model-%s", timestamp),
		TrainingJobName:        fmt.Sprintf("cloudai-training-job-%s", timestamp),
		RoleArn:               "arn:aws:iam::123456789012:role/SageMakerTrainingRole", // Default role ARN (should be configured)
		TrainingImage:         "382416733822.dkr.ecr.us-east-1.amazonaws.com/xgboost:latest", // Default XGBoost image
		TrainingInstanceType:  "ml.m5.large",
		TrainingInstanceCount: 1,
		VolumeSize:           30, // GB
		MaxRuntimeInSeconds:  3600, // 1 hour
		TrainingDataBucket:   trainingDataBucket,  // Just the bucket name (e.g., "my-training-bucket")
		OutputPath:           outputPath,          // Full S3 URI (e.g., "s3://my-training-bucket/model-output/")
		HyperParameters: map[string]string{
			"objective":        "reg:squarederror",
			"num_round":        "100",
			"max_depth":        "6",
			"eta":              "0.3",
			"subsample":        "1.0",
			"colsample_bytree": "1.0",
		},
	}
}

// NewTrainingConfigWithDefaults creates a TrainingConfig with custom parameters and reasonable defaults
func NewTrainingConfigWithDefaults(params TrainingConfigParams) *TrainingConfig {
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	
	config := &TrainingConfig{
		ModelName:              params.ModelName,
		TrainingJobName:        params.TrainingJobName,
		RoleArn:               params.RoleArn,
		TrainingImage:         params.TrainingImage,
		TrainingInstanceType:  params.TrainingInstanceType,
		TrainingInstanceCount: params.TrainingInstanceCount,
		VolumeSize:           params.VolumeSize,
		MaxRuntimeInSeconds:  params.MaxRuntimeInSeconds,
		TrainingDataBucket:   params.TrainingDataBucket,
		OutputPath:           params.OutputPath,
		HyperParameters:      params.HyperParameters,
	}
	
	// Apply defaults for any empty fields
	if config.ModelName == "" {
		config.ModelName = fmt.Sprintf("cloudai-architecture-model-%s", timestamp)
	}
	if config.TrainingJobName == "" {
		config.TrainingJobName = fmt.Sprintf("cloudai-training-job-%s", timestamp)
	}
	if config.RoleArn == "" {
		config.RoleArn = "arn:aws:iam::123456789012:role/SageMakerTrainingRole"
	}
	if config.TrainingImage == "" {
		config.TrainingImage = "382416733822.dkr.ecr.us-east-1.amazonaws.com/xgboost:latest"
	}
	if config.TrainingInstanceType == "" {
		config.TrainingInstanceType = "ml.m5.large"
	}
	if config.TrainingInstanceCount == 0 {
		config.TrainingInstanceCount = 1
	}
	if config.VolumeSize == 0 {
		config.VolumeSize = 30
	}
	if config.MaxRuntimeInSeconds == 0 {
		config.MaxRuntimeInSeconds = 3600
	}
	if config.HyperParameters == nil {
		config.HyperParameters = map[string]string{
			"objective":        "reg:squarederror",
			"num_round":        "100",
			"max_depth":        "6",
			"eta":              "0.3",
			"subsample":        "1.0",
			"colsample_bytree": "1.0",
		}
	}
	
	return config
}

// TrainingConfigParams holds optional parameters for training configuration
type TrainingConfigParams struct {
	ModelName              string
	TrainingJobName        string
	RoleArn               string
	TrainingImage         string
	TrainingInstanceType  string
	TrainingInstanceCount int
	VolumeSize           int
	MaxRuntimeInSeconds  int
	TrainingDataBucket   string
	OutputPath           string
	HyperParameters      map[string]string
}