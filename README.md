# 🏦 settlerWallet
> **The High-Performance, Agentic Wallet Daemon.**

`settlerWallet` is not just a wallet; it's an active financial actor. Written in Go for maximum performance and safety, it transforms private keys from static secrets into autonomous agents that observe, analyze, and act across multiple chains.

---

## ✨ Features

- **🧠 Agentic Brain:** Concurrent goroutines for every user, running autonomous strategies.
- **🛡️ Secure Vault:** BIP39/44 HD derivation with mandatory AES-256-GCM encryption and memory zeroing.
- **⛓️ Multi-Chain:** Native support for **BNB Chain (EVM)** and **Solana**.
- **🚦 Guardrail Engine:** Programmable risk management (slippage, daily spend, whitelisting).
- **📱 Telegram First:** Headless daemon controlled via a familiar Telegram interface.
- **⚡ CGO-Free:** Pure Go implementation for effortless portability.

---

## 🚀 Quick Start

### Installation (via Anyisland)

The easiest way to get `settlerWallet` up and running is through the [Anyisland CLI](https://github.com/nathfavour/anyisland).

```bash
anyisland install nathfavour/settlerwallet
```

### Manual Build

If you prefer building from source:

```bash
git clone https://github.com/nathfavour/settlerwallet.git
cd settlerwallet
go build -o bin/settlerwallet ./cmd/settler
```

---

## 🛠️ Usage

1. **Set your Telegram Token:**
   ```bash
   export TELEGRAM_BOT_TOKEN="your_token_here"
   ```

2. **Launch the Daemon:**
   ```bash
   ./bin/settlerwallet
   ```

3. **Initialize your Agent:**
   Open your Telegram bot and send `/start` followed by `/setup`.

---

## 🏗️ Architecture

- **The Vault:** Secure key storage and signing.
- **The Nexus:** Multi-tenant dispatcher for user agents.
- **Strategy Engine:** Plug-and-play financial logic.
- **The Guardrail:** Safety interceptor for all transactions.

---

## 📜 License

[MIT](LICENSE) - Built with ❤️ by nathfavour.
