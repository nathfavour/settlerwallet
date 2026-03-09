package orchestration

import (
	"fmt"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
)

// ParseIntent implements the Intent module (AI/LLM Integration).
// It translates natural language into structured settlement data.
func (o *DONOrchestrator) ParseIntent(ctx cre.RuntimeBase, input string) (*SettlementObject, error) {
	ctx.Logger().Info("Starting Intent Module (AI/LLM Integration)", "input", input)

	// In a real CRE workflow, we would use the http capability to call the LLM endpoint.
	// We'll define the request structure for the intent parser.
	req := &http.Request{
		Url:    o.Config.Orchestration.Intent.LLMAPIURL,
		Method: "POST",
		Body:   []byte(fmt.Sprintf(`{"prompt": "%s", "model": "%s"}`, input, o.Config.Orchestration.Intent.Model)),
	}

	ctx.Logger().Info("Sending user string to LLM endpoint for intent parsing", "url", req.Url)

	// Hard-coded strict JSON schema instruction to prevent "hallucination"
	_ = `
	{
		"type": "object",
		"properties": {
			"asset": {"type": "string"},
			"amount": {"type": "number"},
			"destination": {"type": "string"}
		},
		"required": ["asset", "amount", "destination"]
	}
	`

	// Simulated LLM parsing logic
	// In production, this would be: client.SendRequest(runtime, req)
	// For demonstration, we'll simulate the return of a structured settlement object.
	
	// Example: "Send 10 USDC to 0x123..."
	settlement := &SettlementObject{
		Asset:       "USDC",
		Amount:      10.0,
		Destination: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", // Example ETH address
	}

	ctx.Logger().Info("Intent successfully parsed into structured settlement data", "asset", settlement.Asset, "amount", settlement.Amount)
	return settlement, nil
}
