# settlerWallet: Engineering Specification

## 1. Vision & First Principles
settlerWallet is a high-performance, multi-tenant "Agentic Wallet" daemon written in Go. It treats a private key not as a static secret, but as an active financial actor. The system is designed to be headless, sovereign, and extensible.

**Core Axioms:**
* **The Vault is Passive:** Keys never leave the encrypted memory space of the Signer.
* **The Brain is Active:** Agents are concurrent goroutines that observe state and propose transactions.
* **Trust but Verify:** Every agentic action must pass through a user-defined "Guardrail Engine" before signing.
* **Interface Agnostic:** Telegram is the primary UX, but the core remains a gRPC/Unix-socket daemon.

## 2. Architectural Overview
The system follows a Modular Monolith pattern, separated into four distinct domains:

### A. The Vault (Secure Storage)
* **Technology:** BIP39 (Mnemonics) & BIP44 (HD Paths).
* **Encryption:** AES-256-GCM. Keys are encrypted at rest using a combination of the User's Telegram_ID and a Server_Secret.
* **Pattern:** Singleton Signer. Only one internal module has access to the decrypted master seed in memory.

### B. The Nexus (The Dispatcher)
* **Technology:** Go Channels & context.Context.
* **Responsibility:** Routes incoming triggers (Telegram commands, Webhooks, or Price Alerts) to the correct User Agent.
* **Concurrency:** Each user gets a dedicated "Agent Loop" (goroutine).

### C. The Strategy Engine (Plugin System)
* **Pattern:** Strategy Interface.
* **Extensibility:** Strategies are defined as Go structs implementing a Think() (Action, error) method.
* **Logic:**
  * Observe: Fetch data from BNB (AsterDEX) or Solana (Jupiter/Drift).
  * Analyze: Compare against user thresholds (e.g., APY > 12%).
  * Act: Construct an unsigned transaction payload.

### D. The Guardrail (Risk Management)
* **Responsibility:** Intercepts every transaction proposal.
* **Default Rules:** Max slippage (1%), Max daily spend per agent, and Contract Whitelisting.

## 3. Technical Stack & Dependencies
* **Language:** Go 1.22+ (Focus on generics and range over integers).
* **Blockchain (EVM/BNB):** geth/ethclient for low-level RPC interaction.
* **Blockchain (Solana):** gagliardetto/solana-go for high-performance transaction construction.
* **Database:** PostgreSQL (for encrypted blobs) + Redis (for real-time price caching).
* **UX:** telebot (Telegram Bot Framework).

## 4. Implementation Specifications

### HD Wallet Derivation Paths
| Chain | Path | Purpose |
|---|---|---|
| BNB Chain | m/44'/60'/0'/0/n | Yield farming & AsterDEX interaction. |
| Solana | m/44'/501'/n'/0' | Agentic trading & Liquidity provision. |

### The Agent Interface (`strategy.go`)
```go
type Strategy interface {
    ID() string
    Description() string
    Execute(ctx context.Context, wallet *Vault) (*TransactionProposal, error)
}
```

### Security Protocol: The "Memory Zeroing" Rule
* Private keys must never be logged.
* Sensitive byte slices should be cleared from memory immediately after signing using `runtime.GC()` or manual zeroing where possible.
* Use `mlock` on Linux to prevent sensitive memory from being swapped to disk.

## 5. Development Roadmap
### Phase 1: The Secure Core
* Implement BIP39/44 HD derivation in Go.
* Build the AES-GCM Vault for storing user secrets.
* Establish the Signer Service that accepts payloads and returns signed bytes.

### Phase 2: The Multi-Chain Connector
* Implement BNB/EVM transaction broadcasting.
* Implement Solana transaction broadcasting.
* Standardize the Balance and Transfer types across both chains.

### Phase 3: The Agentic Loop
* Create the Scheduler that runs user strategies every N blocks.
* Build the Telegram Middleware for multi-user session management.
* Implement the first "Alpha Strategy": Auto-Compounder for AsterDEX.

## 6. Engineering Decisions & Trade-offs
* **Decision: No Web UI.** Maximizes development velocity; Telegram handles auth and accessibility.
* **Decision: Local-first Strategy Execution.** Privacy; financial logic runs on trusted servers.
* **Decision: Go over Rust.** Faster iteration for network-heavy, concurrent I/O.
