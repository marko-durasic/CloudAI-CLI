> **Security update (2026-10-16):** Ollama has no authentication and must not be publicly exposed. The EC2 template now requires an explicit administrator IPv4 /32 for SSH and binds Ollama to localhost. Access it using an SSH tunnel.

### Secure EC2 deployment

From your administrator workstation, determine its public IPv4 and export the allowlist:

```bash
export CLOUDAI_ALLOWED_CIDR="$(curl -fsS https://checkip.amazonaws.com | tr -d '\n')/32"
./deploy-ollama-ec2.sh
```

Verify the discovered address belongs to the intended workstation or VPN. The script refuses to deploy without a /32. Updating your IP later requires a deliberate SSH security-group allowlist change, never 0.0.0.0/0.

After deployment, run the tunnel from the administrator workstation:

```bash
ssh -N -L 11434:127.0.0.1:11434 -i YOUR_KEY.pem ubuntu@YOUR_EC2_PUBLIC_IP
```

Use the local endpoint from a second workstation terminal:

```bash
export OLLAMA_URL=http://127.0.0.1:11434
curl -fsS "$OLLAMA_URL/api/tags"
```

**Existing AWS deployments:** A source-code fix cannot close already-deployed security-group ingress or reconfigure a running Ollama instance. Inspect active stacks and security groups independently. Do not expose port 11434 to the internet.

---
# CloudAI-CLI EC2 Deployment Guide

## 🚀 **Smart Deployment with Quota Handling**

The CloudAI-CLI now includes intelligent EC2 deployment that automatically handles AWS quota limitations and provides multiple deployment options.

## 📋 **Quick Start**

```bash
# Simple deployment (handles quotas automatically)
./deploy-ollama-ec2.sh

# Check quotas first (optional)
./check-quotas.sh
```

## 🔍 **How It Works**

### **1. Quota Detection**
The script automatically checks your AWS quotas before deployment:
- **GPU Instances** (g4dn.xlarge for fast inference)
- **Standard Instances** (t3.medium for CPU-only)
- **Current usage and limits**

### **2. Interactive Quota Handling**
If quotas are insufficient, you get clear options:

```
❌ Cannot deploy GPU instance due to quota limitations

🤔 What would you like to do?

1. 📝 Request quota increase (recommended)
   • Increases GPU instance limit to 4 vCPUs
   • Usually approved within 2-24 hours
   • Enables high-performance AI inference

2. 🔄 See alternative options
   • CPU-only deployment
   • Use AWS Bedrock instead
   • Other alternatives
```

### **3. Automatic Quota Requests**
The script can submit quota requests for you:
- Explains what the request entails
- Shows expected approval timeline
- Provides monitoring commands
- No charges until instances launch

## 🎯 **Deployment Options**

### **Option 1: GPU Instance (Recommended)**
- **Instance**: g4dn.xlarge (NVIDIA T4 GPU)
- **Cost**: ~$0.526/hour (~$12.60/day)
- **Performance**: Fast inference with larger models
- **Requirements**: GPU quota (4 vCPUs)

### **Option 2: CPU Instance (Budget)**
- **Instance**: t3.medium (2 vCPUs, 4GB RAM)
- **Cost**: ~$0.042/hour (~$1/day)
- **Performance**: Slower, works with small models
- **Requirements**: Standard quota (2 vCPUs)

### **Option 3: AWS Bedrock (Serverless)**
- **Infrastructure**: None (serverless)
- **Cost**: Pay-per-request ($0.001-0.01 per request)
- **Performance**: Fast, managed service
- **Requirements**: No quotas needed

## 🛠️ **Scripts Overview**

### **deploy-ollama-ec2.sh**
Enhanced deployment script with:
- ✅ Automatic quota checking
- ✅ Interactive quota requests
- ✅ Multiple deployment options
- ✅ Fallback alternatives
- ✅ Clear error handling

### **check-quotas.sh**
Standalone quota monitoring:
```bash
./check-quotas.sh

# Output:
🔍 CloudAI-CLI AWS Quota Checker
=================================

📊 Current Quotas:
🔴 GPU Instances: 0.0 vCPUs (BLOCKED)
🟢 Standard Instances: 5.0 vCPUs (OK)

📝 Pending Quota Requests:
✅ No pending quota requests

🎯 Recommendations:
🚀 Request GPU quota increase for g4dn.xlarge
```

## 🔄 **Quota Management Workflow**

### **1. Initial Check**
```bash
./check-quotas.sh
```

### **2. Request Quota Increase**
```bash
# Automatic (via deployment script)
./deploy-ollama-ec2.sh

# Manual (if needed)
aws service-quotas request-service-quota-increase \
  --service-code ec2 \
  --quota-code L-DB2E81BA \
  --desired-value 4
```

### **3. Monitor Status**
```bash
# Check approval status
aws service-quotas list-requested-service-quota-change-history \
  --service-code ec2 \
  --query 'RequestedQuotas[?Status==`PENDING`].[QuotaName,Status]' \
  --output table
```

### **4. Deploy When Ready**
```bash
./deploy-ollama-ec2.sh  # Will detect approved quotas
```

## 🎯 **Use Cases**

### **For Infrastructure Questions**
Once deployed, use your EC2 Ollama for:
```bash
export OLLAMA_URL=http://YOUR_EC2_IP:11434
./cloudai setup-interactive  # Choose Local models

# Ask infrastructure questions
./cloudai "What Lambda functions do I have?"
./cloudai "Explain my VPC architecture"
./cloudai "Optimize my S3 bucket costs"
```

### **For Development**
- Test different model sizes
- Experiment with fine-tuning
- Develop AI-powered infrastructure tools
- Private, controlled AI environment

## 🔧 **Troubleshooting**

### **Quota Request Denied**
```bash
# Check denial reason
aws service-quotas list-requested-service-quota-change-history \
  --service-code ec2

# Contact AWS Support if needed
# Or try CPU-only deployment
```

### **CloudFormation Failures**
```bash
# Check events
aws cloudformation describe-stack-events --stack-name cloudai-ollama-server

# Clean up and retry
aws cloudformation delete-stack --stack-name cloudai-ollama-server
./deploy-ollama-ec2.sh
```

### **Instance Not Ready**
```bash
# Check instance status
aws ec2 describe-instances --filters "Name=tag:Name,Values=cloudai-ollama-server"

# SSH and check logs
ssh -i cloudai-ollama-key.pem ubuntu@YOUR_IP
sudo journalctl -u ollama -f
```

## 💰 **Cost Management**

### **Stop Instance When Not Using**
```bash
# Stop (preserves data, stops charges)
aws ec2 stop-instances --instance-ids YOUR_INSTANCE_ID

# Start when needed
aws ec2 start-instances --instance-ids YOUR_INSTANCE_ID
```

### **Terminate When Done**
```bash
# Complete cleanup
aws cloudformation delete-stack --stack-name cloudai-ollama-server
```

## 🌍 **Multi-Region Support**
The scripts work in any AWS region:
```bash
AWS_DEFAULT_REGION=us-west-2 ./deploy-ollama-ec2.sh
AWS_DEFAULT_REGION=eu-west-1 ./deploy-ollama-ec2.sh
```

## 🔒 **Security Notes**
- Uses IAM roles (no hardcoded credentials)
- Security groups restrict access to necessary ports
- SSH key management included
- Private networking supported

## 📧 **Support**
- AWS quota requests usually approve within 24 hours
- New accounts may take longer (up to 48 hours)
- Contact AWS Support for urgent needs or denials 