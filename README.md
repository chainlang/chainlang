# Hanticoin Core

**A standalone Proof-of-Stake Layer-1 blockchain optimized for fast, low-fee payments.**

Hanticoin Core is a minimal blockchain protocol for deterministic value transfer. It uses an account-based state model, validator-based Proof-of-Stake consensus with round-robin block production, and a fixed supply. It does not support smart contracts. The implementation prioritizes simplicity, security, and predictable performance.

---

## 1. Project Title and Tagline

- **Project name:** Hanticoin Core  
- **Tagline:** High-performance Proof-of-Stake Layer-1 blockchain for fast, low-fee payments.  
- **Technical summary:** Account-based chain with 21B HTC fixed supply, 6 decimal precision, ~4 s block time, round-robin validator selection, and P2P sync. Built for payment settlement without general-purpose computation.

---

## 2. Overview

Hanticoin is a **dedicated payment blockchain** with the following design:

| Property | Value |
|----------|--------|
| Consensus | Proof-of-Stake (validator-based, round-robin) |
| Block time | 3–5 s (target 4 s) |
| Finality | 1–2 blocks (longest-chain; no BFT voting) |
| Total supply | 21,000,000,000 HTC (fixed) |
| Precision | 6 decimals (1 HTC = 10⁶ smallest units) |
| State model | Account-based (balance, nonce) |
| Smart contracts | Not supported |

**Purpose:** Reliable, low-latency payment settlement with minimal fees and a clear monetary invariant.

**Design principles:**

- **Simplicity** — No smart contracts; transfer-only transactions and deterministic state.
- **Speed** — Short block interval and low confirmation depth.
- **Stability** — Fixed supply, fee-only validator rewards, slashing for equivocation.

**Target use:** Payment networks (e.g. Somalia and similar markets) where fast, low-cost transfers and predictable economics are priorities.

---

## 3. Features

- **Account-based model** — Balances and nonces per address; no UTXO set.
- **Validator-based consensus** — Ordered validator set; block producer = round-robin by height.
- **Deterministic state machine** — Same block sequence yields same state on all nodes.
- **Low-latency block production** — Target 4 s block time; single-leader per round.
- **Minimal transaction fees** — Configurable minimum fee (e.g. 100 smallest units); fees go to block producer.
- **Fixed monetary supply** — 21B HTC at genesis; no minting.
- **Modular architecture** — Separate packages for core, consensus, P2P, state, mempool, crypto, config.

---

## 4. Architecture

Components and data flow:

- **Core** — Block and transaction types, chain storage (blocks by hash, tip metadata), genesis, Merkle and hash verification.
- **Consensus** — Validator set (from genesis/state), round-robin leader for height, block producer verification.
- **State** — Account balances and nonces, validator set and stakes; `ApplyGenesis`, `ApplyBlock` (incl. fee to producer), `SlashValidator`, `ResetAndReplay` for reorgs.
- **Mempool** — Pending transactions; validation on add; feed for block building.
- **P2P** — TCP transport, newline-delimited JSON messages, peer set, gossip (NewTx, NewBlock), sync (GetBlocks/Blocks).
- **Crypto** — ECDSA P-256, SHA256, Merkle tree, address derivation, key file I/O.
- **Config** — Chain and P2P parameters (supply, decimals, min fee, min stake, block time, listen addr, seeds).

```
                    +---------------------+
                    |   P2P / HTTP API    |
                    +----------+----------+
                               |
    +-----------+    +---------v---------+    +-----------+
    |  Mempool  |    |  Block producer  |    | Consensus |
    +-----+-----+    +---------+--------+    +-----+-----+
          |                    |                     |
          |           +--------v--------+             |
          +----------->   Core (chain)  <------------+
                     +--------+--------+
                              |
                     +--------v--------+
                     |  State (LevelDB)|
                     +----------------+
```

---

## 5. Repository Structure

```
hanticoin/
├── cmd/
│   ├── node/       # Node binary: init, run, keygen, send, init-join
│   └── loadgen/     # Stress-test: POST /tx in loop, report TPS
├── core/            # Block, transaction, chain storage, genesis
├── consensus/       # Validator set, ValidatorForHeight, VerifyBlockValidator
├── p2p/             # Transport, messages, PeerSet
├── state/           # Balances, nonces, validators; ApplyBlock, Slash, ResetAndReplay
├── mempool/         # Pending txs, validation, Pending(n), Remove
├── crypto/          # ECDSA, SHA256, Merkle, address, keyfile
├── config/          # ChainConfig, P2PConfig, defaults
├── docs/            # SPEC, ARCHITECTURE, TESTNET, MAINNET, SECURITY, WHITEPAPER, etc.
├── scripts/         # testnet-bootstrap.sh
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

| Directory | Contents |
|-----------|----------|
| `cmd/node` | Entrypoint: `-init`, `-run`, `-init-join`, `-keygen`, `-send-*` |
| `cmd/loadgen` | Load generator for testnet |
| `core` | Block/transaction types, chain, genesis |
| `consensus` | PoS: validator set, round-robin leader |
| `p2p` | Networking, messages, peers |
| `state` | Account and validator state |
| `mempool` | Transaction pool |
| `crypto` | Signatures, hashing, Merkle, keys |
| `config` | Chain and P2P configuration |
| `docs` | Technical documentation |

---

## 6. Installation

**Requirements**

- Go 1.21 or later  
- Linux or macOS (Windows may work but is not regularly tested)

**Clone and build**

```bash
git clone https://github.com/your-org/hanticoin.git
cd hanticoin
go build -o hanticoin ./cmd/node
```

Optional (stress-test tool):

```bash
go build -o loadgen ./cmd/loadgen
```

**Output:** The `hanticoin` binary supports init, run, keygen, send, and init-join. No separate daemon binary.

---

## 7. Running a Node

**First node (bootstrap)**

```bash
./hanticoin -init -data-dir=./data
./hanticoin -run -data-dir=./data
```

- `-init` creates the data directory, generates a validator key, writes genesis (with this node as sole validator), applies genesis to state, and appends the genesis block.  
- `-run` starts the node: P2P server, HTTP API, block production when this node is the validator for the next height, and sync when receiving blocks from peers.

**Second node (join same chain)**

```bash
./hanticoin -init-join=./data -data-dir=./data2
./hanticoin -run -data-dir=./data2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030
```

- `-init-join` copies genesis from the first node’s data dir and creates a new key (this node is not in the validator set unless genesis is edited).  
- `-run` with `-p2p-seeds` connects to the first node and syncs from height 1.

**Validator vs full node**

- **Validator:** A node whose key is in the genesis validator set. When `-run` is used, it produces a block at height \(r\) if it is the round-robin leader for \(r\) (no separate “validator mode” flag).  
- **Full node:** Any node running `-run`; it may or may not be in the validator set. Non-validators sync and relay only.

**Ports**

- P2P: default `:3030` (override with `-p2p-listen`).  
- HTTP API: default `:8080` (see code for override).

**Logging**

- Logs go to stdout/stderr. No built-in log rotation; use the OS or a process manager.

---

## 8. Configuration

Parameters are defined in `config` (defaults in code) and in **genesis** (written at init). There is no separate config file by default; node behavior is controlled by flags and genesis.

**Chain parameters (config)**

| Parameter | Default | Meaning |
|-----------|---------|---------|
| TotalSupply | 21B × 10⁶ | Total supply in smallest units |
| Decimals | 6 | 1 HTC = 10⁶ units |
| MinFee | 100 | Minimum fee per tx (smallest units) |
| MinStake | 5M × 10⁶ | Minimum stake to be validator (5M HTC) |
| MaxBlockSize | 1_000_000 | Approx. max block size (bytes) |
| BlockTimeTargetSeconds | 4 | Target block interval |

**P2P (flags)**

| Flag | Default | Meaning |
|------|--------|--------|
| `-data-dir` | `./data` | Data directory (genesis, state, chain) |
| `-p2p-listen` | `:3030` | P2P listen address |
| `-p2p-seeds` | (none) | Comma-separated seed peers |

**Genesis**

- Stored under the data directory (e.g. `./data/genesis.json`). Contains genesis block, allocation (address → balance), and validator set (address, stake). All nodes on the same network must use the same genesis.

**Example: custom data dir and seeds**

```bash
./hanticoin -run -data-dir=/var/lib/hanticoin -p2p-listen=0.0.0.0:3030 -p2p-seeds=seed1.example.com:3030,seed2.example.com:3030
```

---

## 9. Development

**Tests**

```bash
go test ./...
```

**Format**

```bash
go fmt ./...
```

**Linting**

```bash
golangci-lint run
```

(Install with `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` if needed.)

**Contribution**

- Open an issue or PR on the repository.  
- Follow existing style (format, naming).  
- Add or update tests for behavior changes.

**Branches**

- `main`: stable, release-ready.  
- Feature branches as needed; merge after review.

**Versioning**

- Semantic versioning (e.g. v1.0.0) for releases when adopted.

---

## 10. Security Model

**Slashing**

- **Equivocation (double-signing):** If a validator produces two different blocks at the same height, any node that observes both slashes that validator: stake set to zero and validator removed from the set.  
- **Penalty:** Full stake loss (\(\alpha = 1\)); no partial slash in the base design.

**Validator responsibilities**

- Keep private key secure.  
- Produce at most one block per height; do not sign conflicting blocks.  
- Run a correct, up-to-date node and follow fork-choice (longest chain by height).

**Upgrades**

- Protocol or parameter changes (e.g. new genesis, new validator set) require coordination and, for mainnet, a clear upgrade process (see [docs/MAINNET.md](docs/MAINNET.md)).

**Vulnerability reporting**

- Report security issues privately (e.g. to a listed security contact or maintainer). Do not disclose in public issues before a fix or advisory is agreed.

---

## 11. Roadmap

- **Testnet** — Public or invited testnet; bootstrap script and loadgen available ([docs/TESTNET.md](docs/TESTNET.md)).  
- **Audit** — Independent security review before mainnet ([docs/SECURITY.md](docs/SECURITY.md)).  
- **Mainnet** — Launch with audited code, documented parameters and genesis ([docs/MAINNET.md](docs/MAINNET.md)).  
- **Decentralization** — Expand validator set and geographic distribution over time.

---

## 12. License

This project is licensed under the **MIT License**. See [LICENSE](LICENSE) for the full text.

---

## 13. Disclaimer

Hanticoin Core is **experimental software**. Use at your own risk. Running a node or validator may involve operational and financial risk. The authors and contributors do not guarantee correctness, availability, or fitness for any particular purpose and are not liable for any loss or damage arising from use of this software. This is not financial or legal advice.

---

## Quick reference

**Create key and send HTC (node running)**

```bash
./hanticoin -keygen=./mykey.json
./hanticoin -send-key=./mykey.json -send-to=<addr_hex> -send-amount=1000000 -send-api=http://localhost:8080
```

**HTTP API**

| Endpoint | Description |
|----------|-------------|
| `POST /tx` | Submit transaction |
| `GET /status` | Chain height, mempool size |
| `GET /account?address=<hex>` | Balance and nonce |
| `GET /blocks?limit=20` | Last N blocks |
| `GET /block?height=N` or `?hash=HEX` | One block |
| `GET /metrics` | height, mempool, peers, block_time_s |

**Testnet**

- Bootstrap script: `./scripts/testnet-bootstrap.sh`  
- Loadgen: `./loadgen -api=http://localhost:8080 -key=./path/to/validator_key.json -total=2000 -c=10`  
- Full runbook: [docs/TESTNET.md](docs/TESTNET.md)

**Documentation**

| Doc | Description |
|-----|-------------|
| [docs/SPEC.md](docs/SPEC.md) | Chain spec: params, block/tx, crypto, consensus |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, algorithms, mechanisms |
| [docs/TESTNET.md](docs/TESTNET.md) | Testnet runbook, loadgen, limitations |
| [docs/TWO_NODE_SYNC.md](docs/TWO_NODE_SYNC.md) | Two-node sync: step-by-step usage and testing |
| [docs/MAINNET.md](docs/MAINNET.md) | Mainnet prep, validator guide, node ops |
| [docs/SECURITY.md](docs/SECURITY.md) | Security checklist and recommendations |
| [docs/WHITEPAPER.md](docs/WHITEPAPER.md) | Academic-style protocol specification |
| [docs/CONSENSUS_MODEL.md](docs/CONSENSUS_MODEL.md) | Formal consensus model |
