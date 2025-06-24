#!/bin/bash
# Deploy CloudAI-CLI Ollama Server on EC2

set -e

echo "🚀 CloudAI-CLI Ollama EC2 Deployment"
echo "=" + $(printf '%.0s=' {1..50})
echo ""

# Check if AWS CLI is configured
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo "❌ AWS CLI not configured. Run: aws configure"
    exit 1
fi

# Get current region
REGION=$(aws configure get region)
if [ -z "$REGION" ]; then
    REGION="us-west-2"
    echo "🌍 No region set, using default: $REGION"
else
    echo "🌍 Using region: $REGION"
fi

# Check for existing key pairs
echo ""
echo "🔑 Checking for SSH key pairs..."
KEY_PAIRS=$(aws ec2 describe-key-pairs --query 'KeyPairs[].KeyName' --output text 2>/dev/null || echo "")

if [ -z "$KEY_PAIRS" ]; then
    echo "❌ No SSH key pairs found. Creating one..."
    KEY_NAME="cloudai-ollama-key"
    aws ec2 create-key-pair --key-name $KEY_NAME --query 'KeyMaterial' --output text > $KEY_NAME.pem
    chmod 400 $KEY_NAME.pem
    echo "✅ Created key pair: $KEY_NAME.pem"
else
    KEY_NAME=$(echo $KEY_PAIRS | awk '{print $1}')
    echo "✅ Using existing key pair: $KEY_NAME"
fi

# Deploy CloudFormation stack
STACK_NAME="cloudai-ollama-server"
echo ""
echo "☁️  Deploying CloudFormation stack: $STACK_NAME"

aws cloudformation deploy \
    --template-file ec2-ollama-stack.yaml \
    --stack-name $STACK_NAME \
    --parameter-overrides \
        KeyPairName=$KEY_NAME \
        InstanceType=g4dn.xlarge \
    --capabilities CAPABILITY_IAM \
    --region $REGION

if [ $? -eq 0 ]; then
    echo "✅ Stack deployed successfully!"
    
    # Get outputs
    echo ""
    echo "📋 Getting stack outputs..."
    OUTPUTS=$(aws cloudformation describe-stacks \
        --stack-name $STACK_NAME \
        --query 'Stacks[0].Outputs' \
        --region $REGION)
    
    PUBLIC_IP=$(echo $OUTPUTS | jq -r '.[] | select(.OutputKey=="PublicIP") | .OutputValue')
    OLLAMA_URL=$(echo $OUTPUTS | jq -r '.[] | select(.OutputKey=="OllamaURL") | .OutputValue')
    SSH_CMD=$(echo $OUTPUTS | jq -r '.[] | select(.OutputKey=="SSHCommand") | .OutputValue')
    
    echo "🎉 Deployment Complete!"
    echo "=" + $(printf '%.0s=' {1..50})
    echo ""
    echo "🌐 Ollama API URL: $OLLAMA_URL"
    echo "🔗 SSH Command: $SSH_CMD"
    echo ""
    echo "⏳ The instance is starting up and installing Ollama..."
    echo "   This takes about 5-10 minutes for full setup."
    echo ""
    echo "🔧 To use with CloudAI-CLI:"
    echo "   export OLLAMA_URL=$OLLAMA_URL"
    echo "   cloudai setup-interactive  # Choose option 1 (Local models)"
    echo ""
    echo "🧪 Test when ready:"
    echo "   curl $OLLAMA_URL/api/tags"
    echo ""
    echo "💰 Cost: ~$0.526/hour (~$12.60/day for 24/7)"
    echo "💡 Stop instance when not in use to save costs!"
    echo ""
    echo "🗑️  To delete everything:"
    echo "   aws cloudformation delete-stack --stack-name $STACK_NAME --region $REGION"
    
else
    echo "❌ Stack deployment failed!"
    exit 1
fi 