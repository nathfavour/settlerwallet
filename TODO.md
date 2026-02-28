# TODO.md: settlerWallet Roadmap

## Phase 1: The Secure Core
- [x] Implement BIP39/44 HD derivation logic in Go.
- [x] Build the AES-256-GCM Vault for encrypted user secrets storage.
- [ ] Create the Signer Service for payload signing.
- [x] Implement `mlock` and memory zeroing utilities for Linux.

## Phase 2: The Multi-Chain Connector
- [x] Implement BNB/EVM transaction broadcasting using `ethclient`.
- [x] Implement Solana transaction construction and broadcasting using `gagliardetto/solana-go`.
- [x] Define standardized `Balance` and `Transfer` types across both chains.
- [ ] Implement basic RPC connection management/failover.

## Phase 3: The Agentic Loop
- [ ] Create the `Nexus` dispatcher for routing events to user goroutines.
- [ ] Build the `Scheduler` to run strategies every N blocks.
- [x] Implement the `Guardrail Engine` for slippage and daily spend limits.
- [x] Build the `telebot` middleware for multi-user session management (Menus & Persistent Vaults).
- [ ] Implement the "Alpha Strategy": Auto-Compounder for AsterDEX.

## Long-term & Polish
- [ ] gRPC/Unix-socket daemon interface for CLI/TUI clients.
- [x] Persistence layer: SQLite for user vaults and guardrail rules.
- [ ] Persistence layer: PostgreSQL for blobs and Redis for price caching.
- [ ] Advanced Guardrails: Contract whitelisting and MEV protection.
