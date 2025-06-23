# CloudAI-CLI

> **Ask your AWS account a question – get the answer in seconds.**

CloudAI-CLI is a **single-binary Go tool** that turns plain‑English prompts into AWS SDK calls, revealing live infrastructure topology *and* high‑level cost drivers.  
No more hunting through consoles or memorising `aws cli` flags.

---

## ✨ Why CloudAI-CLI?

| Pain today                                                                  | CloudAI-CLI solution                                              |
|-----------------------------------------------------------------------------|-------------------------------------------------------------------|
| "Which Lambda handles this API path?" takes many clicks in the console.     | `cloudai "What Lambda backs GET /users on prod-api?"`             |
| Hard to see all triggers (SQS, cron, API GW) for a function.                | `cloudai "What triggers the process-order Lambda?"`               |
| Cost Explorer UI feels slow & clunky.                                       | `cloudai "Top 3 services by cost last 7 days"`                    |

_All v0.1 operations are **read‑only** for total safety._

---

## 🚀 Key features (v0.1)

### Infrastructure graph

1. **API Gateway → Lambda lookup**

   ```bash
   cloudai "Which Lambda handles GET /users on prod-api?"
   ```

2. **Lambda trigger inspector**

   ```bash
   cloudai "What triggers the process-order Lambda?"
   ```

### FinOps lite

3. **Top spenders**

   ```bash
   cloudai "Top 3 services by cost last 7 days"
   ```

Use `--json` for automation pipelines and `--plan` to print remediation scripts (never executed).

---

## 🛠 Installation

```bash
# Requires Go ≥ 1.22
go install github.com/<your-user>/cloudai@latest

# Or grab a pre-built binary (macOS, Linux, Windows) from the Releases page
```

_Add an alias for speed_: `alias cai="cloudai"`

---

## ⚡ Quick start

```bash
# Map API path to its Lambda
cai "Which Lambda backs GET /users on prod-api"

# List triggers for a Lambda
cai "What triggers process-order Lambda?"

# Show biggest spenders (JSON for jq)
cai --json "Top 3 services by cost last 7 days" | jq .
```

---

## 🧪 Demo Setup

Don't have AWS resources to test with? No problem! We provide a minimal CDK stack to create demo resources.

### Option 1: Quick Demo Setup
```bash
# Set up AWS credentials and verify access
cloudai setup

# Deploy demo resources (API Gateway + Lambda)
cd demo-cdk
npm install
npx cdk deploy

# Test with CloudAI-CLI
cloudai "Which Lambda handles GET /hello on cloudai-demo-api?"

# Clean up when done
npx cdk destroy
```

### Option 2: Manual Setup
If you prefer to create resources manually, see the [demo-cdk/README.md](demo-cdk/README.md) for detailed instructions.

---

## 🗺 Roadmap

| Version | Highlights                                                                                           |
|---------|-------------------------------------------------------------------------------------------------------|
| **v0.1** | `infra.apigw_to_lambda` · `infra.lambda_triggers` · `cost.top` (read-only)                            |
| v0.2    | S3 storage-class recommendations · Reserved/Spot purchase planner                                     |
| v0.3    | `--apply` mode with IAM guard-rails · Slack / VS Code extensions                                      |
| v1.0    | Multi-cloud back-ends (GCP, Azure)                                                                    |

---

## 📦 Tech stack

| Layer      | Choice                          |
|------------|---------------------------------|
| Language   | Go 1.22                         |
| CLI        | Cobra + Viper                   |
| LLM        | OpenAI GPT-4o (default) / Ollama|
| AWS access | AWS SDK v2                      |
| Tables     | olekukonko/tablewriter          |
| CI/CD      | GitHub Actions + Goreleaser     |

Static binary size ≈ 11 MB (Darwin arm64).

---

## 🤝 Contributing

We welcome PRs! Pick a `good first issue` or propose a new intent.

```bash
cloudai "Add support for cost anomalies"
```

See **CONTRIBUTING.md** for setup details.

---

## 📝 License

MIT – free for personal and commercial use.

---

_CloudAI-CLI – conversational visibility for your AWS architecture._
