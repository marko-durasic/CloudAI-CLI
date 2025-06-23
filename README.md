# CloudAI-CLI

> **[Work in Progress]** Ask questions about your AWS infrastructure, get answers from a local-first AI.

CloudAI-CLI is a single-binary Go tool that uses a **local-first AI** to answer natural language questions about your cloud infrastructure. It works by scanning your Infrastructure as Code (IaC) files to build a knowledge base, ensuring privacy and accuracy.

No more hunting through consoles or memorising `aws cli` flags.

---

## 🚀 Core Concept (Scan then Ask)

The workflow is designed to be private, fast, and developer-friendly:

1.  **Scan:** You run `cloudai scan` in your IaC project directory (e.g., CDK, Terraform). The tool parses your infrastructure definition and saves it to a local `.cloudai/cache.json` file. This step **does not** require AWS credentials.
2.  **Ask:** You ask any question about the scanned infrastructure, like `cloudai "What is the runtime of the main Lambda?"`. The tool uses the local cache and a local LLM (like Ollama) to give you an answer, without sending your data to a third-party service.

_Live AWS account scanning will be added as a fallback option in a future version._

---

## ⚡ Quick Start

```bash
# 1. Scan your project to build a knowledge base
# (This example uses the included demo project)
cd demo-cdk
cloudai scan

# 2. Ask any question about the infrastructure
cloudai "What AWS region is this stack deployed to?"
cloudai "What is the runtime of the cloudai-demo-hello function?"
```

---

## 🧪 Demo Setup

To test the tool, you can use the included CDK demo project.

```bash
# Navigate to the demo directory
cd demo-cdk

# Install CDK dependencies
npm install

# Deploy the demo stack to your AWS account
# (Requires AWS credentials to be configured)
npx cdk deploy

# Now you can scan and ask questions as shown in the Quick Start
cloudai scan
cloudai "Which Lambda handles GET /hello on cloudai-demo-api?"

# Clean up when you're done
npx cdk destroy
```

---

## 🛠 Installation

```bash
# Requires Go ≥ 1.22
go install github.com/marko-durasic/CloudAI-CLI@latest

# Or grab a pre-built binary from the Releases page (coming soon)
```

_Add an alias for speed_: `alias cai="cloudai"`

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

---

_CloudAI-CLI – conversational visibility for your cloud architecture._
