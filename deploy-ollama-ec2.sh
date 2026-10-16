#!/bin/bash
# Deploy CloudAI-CLI Ollama Server on EC2

set -e

echo "🚀 CloudAI-CLI Ollama EC2 Deployment"
echo "=" + $(printf '%.0s=' {1..50})
echo ""

# Helper functions
check_quota() {
    local service_code="$1"
    local quota_code="$2"
    local quota_name="$3"
    
    QUOTA_VALUE=$(aws service-quotas get-service-quota \
        --service-code "$service_code" \
        --quota-code "$quota_code" \
        --query 'Quota.Value' \
        --output text 2>/dev/null || echo "0")
    
    echo "📊 Current $quota_name quota: $QUOTA_VALUE vCPUs"
    
    # Check if quota is greater than 0 (fallback if bc not available)
    if command -v bc >/dev/null 2>&1; then
        if [ "$(echo "$QUOTA_VALUE > 0" | bc)" = "1" ]; then
            return 0
        else
            return 1
        fi
    else
        # Fallback: simple string comparison for common cases
        if [ "$QUOTA_VALUE" != "0" ] && [ "$QUOTA_VALUE" != "0.0" ]; then
            return 0
        else
            return 1
        fi
    fi
}

request_quota_increase() {
    local service_code="$1"
    local quota_code="$2"
    local quota_name="$3"
    local desired_value="$4"
    
    echo ""
    echo "📝 Requesting quota increase for $quota_name..."
    echo "   Current: $QUOTA_VALUE vCPUs → Requested: $desired_value vCPUs"
    
    REQUEST_ID=$(aws service-quotas request-service-quota-increase \
        --service-code "$service_code" \
        --quota-code "$quota_code" \
        --desired-value "$desired_value" \
        --query 'RequestedQuota.Id' \
        --output text 2>/dev/null || echo "FAILED")
    
    if [ "$REQUEST_ID" != "FAILED" ]; then
        echo "✅ Quota increase request submitted (ID: $REQUEST_ID)"
        echo "⏱️  Typical approval time: 15 minutes - 24 hours"
        echo "📧 You'll receive an email notification when approved"
        return 0
    else
        echo "❌ Failed to submit quota request"
        return 1
    fi
}

show_alternatives() {
    echo ""
    echo "🔄 Alternative Options (No EC2 Instances Available):"
    echo ""
    if [ "$STANDARD_QUOTA_OK" = true ]; then
        echo "1. 🖥️  CPU-Only Instance (t3.medium - No GPU)"
        echo "   • Cost: ~$0.042/hour (~$1/day)"
        echo "   • Works with small models (phi3:mini)"
        echo "   • Uses standard quota (available)"
        echo ""
        echo "2. ☁️  Use AWS Bedrock (Remote AI)"
        echo "   • No EC2 instance needed"
        echo "   • Pay-per-request model"
        echo "   • Fast and reliable"
        echo ""
        echo "3. ⏳ Wait for Quota Approval"
        echo "   • Monitor GPU quota approval status"
        echo "   • Deploy GPU instance when ready"
        echo ""
        read -p "Choose option (1-3): " choice
        
        case $choice in
            1)
                deploy_cpu_instance
                ;;
            2)
                setup_bedrock_alternative
                ;;
            3)
                show_quota_monitoring
                ;;
            *)
                echo "❌ Invalid choice. Exiting..."
                exit 1
                ;;
        esac
    else
        echo "❌ **All EC2 quotas are 0** - No instances can be launched!"
        echo ""
        echo "1. ☁️  Use AWS Bedrock (Recommended)"
        echo "   • No EC2 instance needed"
        echo "   • No quotas required"
        echo "   • Pay-per-request model"
        echo "   • Fast and reliable"
        echo ""
        echo "2. 📝 Request Standard Instance Quota"
        echo "   • Request quota for t3.medium (cheaper option)"
        echo "   • Cost: ~$1/day when approved"
        echo "   • Works with small models"
        echo ""
        echo "3. 📝 Request GPU Instance Quota"
        echo "   • Request quota for g4dn.xlarge (best performance)"
        echo "   • Cost: ~$12.60/day when approved"
        echo "   • Works with all models"
        echo ""
        read -p "Choose option (1-3): " choice
        
        case $choice in
            1)
                setup_bedrock_alternative
                ;;
            2)
                echo ""
                echo "📝 Requesting Standard Instance quota (cheaper option)..."
                if request_quota_increase "ec2" "L-1216C47A" "Standard instances" "8"; then
                    echo ""
                    echo "✅ Standard quota requested! This enables t3.medium deployment."
                    show_quota_monitoring
                else
                    echo "❌ Request failed. Try option 1 (Bedrock) instead."
                fi
                ;;
            3)
                echo ""
                echo "📝 Requesting GPU Instance quota (best performance)..."
                if request_quota_increase "ec2" "L-DB2E81BA" "GPU instances" "4"; then
                    echo ""
                    echo "✅ GPU quota requested! This enables g4dn.xlarge deployment."
                    show_quota_monitoring
                else
                    echo "❌ Request failed. Try option 1 (Bedrock) instead."
                fi
                ;;
            *)
                echo "❌ Invalid choice. Exiting..."
                exit 1
                ;;
        esac
    fi
}

deploy_cpu_instance() {
    echo ""
    echo "🖥️  Deploying CPU-only instance..."
    
    # Double-check standard quota before attempting
    if ! check_quota "ec2" "L-1216C47A" "Standard instances"; then
        echo "❌ Standard instance quota is still 0. Cannot deploy t3.medium."
        echo "💡 Request standard quota first, then run this script again."
        exit 1
    fi
    
    aws cloudformation deploy \
        --template-file ec2-ollama-stack.yaml \
        --stack-name cloudai-ollama-server-cpu \
        --parameter-overrides \
            KeyPairName=$KEY_NAME \
            InstanceType=t3.medium \
            AllowedCIDR="$CLOUDAI_ALLOWED_CIDR" \
        --capabilities CAPABILITY_IAM \
        --region $REGION
    
    if [ $? -eq 0 ]; then
        echo "✅ CPU instance deployed successfully!"
        echo "⚠️  Note: This will be slower than GPU instances"
        echo "📝 Only small models (phi3:mini) recommended"
        get_deployment_outputs "cloudai-ollama-server-cpu"
    else
        echo "❌ CPU deployment failed. Likely quota or other AWS issues."
        echo "💡 Try AWS Bedrock as an alternative (no EC2 needed)."
        exit 1
    fi
}

setup_bedrock_alternative() {
    echo ""
    echo "☁️  Setting up AWS Bedrock alternative..."
    echo ""
    echo "✅ No EC2 instance needed!"
    echo "🔧 To configure CloudAI-CLI for Bedrock:"
    echo ""
    echo "   ./cloudai setup-interactive"
    echo "   # Choose option 2 (Remote models)"
    echo "   # Follow the prompts"
    echo ""
    echo "💡 Benefits:"
    echo "   • No infrastructure to manage"
    echo "   • Fast inference"
    echo "   • Pay only for what you use"
    echo "   • No quota issues"
    echo ""
    exit 0
}

show_quota_monitoring() {
    echo ""
    echo "📊 Quota Monitoring Commands:"
    echo ""
    echo "# Check pending requests:"
    echo "aws service-quotas list-requested-service-quota-change-history \\"
    echo "  --service-code ec2 \\"
    echo "  --query 'RequestedQuotas[?Status==\`PENDING\`].[QuotaName,Status]' \\"
    echo "  --output table"
    echo ""
    echo "# Check current quotas:"
    echo "aws service-quotas get-service-quota \\"
    echo "  --service-code ec2 \\"
    echo "  --quota-code L-DB2E81BA \\"
    echo "  --query 'Quota.Value'"
    echo ""
    echo "💡 Run this script again once quotas are approved!"
    exit 0
}

get_deployment_outputs() {
    local stack_name="$1"
    
    echo ""
    echo "📋 Getting stack outputs..."
    OUTPUTS=$(aws cloudformation describe-stacks \
        --stack-name "$stack_name" \
        --query 'Stacks[0].Outputs' \
        --region $REGION)
    
    PUBLIC_IP=$(echo $OUTPUTS | jq -r '.[] | select(.OutputKey=="PublicIP") | .OutputValue')
    OLLAMA_URL=$(echo $OUTPUTS | jq -r '.[] | select(.OutputKey=="OllamaURL") | .OutputValue')
    SSH_CMD=$(echo "$OUTPUTS" | jq -r '.[] | select(.OutputKey=="SSHCommand") | .OutputValue')
    SSH_TUNNEL=$(echo "$OUTPUTS" | jq -r '.[] | select(.OutputKey=="SSHTunnel") | .OutputValue')
    
    echo "🎉 Deployment Complete!"
    echo "=" + $(printf '%.0s=' {1..50})
    echo ""
    echo "🔒 Local Ollama URL after SSH tunnel: $OLLAMA_URL"
    echo "🔑 SSH Command: $SSH_CMD"
    echo "🔐 On your trusted workstation run: $SSH_TUNNEL"
    echo ""
    echo "⏳ The instance is starting up and installing Ollama..."
    echo "   This takes about 5-10 minutes for full setup."
    echo ""
    echo "🔧 To use with CloudAI-CLI:"
    echo "   export OLLAMA_URL=$OLLAMA_URL"
    echo "   cloudai setup-interactive  # Choose option 1 (Local models)"
    echo ""
    echo "   Keep the SSH tunnel running while using CloudAI-CLI."
    echo "🧪 Test the local tunnel when ready:"
    echo "   curl $OLLAMA_URL/api/tags"
    echo ""
    echo "🗑️  To delete everything:"
    echo "   aws cloudformation delete-stack --stack-name $stack_name --region $REGION"
}

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

# Require a single trusted administrator IP; never deploy internet-accessible Ollama/SSH.
if [[ ! "${CLOUDAI_ALLOWED_CIDR:-}" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}/32$ ]]; then
    echo "❌ Set CLOUDAI_ALLOWED_CIDR to your workstation public IPv4/32."
    echo '   Example: export CLOUDAI_ALLOWED_CIDR="203.0.113.10/32" (replace with your actual IP)'
    echo "   Ollama is reached only by SSH tunnel, never via a public API endpoint."
    exit 1
fi

# Check quotas before deployment
echo ""
echo "🔍 Checking AWS quotas..."

# Check GPU instance quota
if ! check_quota "ec2" "L-DB2E81BA" "GPU instances (G and VT)"; then
    GPU_QUOTA_OK=false
    echo "⚠️  GPU instance quota is 0 - cannot deploy g4dn.xlarge"
else
    GPU_QUOTA_OK=true
    echo "✅ GPU instance quota available"
fi

# Check standard instance quota
if ! check_quota "ec2" "L-1216C47A" "Standard instances"; then
    STANDARD_QUOTA_OK=false
    echo "⚠️  Standard instance quota is 0 - limited options"
else
    STANDARD_QUOTA_OK=true
    echo "✅ Standard instance quota available"
fi

# Handle quota issues interactively
if [ "$GPU_QUOTA_OK" = false ]; then
    echo ""
    echo "❌ Cannot deploy GPU instance due to quota limitations"
    echo ""
    echo "🤔 What would you like to do?"
    echo ""
    echo "1. 📝 Request quota increase (recommended)"
    echo "   • Increases GPU instance limit to 4 vCPUs"
    echo "   • Usually approved within 2-24 hours"
    echo "   • Enables high-performance AI inference"
    echo ""
    echo "2. 🔄 See alternative options"
    echo "   • CPU-only deployment"
    echo "   • Use AWS Bedrock instead"
    echo "   • Other alternatives"
    echo ""
    read -p "Choose option (1 or 2): " quota_choice
    
    case $quota_choice in
        1)
            echo ""
            echo "📝 Requesting GPU quota increase..."
            echo ""
            echo "ℹ️  This will:"
            echo "   • Request 4 vCPUs for GPU instances (enough for g4dn.xlarge)"
            echo "   • Submit the request to AWS automatically"
            echo "   • You'll get email notification when approved"
            echo "   • No charges until you actually launch instances"
            echo ""
            read -p "Proceed with quota request? (y/N): " confirm
            
            if [[ $confirm =~ ^[Yy]$ ]]; then
                if request_quota_increase "ec2" "L-DB2E81BA" "GPU instances" "4"; then
                    echo ""
                    echo "🎯 Next steps:"
                    echo "1. Wait for quota approval (usually 2-24 hours)"
                    echo "2. Run this script again: ./deploy-ollama-ec2.sh"
                    echo "3. Or monitor status with quota monitoring commands"
                    
                    show_quota_monitoring
                else
                    echo ""
                    echo "❌ Quota request failed. Showing alternatives..."
                    show_alternatives
                fi
            else
                show_alternatives
            fi
            ;;
        2)
            show_alternatives
            ;;
        *)
            echo "❌ Invalid choice. Exiting..."
            exit 1
            ;;
    esac
    exit 0
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
echo "🚀 Using g4dn.xlarge instance (GPU-enabled)"

aws cloudformation deploy \
    --template-file ec2-ollama-stack.yaml \
    --stack-name $STACK_NAME \
    --parameter-overrides \
        KeyPairName=$KEY_NAME \
        InstanceType=g4dn.xlarge \
        AllowedCIDR="$CLOUDAI_ALLOWED_CIDR" \
    --capabilities CAPABILITY_IAM \
    --region $REGION

if [ $? -eq 0 ]; then
    echo "✅ Stack deployed successfully!"
    echo "💰 Cost: ~$0.526/hour (~$12.60/day for 24/7)"
    echo "💡 Stop instance when not in use to save costs!"
    
    get_deployment_outputs "$STACK_NAME"
else
    echo "❌ Stack deployment failed!"
    echo ""
    echo "🔍 This might be due to:"
    echo "   • Network/connectivity issues"
    echo "   • CloudFormation template errors"
    echo "   • Other AWS service limits"
    echo ""
    echo "📋 Check CloudFormation events:"
    echo "   aws cloudformation describe-stack-events --stack-name $STACK_NAME"
    echo ""
    echo "🔄 Would you like to try alternatives?"
    read -p "Show alternatives? (y/N): " show_alt
    
    if [[ $show_alt =~ ^[Yy]$ ]]; then
        show_alternatives
    fi
    
    exit 1
fi 