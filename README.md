# CloudAI-CLI

CloudAI-CLI is a single-binary Go tool that turns plain-English prompts into AWS SDK calls, revealing live infrastructure topology and high-level cost drivers.

## 🚀 Quick Start

### 1. Install CloudAI-CLI Globally
```bash
# Clone and install
git clone https://github.com/marko-durasic/CloudAI-CLI.git
cd CloudAI-CLI
go install ./cmd/cloudai

# Verify installation
cloudai --help
```

### 2. Interactive Setup (NEW!)
```bash
cloudai setup-interactive
```

Choose from **6 deployment options**:
- **Option 1**: Local Ollama (FREE, private)
- **Option 2**: EC2 Ollama (GPU-powered) 
- **Option 3**: SageMaker (Fine-tuned)
- **Option 4**: AWS Bedrock (Managed)
- **Option 5**: Privacy Remote API (Hybrid)
- **Option 6**: Privacy CLI Tools (Hybrid)

### 3. Scan Your Infrastructure
Navigate to your project directory and scan your Infrastructure as Code files:

```bash
# Go to your project directory first
cd /path/to/your/aws-project

# For CDK projects, compile first (creates cdk.out/)
npx cdk synth          # Required for CDK projects

# Scan IaC files (CDK, Terraform, CloudFormation, etc.)
cloudai scan
```

**What it scans:**
- **AWS CDK**: Compiled templates in `cdk.out/` directory (run `cdk synth` first)
- **Terraform**: `*.tf` files (coming soon)
- **CloudFormation**: `*.yaml`, `*.yml`, `*.json` template files (coming soon)
- **Serverless framework**: `serverless.yml` (coming soon)
- **AWS SAM**: `template.yaml` (coming soon)

**⚠️ Important for CDK users:**
- CloudAI scans **compiled CloudFormation templates**, not TypeScript source
- You must run `cdk synth` to generate the `cdk.out/` directory first
- Just having deployed infrastructure isn't enough - need local compilation

### 4. Ask Questions
From the same project directory, ask questions about your infrastructure:

```bash
cloudai "What AWS services am I using?"
cloudai "List my Lambda functions"
cloudai "How can I reduce costs?"
cloudai "Any security risks in my setup?"
cloudai "What's the purpose of each Lambda function?"
```

---

## 🧪 Testing Guide

### Test 1: Fast Setup Interface
**What to test**: The new lightning-fast setup experience

```bash
# Install globally first (if not done)
go install ./cmd/cloudai

# Run setup from any directory
cloudai setup-interactive
```

**Expected**: Should display 6 options in compact format immediately (no delay)

**Try**:
- Type `h` → Should show detailed descriptions
- Type `1-6` → Should start respective setup flows
- Invalid input → Should show helpful error message

---

### Test 2: Local Ollama (Option 1)
**Prerequisites**: Install Ollama first
```bash
# Install Ollama from https://ollama.com/
ollama serve
ollama pull llama3.2:3b
```

**Test steps**:
```bash
cloudai setup-interactive
# Choose option 1
# Follow the prompts
```

**Expected**:
- Detects Ollama automatically
- Shows available models
- Configures and tests connection
- Displays demo commands

**Test the setup**:
```bash
# Navigate to a project with IaC files (like demo-cdk/)
cd demo-cdk
# Note: demo-cdk already has cdk.out/ - for your own projects run: cdk synth
cloudai scan  # Scans compiled CloudFormation templates in cdk.out/
cloudai "What Lambda functions do I have?"
cloudai "Explain the architecture"
```

---

### Test 3: EC2 Deployment (Option 2)
**Prerequisites**: AWS credentials configured
```bash
aws configure  # If not already done
```

**Test steps**:
```bash
cloudai setup-interactive
# Choose option 2
# Follow the prompts
```

**Expected**:
- Checks AWS credentials
- Explains quota requirements
- Guides to deployment script
- Shows next steps

**Test deployment** (if quotas approved):
```bash
./deploy-ollama-ec2.sh
# Follow quota handling prompts
```

---

### Test 4: Privacy Options (Options 5 & 6)
**Test Option 5** (Privacy Remote API):
```bash
cloudai setup-interactive
# Choose option 5
# You'll need: Local Ollama + API key (OpenAI/Anthropic)
```

**Test Option 6** (Privacy CLI Tools):
```bash
cloudai setup-interactive
# Choose option 6  
# You'll need: Local Ollama + CLI tool (Gemini/Bard)
```

**Expected**:
- Requires local Ollama first
- Guides through API/CLI setup
- Explains privacy protection
- Saves hybrid configuration

---

### Test 5: AWS Options (Options 3 & 4)
**Test Option 3** (SageMaker):
```bash
cloudai setup-interactive
# Choose option 3
```

**Test Option 4** (Bedrock):
```bash
cloudai setup-interactive
# Choose option 4
```

**Expected**:
- Checks AWS credentials
- Explains requirements
- Saves configuration
- Provides next steps

---

### Test 6: Help and Error Handling
**Test help feature**:
```bash
cloudai setup-interactive
# Type 'h' → Should show detailed options
# Type 'help' → Should also work
```

**Test error handling**:
```bash
cloudai setup-interactive
# Type invalid input like 'abc' or '99'
# Should show helpful error message
```

---

### Test 7: Configuration Persistence
**Test config saving**:
```bash
# Complete any setup option
cloudai setup-interactive
# Choose any option and complete setup

# Check config was saved
cat ~/.cloudai.yaml

# Test config is used
cloudai scan  # Should use saved config
```

---

### Test 8: Infrastructure Analysis
**Test with demo project**:
```bash
# Navigate to demo project with CDK files
cd demo-cdk
# (demo-cdk already has cdk.out/ pre-built)
cloudai scan  # Scans the compiled CloudFormation templates

# Test various questions about the scanned infrastructure
cloudai "What services are in this project?"
cloudai "How many Lambda functions?"
cloudai "What's the S3 bucket for?"
cloudai "Explain the Step Function workflow"
```

**Test with real AWS account**:
```bash
# Navigate to your project directory with IaC files
cd /path/to/your/project  

# For CDK projects, compile first
cdk synth                 # Creates cdk.out/ with CloudFormation templates

# Scan your infrastructure
cloudai scan              # Scans compiled templates (not source files)
cloudai "What's my most expensive service?"
cloudai "Any security risks in my setup?"
```

---

## 🎯 What to Look For

### ✅ Good Experience
- **Fast setup** (no delays)
- **Clear options** (easy to understand)
- **Helpful guidance** (explains next steps)
- **Error recovery** (helpful error messages)
- **Working Q&A** (accurate infrastructure analysis)

### ❌ Issues to Report
- **Slow startup** (takes >2 seconds to show options)
- **Confusing UI** (unclear what options do)
- **Setup failures** (crashes or hangs)
- **Wrong answers** (inaccurate infrastructure analysis)
- **Config issues** (settings not saved/loaded)

---

## 🔧 Troubleshooting

### Setup Issues
```bash
# If setup is slow
cloudai --help  # Should be fast - if not, it's a config issue

# If Ollama not detected
curl http://localhost:11434/api/tags  # Should return model list

# If AWS issues
aws sts get-caller-identity  # Should return your account info
```

### Analysis Issues
```bash
# If scan finds no files
cloudai scan --verbose  # Shows what files it's looking for
# Make sure you're in a directory with IaC files (*.tf, *.ts, *.yaml, etc.)

# If no cache found
ls .cloudai/  # Should contain cache.json after successful scan

# If wrong answers or incomplete results
cat .cloudai/cache.json | jq .  # Check what infrastructure was scanned

# If "No infrastructure found" error
ls cdk.out/ 2>/dev/null              # CDK: Check if cdk.out/ exists
ls *.tf *.yaml *.yml 2>/dev/null     # Check if other IaC files exist

# For CDK projects specifically:
cdk synth                            # Generate cdk.out/ directory first
cloudai scan                         # Then try scanning again
```

---

## 📋 Test Checklist

- [ ] Fast setup display (<2 seconds)
- [ ] All 6 options work
- [ ] Help feature ('h') works  
- [ ] Local Ollama setup works
- [ ] EC2 option explains requirements
- [ ] Privacy options require local Ollama
- [ ] AWS options check credentials
- [ ] Configuration saves to ~/.cloudai.yaml
- [ ] Scan works in demo-cdk/
- [ ] Q&A gives accurate answers
- [ ] Error messages are helpful

---

## 🚀 Next Steps After Testing

Once you've tested the core functionality:

1. **Try different models** (phi3:mini, llama3.2:1b)
2. **Test on real projects** (your actual infrastructure)
3. **Experiment with questions** (cost, security, architecture)
4. **Test quota handling** (if you have AWS access)
5. **Try privacy options** (if you have API keys)

Have fun testing! 🎉
