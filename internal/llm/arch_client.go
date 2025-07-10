package llm

import (
    "fmt"
    "os"
)

// NewArchClientFromEnv attempts to construct a specialised architecture model
// client (usually a SageMaker endpoint fine-tuned on your infra docs) based on
// three environment variables:
//   CLOUDAI_ARCH_ENDPOINT   – SageMaker endpoint name
//   CLOUDAI_ARCH_REGION     – AWS region (default us-east-1)
//   CLOUDAI_ARCH_MODEL_ID   – Optional metadata only, used for cost calc & logs
//
// If CLOUDAI_ARCH_ENDPOINT is not set the function returns (nil, nil) so callers
// can treat the absence of a specialised model as non-fatal.
func NewArchClientFromEnv() (*Client, error) {
    endpoint := os.Getenv("CLOUDAI_ARCH_ENDPOINT")
    if endpoint == "" {
        return nil, nil
    }

    region := os.Getenv("CLOUDAI_ARCH_REGION")
    if region == "" {
        region = "us-east-1"
    }
    modelID := os.Getenv("CLOUDAI_ARCH_MODEL_ID")
    if modelID == "" {
        modelID = "arch-bot"
    }

    cfg := &AWSModelConfig{
        Type:         AWSModelSageMaker,
        ModelID:      modelID,
        EndpointName: endpoint,
        Region:       region,
        MaxTokens:    2048,
        Temperature:  0.1,
    }

    awsClient, err := NewAWSClient(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create architecture SageMaker client: %w", err)
    }

    // Use a relaxed daily budget – specialised model, likely infrequent
    cm := NewCostManager(2.0) // $2/day default

    return &Client{
        useAWS:      true,
        awsClient:   awsClient,
        costManager: cm,
    }, nil
}