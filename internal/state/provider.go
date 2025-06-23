package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Provider is the interface for different state providers (IaC, Live AWS, Cache).
type Provider interface {
	Scan(ctx context.Context, path string) (map[string]interface{}, error)
}

// IaCProvider scans Infrastructure as Code files.
type IaCProvider struct{}

func (p *IaCProvider) Scan(ctx context.Context, path string) (map[string]interface{}, error) {
	// Check for CDK output
	cdkOutPath := filepath.Join(path, "cdk.out")
	if _, err := os.Stat(cdkOutPath); err == nil {
		return p.scanCdk(cdkOutPath)
	}

	// TODO: Add CloudFormation and Terraform file checks here

	return nil, fmt.Errorf("no supported IaC files found in %s", path)
}

func (p *IaCProvider) scanCdk(cdkOutPath string) (map[string]interface{}, error) {
	manifestPath := filepath.Join(cdkOutPath, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("could not read cdk manifest.json: %w", err)
	}

	var manifest struct {
		Artifacts map[string]struct {
			Type       string `json:"type"`
			Properties struct {
				TemplateFile string `json:"templateFile"`
			} `json:"properties"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("could not parse cdk manifest.json: %w", err)
	}

	// Find the first CloudFormation stack artifact
	for _, artifact := range manifest.Artifacts {
		if artifact.Type == "aws:cloudformation:stack" {
			templatePath := filepath.Join(cdkOutPath, artifact.Properties.TemplateFile)
			templateBytes, err := os.ReadFile(templatePath)
			if err != nil {
				return nil, fmt.Errorf("could not read template file %s: %w", templatePath, err)
			}

			var templateData map[string]interface{}
			if err := json.Unmarshal(templateBytes, &templateData); err != nil {
				return nil, fmt.Errorf("could not parse template file %s: %w", templatePath, err)
			}
			return templateData, nil
		}
	}

	return nil, fmt.Errorf("no aws:cloudformation:stack artifact found in cdk manifest")
}
