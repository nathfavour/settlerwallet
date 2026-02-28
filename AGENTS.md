# AGENTS.md: Developer & Agent Context

This document provides essential context for AI agents and human developers working on the `settlerWallet` project.

## 1. Project Overview
`settlerWallet` is a high-performance, multi-tenant "Agentic Wallet" daemon written in Go (1.22+). It is headless and uses Telegram as its primary UI.

## 2. Key Architecture & Domains
- **The Vault:** Encrypted storage (AES-256-GCM) using BIP39/44. Memory safety is paramount.
- **The Nexus:** Dispatcher routing events/triggers via Go channels to dedicated user goroutines.
- **Strategy Engine:** Extensible plugin system where agents (Go structs) define financial logic.
- **Guardrail:** Security layer that intercepts all transaction proposals before signing.

## 3. Tech Stack
- **Languages:** Go 1.22+ (generics, range over integers).
- **Blockchains:** BNB (geth/ethclient), Solana (gagliardetto/solana-go).
- **Storage:** PostgreSQL (blobs), Redis (caching).
- **UI:** Telegram (telebot).

## 4. Coding Standards & Conventions
- **Memory Security:** 
  - Never log private keys or sensitive data.
  - Zero out sensitive byte slices immediately after use.
  - Use `mlock` to prevent swap to disk.
- **Interfaces:** 
  - All strategies must implement the `Strategy` interface in `strategy.go`.
  - Use `context.Context` for all async operations.
- **Concurrency:** Each user has a dedicated goroutine loop. Use channels for communication between Nexus and Agent loops.

## 5. Development Workflow
- **Discovery:** Check `ENGINEERING.md` for the single source of truth.
- **Implementation:** Focus on `/internal/vault/` first as per Phase 1.
- **Testing:** Ensure all blockchain interactions have robust unit and integration tests.

## 6. Memory Zeroing Example (Go)
```go
func clearBytes(b []byte) {
    for i := range b {
        b[i] = 0
    }
}
```
## 7. Build Instructions
- Always build binaries into the `bin/` directory to prevent workspace clutter and git pollution.
- Example: `go build -o bin/daemon ./cmd/daemon`
- The `bin/` directory is ignored by git.
