package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps AWS service clients
type Client struct {
	APIGateway   *apigateway.Client
	Lambda       *lambda.Client
	S3           *s3.Client
	CostExplorer *costexplorer.Client
}

// NewClient creates a new AWS client with all required services
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		APIGateway:   apigateway.NewFromConfig(cfg),
		Lambda:       lambda.NewFromConfig(cfg),
		S3:           s3.NewFromConfig(cfg),
		CostExplorer: costexplorer.NewFromConfig(cfg),
	}, nil
}
