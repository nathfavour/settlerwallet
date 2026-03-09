package main

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/nathfavour/settlerwallet/internal/services/orchestration"
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/spf13/cobra"
)

// MockRuntime implements cre.RuntimeBase for local simulation.
type MockRuntime struct {
	logger *slog.Logger
}

func (m *MockRuntime) CallCapability(request *sdk.CapabilityRequest) cre.Promise[*sdk.CapabilityResponse] {
	return nil
}

func (m *MockRuntime) Rand() (*rand.Rand, error) {
	return rand.New(rand.NewSource(time.Now().UnixNano())), nil
}

func (m *MockRuntime) Now() time.Time {
	return time.Now()
}

func (m *MockRuntime) Logger() *slog.Logger {
	return m.logger
}

var orchestrateCmd = &cobra.Command{
	Use:   "orchestrate [input]",
	Short: "Run the DON-based orchestration layer for a natural language intent.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		fmt.Printf("🧠 Starting DON-based Orchestration for: \"%s\"\n", input)

		// 1. Initialize Orchestrator
		orchestrator, err := orchestration.NewDONOrchestrator("orchestration.yaml")
		if err != nil {
			log.Fatalf("❌ Failed to initialize orchestrator: %v", err)
		}

		// 2. Initialize Mock Runtime
		runtime := &MockRuntime{
			logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
		}

		// 3. Run Orchestration Workflow
		fmt.Println("⏳ Executing pre-flight orchestration (Compliance -> Privacy -> Intent)...")
		settlement, err := orchestrator.OrchestrateWorkflow(runtime, input)
		if err != nil {
			log.Fatalf("❌ Orchestration failed: %v", err)
		}

		// 4. Results
		fmt.Println("\n✅ Orchestration Successful!")
		fmt.Println("--------------------------------")
		fmt.Printf("Asset:       %s\n", settlement.Asset)
		fmt.Printf("Amount:      %.2f\n", settlement.Amount)
		fmt.Printf("Destination: %s\n", settlement.Destination)
		fmt.Println("--------------------------------")
		fmt.Println("🚀 Intent is now ready for core wallet signing.")
	},
}

func init() {
	rootCmd.AddCommand(orchestrateCmd)
}
