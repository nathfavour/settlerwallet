# TODO.md: settlerWallet Roadmap

## Phase 1: The Secure Core
- [ ] Implement BIP39/44 HD derivation logic in Go.
- [ ] Build the AES-256-GCM Vault for encrypted user secrets storage.
- [ ] Create the Signer Service for payload signing.
- [ ] Implement `mlock` and memory zeroing utilities for Linux.

## Phase 2: The Multi-Chain Connector
- [ ] Implement BNB/EVM transaction broadcasting using `ethclient`.
- [ ] Implement Solana transaction construction and broadcasting using `gagliardetto/solana-go`.
- [ ] Define standardized `Balance` and `Transfer` types across both chains.
- [ ] Implement basic RPC connection management/failover.

## Phase 3: The Agentic Loop
- [ ] Create the `Nexus` dispatcher for routing events to user goroutines.
- [ ] Build the `Scheduler` to run strategies every N blocks.
- [ ] Implement the `Guardrail Engine` for slippage and daily spend limits.
- [ ] Build the `telebot` middleware for multi-user session management.
- [ ] Implement the "Alpha Strategy": Auto-Compounder for AsterDEX.

## Long-term & Polish
- [ ] gRPC/Unix-socket daemon interface for CLI/TUI clients.
- [ ] Persistence layer: PostgreSQL for blobs and Redis for price caching.
- [ ] Advanced Guardrails: Contract whitelisting and MEV protection.
