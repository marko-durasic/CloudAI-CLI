#!/bin/bash
# Check AWS EC2 Quotas for CloudAI-CLI

echo "🔍 CloudAI-CLI AWS Quota Checker"
echo "================================="
echo ""

# Check if AWS CLI is configured
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo "❌ AWS CLI not configured. Run: aws configure"
    exit 1
fi

REGION=$(aws configure get region)
echo "🌍 Region: $REGION"
echo ""

check_quota() {
    local service_code="$1"
    local quota_code="$2"
    local quota_name="$3"
    
    echo "Checking $quota_name..."
    QUOTA_VALUE=$(aws service-quotas get-service-quota \
        --service-code "$service_code" \
        --quota-code "$quota_code" \
        --query 'Quota.Value' \
        --output text 2>/dev/null || echo "Error")
    
    if [ "$QUOTA_VALUE" = "Error" ]; then
        echo "❌ Could not check $quota_name quota"
    elif [ "$QUOTA_VALUE" = "0" ] || [ "$QUOTA_VALUE" = "0.0" ]; then
        echo "🔴 $quota_name: $QUOTA_VALUE vCPUs (BLOCKED)"
    else
        echo "🟢 $quota_name: $QUOTA_VALUE vCPUs (OK)"
    fi
}

echo "📊 Current Quotas:"
echo "=================="
check_quota "ec2" "L-DB2E81BA" "GPU Instances (G and VT)"
check_quota "ec2" "L-1216C47A" "Standard Instances (A,C,D,H,I,M,R,T,Z)"
check_quota "ec2" "L-34B43A08" "Spot Instances"

echo ""
echo "📝 Pending Quota Requests:"
echo "=========================="

PENDING_REQUESTS=$(aws service-quotas list-requested-service-quota-change-history \
    --service-code ec2 \
    --query 'RequestedQuotas[?Status==`PENDING`]' \
    --output text 2>/dev/null)

if [ -z "$PENDING_REQUESTS" ] || [ "$PENDING_REQUESTS" = "None" ]; then
    echo "✅ No pending quota requests"
else
    aws service-quotas list-requested-service-quota-change-history \
        --service-code ec2 \
        --query 'RequestedQuotas[?Status==`PENDING`].[QuotaName,DesiredValue,Status,Created]' \
        --output table
fi

echo ""
echo "🎯 Recommendations:"
echo "==================="

# Check GPU quota
GPU_QUOTA=$(aws service-quotas get-service-quota \
    --service-code ec2 \
    --quota-code L-DB2E81BA \
    --query 'Quota.Value' \
    --output text 2>/dev/null || echo "0")

if [ "$GPU_QUOTA" = "0" ] || [ "$GPU_QUOTA" = "0.0" ]; then
    echo "🚀 For GPU instances (g4dn.xlarge): Request quota increase"
    echo "   aws service-quotas request-service-quota-increase \\"
    echo "     --service-code ec2 \\"
    echo "     --quota-code L-DB2E81BA \\"
    echo "     --desired-value 4"
    echo ""
else
    echo "✅ GPU instances available - can deploy g4dn.xlarge"
fi

# Check standard quota
STANDARD_QUOTA=$(aws service-quotas get-service-quota \
    --service-code ec2 \
    --quota-code L-1216C47A \
    --query 'Quota.Value' \
    --output text 2>/dev/null || echo "0")

if [ "$STANDARD_QUOTA" = "0" ] || [ "$STANDARD_QUOTA" = "0.0" ]; then
    echo "🖥️  For CPU instances (t3.medium): Request quota increase"
    echo "   aws service-quotas request-service-quota-increase \\"
    echo "     --service-code ec2 \\"
    echo "     --quota-code L-1216C47A \\"
    echo "     --desired-value 8"
    echo ""
else
    echo "✅ Standard instances available - can deploy t3.medium"
fi

echo "🔄 Run this script anytime: ./check-quotas.sh"
echo "🚀 Deploy when ready: ./deploy-ollama-ec2.sh" 