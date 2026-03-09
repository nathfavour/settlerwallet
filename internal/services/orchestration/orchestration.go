package orchestration

import (
	"fmt"
	"os"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"gopkg.in/yaml.v3"
)

type DONOrchestrator struct {
	Config Config
}

func NewDONOrchestrator(configPath string) (*DONOrchestrator, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read orchestration config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal orchestration config: %w", err)
	}

	return &DONOrchestrator{Config: cfg}, nil
}

// StartOrchestration initializes and starts the CRE workflow.
// This acts as the pre-flight orchestrator before the wallet is triggered.
func (o *DONOrchestrator) StartOrchestration() error {
	// Initialize CRE Runtime (In a real deployment, this is provided by the DON)
	// For local simulation, we can use the SDK components.
	
	// Example: Registering a workflow handler
	// In a real CRE environment, this would be the entry point.
	return nil
}

// OrchestrateWorkflow handles the primary settlement workflow logic.
func (o *DONOrchestrator) OrchestrateWorkflow(ctx cre.RuntimeBase, input string) (*SettlementObject, error) {
	// 1. Compliance (Gatekeeper) - Must be first to save gas/compute
	if o.Config.Orchestration.Compliance.Enabled {
		if err := o.RunComplianceCheck(ctx, input); err != nil {
			return nil, fmt.Errorf("compliance failure: %w", err)
		}
	}

	// 2. Privacy (Confidential HTTP)
	if o.Config.Orchestration.Privacy.Enabled {
		if err := o.RunPrivacyVerification(ctx, input); err != nil {
			return nil, fmt.Errorf("privacy verification failure: %w", err)
		}
	}

	// 3. Intent (AI/LLM Integration)
	var settlement *SettlementObject
	if o.Config.Orchestration.Intent.Enabled {
		var err error
		settlement, err = o.ParseIntent(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("intent parsing failure: %w", err)
		}
	}

	return settlement, nil
}
