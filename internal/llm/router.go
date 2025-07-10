package llm

import (
    "context"
    "strings"
)

// Router decides which LLM backend should handle a given question and ensures
// that sensitive data is redacted before it leaves the local process.
//
// The default heuristic is intentionally *very* simple for now – keyword
// matching – but the public API allows you to swap in a smarter classifier
// (e.g. embeddings similarity or a fine-tuned classifier) later without
// changing callers.
//
// A Router is cheap to create; instantiate one per CLI invocation.

type Router struct {
    archClient    *Client // Fine-tuned SageMaker (architecture-aware) model – optional
    generalClient *Client // General purpose LLM (Bedrock/Ollama/OpenAI)

    protector *DataProtector

    // naive keyword trigger list for the architecture brain
    archKeywords []string
}

// NewRouter constructs a router.
//
// If archClient is nil the router silently falls back to the generalClient.
func NewRouter(archClient, generalClient *Client) *Router {
    kw := []string{"architecture", "lambda", "sns", "s3", "vpc", "subnet", "step function", "eventbridge", "api gateway", "trigger", "cloudformation"}
    return &Router{
        archClient:    archClient,
        generalClient: generalClient,
        protector:     NewDataProtector(),
        archKeywords:  kw,
    }
}

// Answer selects the backend, scrubs the prompt + context, forwards the request
// and returns the de-scrubbed answer.
func (r *Router) Answer(ctx context.Context, question, context string) (string, error) {
    // 1. Scrub potentially sensitive data.
    scrubbedQuestion := r.protector.Scrub(question)
    scrubbedContext := r.protector.Scrub(context)

    // 2. Choose backend.
    client := r.chooseClient(strings.ToLower(question))

    // 3. Forward.
    answer, err := client.Answer(ctx, scrubbedQuestion, scrubbedContext)
    if err != nil {
        return "", err
    }

    // 4. De-scrub.
    return r.protector.Unscrub(answer), nil
}

func (r *Router) chooseClient(lowerQ string) *Client {
    if r.archClient == nil {
        return r.generalClient
    }

    for _, kw := range r.archKeywords {
        if strings.Contains(lowerQ, kw) {
            return r.archClient
        }
    }

    // default
    return r.generalClient
}