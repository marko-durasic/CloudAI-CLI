# CloudAI-CLI

> **Conversational visibility for your cloud architecture.**
> 
> Ask questions about your AWS infrastructure, get answers from a local-first AI. No cloud vendor lock-in, no data leaves your machine.

---

## ✨ Visual Demo

```bash
$ cloudai scan
📋 Infrastructure Summary:
   • Lambda: cloudai-demo-hello (DemoLambda269372AF)
   • API Gateway: cloudai-demo-api (DemoApiE67238F8)

$ cloudai "Which Lambda handles GET /hello on cloudai-demo-api?"
🤖 AI Answer:
The Lambda function handling GET /hello on cloudai-demo-api is: **cloudai-demo-hello**.
```

---

## 🚀 Getting Started

### ⚡ Quick Start (Recommended)

1. **Install Go ≥ 1.22** → [Download Go](https://go.dev/dl/)

2. **Clone and install CloudAI-CLI:**
   ```bash
   git clone https://github.com/ddjura/CloudAI-CLI.git
   cd CloudAI-CLI
   go install ./cmd/cloudai
   ```

3. **Choose your AI backend:**

   **Option A: Local Models (Privacy-focused, Free)**
   ```bash
   # Install Ollama: https://ollama.com/
   ollama pull llama3.2:3b
   cloudai setup-interactive  # Choose option 1
   ```

   **Option B: AWS Models (Fast, Paid)** 🚧 *Work in Progress*
   ```bash
   cloudai auto-setup  # Attempts to enable Bedrock automatically
   ```
   Note: Bedrock access may require manual approval for new AWS accounts.

   **Option C: EC2 GPU Server (High Performance)** 🚧 *Work in Progress*
   ```bash
   ./deploy-ollama-ec2.sh  # Deploy GPU-accelerated Ollama server
   ```
   Cost: ~$0.50/hour, provides GPU-accelerated inference.

### 🔧 Manual Setup (Alternative)

If you prefer manual control:

1. **Install Go ≥ 1.22**  
   [Download Go](https://go.dev/dl/)

2. **Choose your LLM provider:**

   **Option A: AWS Models (Recommended for speed)**
   ```bash
   # Configure AWS credentials
   aws configure
   
   # Set up AWS model (much faster than local)
   export AWS_MODEL_TYPE=bedrock
   export AWS_MODEL_ID=anthropic.claude-3-haiku-20240307-v1:0
   export AWS_REGION=us-east-1
   ```
   
   **⚠️ Important**: You need to enable Bedrock models in your AWS account:
   1. Go to [AWS Bedrock Console](https://console.aws.amazon.com/bedrock/)
   2. Navigate to "Model access" in the left sidebar
   3. Click "Enable specific models" or "Enable all models"
   4. Wait for model access to be granted (usually instant)

   **Option B: Local Ollama (Privacy-focused)**
   ```bash
   # Install [Ollama](https://ollama.com/) and start the server:
   ollama serve
   
   # Pull a model
   ollama pull llama3.2:3b
   ```

3. **Clone and build CloudAI-CLI:**
   ```bash
   git clone https://github.com/ddjura/CloudAI-CLI.git
   cd CloudAI-CLI
   go install ./cmd/cloudai
   ```

4. **Make `cloudai` command available:**
   
   **Option A: Add Go bin to your PATH (Recommended)**
   ```bash
   # Add this line to your shell profile (~/.zshrc, ~/.bashrc, etc.)
   export PATH=$PATH:$(go env GOPATH)/bin
   
   # Then reload your shell profile
   source ~/.zshrc  # or ~/.bashrc
   ```
   
   **Option B: Use the full path**
   ```bash
   # Run directly from Go bin directory
   $(go env GOPATH)/bin/cloudai scan
   ```
   
   **Option C: Create a symlink (Linux/macOS)**
   ```bash
   # Create a symlink in /usr/local/bin (requires sudo)
   sudo ln -s $(go env GOPATH)/bin/cloudai /usr/local/bin/cloudai
   ```

5. **Verify installation:**
   ```bash
   cloudai --help
   ```

6. **(Optional) For the demo:**
   - Install Node.js and AWS CDK
   - Configure AWS credentials

7. **Scan and ask questions:**
   ```bash
   cd demo-cdk
   cloudai scan
   cloudai "Which Lambda handles GET /hello on cloudai-demo-api?"
   ```

8. **See or override the selected model:**
   ```bash
   cloudai model
   ```

---

## 🧪 Demo Project

A CDK demo project is included for you to test the tool:

```bash
cd demo-cdk
npm install
npx cdk deploy  # Requires AWS credentials
cloudai scan
cloudai "Which Lambda handles GET /hello on cloudai-demo-api?"
```

---

## ✨ Why CloudAI-CLI?

- 🚀 **Lightning Fast**: AWS-hosted models (Bedrock, SageMaker) for instant responses, or local Ollama for privacy
- 🔒 **Privacy Options**: Choose between AWS models (fast) or local Ollama (private) - no data leaves your machine with local models
- ⚡ **Smart Model Selection**: Automatically picks the best LLM model for your setup (AWS > Ollama > OpenAI)
- 🤖 **Auto Model Selection**: For local models, automatically picks the best LLM model for your hardware and available models in Ollama
- 🧠 **Persistent Model Config**: Remembers your model choice in `~/.cloudai.yaml` for future runs
- 🖥️ **System-Aware**: Detects your CPU, RAM, and GPU to optimize performance for local models
- 🎯 **Natural Language**: Ask questions in plain English, get precise AWS resource answers
- 🛠 **Developer Friendly**: Single binary, no complex setup, works with existing IaC projects
- 📊 **Smart Summaries**: Get clear, actionable insights about your infrastructure
- 🔄 **Scan Once, Ask Often**: Parse your IaC once, then ask unlimited questions

## 🆚 How it Compares

| Feature | CloudAI-CLI | AWS Console | AWS CLI | Other Tools |
|---------|-------------|-------------|---------|-------------|
| **Privacy** | 🔒 Local-first | ❌ Cloud-based | ❌ Cloud-based | ❌ Cloud-based |
| **Speed** | ⚡ Instant cache | 🐌 Manual navigation | 🐌 Command memorization | 🐌 Complex queries |
| **Natural Language** | ✅ Plain English | ❌ UI navigation | ❌ Technical syntax | ❌ Technical syntax |
| **Offline Support** | ✅ Full offline | ❌ Requires internet | ❌ Requires internet | ❌ Requires internet |
| **Setup Complexity** | ✅ Single binary | ✅ Web browser | ❌ Complex config | ❌ Complex setup |
| **Infrastructure Focus** | ✅ AWS-native | ✅ AWS-native | ✅ AWS-native | ❌ Generic |

---

## ⚡ High-Performance Options

### EC2 GPU Server (Work in Progress)

For maximum performance, deploy Ollama on a GPU-enabled EC2 instance:

```bash
# One-command deployment (WIP)
./deploy-ollama-ec2.sh
```

**Benefits:**
- 🚀 GPU-accelerated inference (10x faster than CPU)
- 🌐 Remote access from anywhere
- 📈 Scalable (can run larger models)
- 🔄 Share with team members

**Cost:** ~$0.526/hour (g4dn.xlarge with T4 GPU)

**Files:**
- `deploy-ollama-ec2.sh` - Automated deployment script
- `ec2-ollama-stack.yaml` - CloudFormation template
- `setup-ec2-ollama.sh` - Manual setup script

---

## 🔧 Troubleshooting

### Setup Issues

**Problem: AWS credentials not found**
```bash
# Solution: Configure AWS credentials
aws configure
# Enter your AWS Access Key ID and Secret Access Key
```

**Problem: Auto-setup fails**
```bash
# Try the step-by-step approach
cloudai bedrock-setup     # Enable Bedrock access
cloudai setup-interactive # Configure settings
```

### AWS Model Access Issues

**Error: "You don't have access to the model with the specified model ID"**

This means the Bedrock model isn't enabled in your AWS account:

1. **Use auto-setup (easiest)**:
   ```bash
   cloudai auto-setup  # Handles everything automatically
   ```

2. **Or enable manually**:
   ```bash
   cloudai bedrock-setup  # Opens console and waits for you
   ```

3. **Check your region**: Some models are only available in specific regions
   ```bash
   export AWS_REGION=us-east-1  # Try us-east-1 first
   ```

### Cost Management

**Daily Budget Exceeded**
```bash
cloudai cost  # Check current usage
```

**Reset Daily Budget**
```bash
cloudai setup-interactive  # Reconfigure with new budget
```

---

## 🛠 Advanced Usage & Configuration

- On first run, CloudAI-CLI auto-selects the best model for your hardware and available Ollama models.
- The selected model is saved to `~/.cloudai.yaml` for future runs.
- You can override the model at any time with the `OLLAMA_MODEL` environment variable or by editing the config file.
- Use `cloudai model` to see your system specs, available models, and current selection.

---

## 🗺 Roadmap

| Version   | Highlights                                                                                               |
|-----------|----------------------------------------------------------------------------------------------------------|
| **v0.1**  | **WIP:** Local-first `scan` for CDK · RAG pipeline for Q&A with Ollama/OpenAI support.                      |
| v0.2      | Add support for Terraform scanning · Fallback to live AWS scan · Cost analysis features.                   |
| v0.3      | `--apply` mode with IAM guard-rails · Deeper resource analysis (e.g., S3 storage classes).               |
| v1.0      | Multi-cloud back-ends (GCP, Azure) · CI/CD integration.                                                    |

---

## 📦 Tech stack

| Layer      | Choice                                       |
|------------|----------------------------------------------|
| Language   | Go 1.22                                      |
| CLI        | Cobra + Viper                                |
| LLM        | Ollama (local-first) / OpenAI GPT-4o (fallback) |
| IaC Parser | Native Go (for now)                          |
| CI/CD      | GitHub Actions + Goreleaser (planned)        |

---

## 🤝 Contributing

We welcome PRs! Pick an issue or propose a new feature. See **CONTRIBUTING.md** for setup details.

---

## 📝 License

MIT
