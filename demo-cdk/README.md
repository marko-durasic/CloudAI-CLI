# CloudAI-CLI Demo CDK Stack

This AWS CDK stack creates a minimal API Gateway + Lambda setup for testing CloudAI-CLI.

## Prerequisites
- [AWS CDK](https://docs.aws.amazon.com/cdk/latest/guide/getting_started.html) installed (`npm install -g aws-cdk`)
- Node.js 18+
- AWS credentials configured (profile or env vars)

## Quickstart

```sh
# Clone this repo and enter the demo-cdk directory
cd demo-cdk
npm install
cdk bootstrap
cdk deploy
```

## What it creates
- Lambda function: `cloudai-demo-hello`
- API Gateway: `cloudai-demo-api` with `/hello` resource and `GET` method

## Test with CloudAI-CLI
After deployment, run:
```sh
go run ../cmd/cloudai/main.go "Which Lambda handles GET /hello on cloudai-demo-api?"
```

## Cleanup
```sh
cdk destroy
``` 