# Hanticoin

**High-Performance Proof-of-Stake Layer-1 Blockchain for Fast, Low-Fee Payments in Somalia.**

Hanticoin Core is the reference implementation of the Hanticoin protocol — a minimal, account-based blockchain designed exclusively for secure and deterministic value transfer. It does not support smart contracts and does not attempt to be a general computing platform.

---

## Why Hanticoin

Blockchain technology began with Bitcoin: the first decentralized digital monetary system. Bitcoin introduced a new model of trust where monetary supply is transparent, rules are fixed, and no single authority controls issuance. Its primary purpose was monetary integrity — to eliminate uncontrolled money creation and restore trust in digital value transfer.

Hanticoin builds on that foundation but focuses specifically on **payment infrastructure for Somalia**.

Digital payments are already dominant in Somalia. However, most existing systems are centralized, opaque, and institution-dependent. Users cannot independently verify:

- Total digital supply
- Reserve backing
- Issuance policy
- Settlement integrity

If a digital monetary system does not have transparent supply or verifiable backing, users must rely entirely on institutional trust. Hanticoin replaces institutional trust with **cryptographic certainty**.

### The Somalia Opportunity

Somalia has high mobile payment adoption, strong remittance inflows, limited traditional banking infrastructure, and a digital-first population. This creates a unique opportunity for a purpose-built settlement blockchain.

Unlike general-purpose platforms, Hanticoin is optimized exclusively for value transfer. This reduces complexity, improves security, and increases performance predictability.

---

## What Is Hanticoin

Hanticoin is a **standalone Layer-1 Proof-of-Stake blockchain** designed for fast, low-fee digital payments. By limiting scope to payments only, the protocol minimizes attack surface, reduces operational risk, and enables predictable performance.

| Property | Value |
|---|---|
| Consensus | Proof-of-Stake (validator-based, round-robin) |
| Block time | 3–5 s (target 4 s) |
| Finality | 1–2 blocks (longest-chain) |
| Total supply | 21,000,000,000 HTC (fixed — hard invariant) |
| Precision | 6 decimals (1 HTC = 10⁶ smallest units) |
| State model | Account-based (balance, nonce) |
| Smart contracts | Not supported |
| Validator rewards | Transaction fees only — no inflation beyond genesis |

The protocol enforces a hard monetary invariant: **total supply cannot exceed 21 billion HTC**.

### What the Protocol Guarantees

- Transparent and publicly verifiable total supply
- Immutable transaction history
- Deterministic state execution
- Cryptographic ownership
- Supermajority finality via stake-based validator accountability
- No central authority can inflate supply beyond protocol rules

---

## Features

- **Account-based model** — Balances and nonces per address; no UTXO set.
- **Validator-based consensus** — Ordered validator set; block producer selected round-robin by height.
- **Deterministic state machine** — Same block sequence yields the same state on all nodes.
- **Low-latency block production** — Target 4 s block time; single leader per round.
- **Minimal transaction fees** — Configurable minimum fee (e.g. 100 smallest units); fees go to the block producer.
- **Fixed monetary supply** — 21B HTC at genesis; no minting.
- **Slashing for equivocation** — Double-signing results in full stake loss and removal from the validator set.
- **Modular architecture** — Separate packages for core, consensus, P2P, state, mempool, crypto, and config.

---

## Architecture

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

| Component | Responsibility |
|---|---|
| **Core** | Block and transaction types, chain storage (blocks by hash, tip metadata), genesis, Merkle and hash verification |
| **Consensus** | Validator set (from genesis/state), round-robin leader selection by height, block producer verification |
| **State** | Account balances and nonces, validator set and stakes; `ApplyGenesis`, `ApplyBlock` (incl. fee to producer), `SlashValidator`, `ResetAndReplay` for reorgs |
| **Mempool** | Pending transactions; validation on add; feed for block building |
| **P2P** | TCP transport, newline-delimited JSON messages, peer set, gossip (NewTx, NewBlock), sync (GetBlocks/Blocks) |
| **Crypto** | ECDSA P-256, SHA256, Merkle tree, address derivation, key file I/O |
| **Config** | Chain and P2P parameters (supply, decimals, min fee, min stake, block time, listen addr, seeds) |

---

## Repository Structure

```
hanticoin/
├── cmd/
│   ├── node/        # Node binary: init, run, keygen, send, init-join
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

---

## Installation

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

The `hanticoin` binary supports `init`, `run`, `keygen`, `send`, and `init-join`. No separate daemon binary is needed.

---

## Running a Node

**First node (bootstrap)**

```bash
./hanticoin -init -data-dir=./data
./hanticoin -run -data-dir=./data
```

- `-init` creates the data directory, generates a validator key, writes genesis (with this node as sole validator), applies genesis to state, and appends the genesis block.
- `-run` starts the node: P2P server, HTTP API, block production when this node is the validator for the next height, and sync when receiving blocks from peers.

**Second node (join existing chain)**

```bash
./hanticoin -init-join=./data -data-dir=./data2
./hanticoin -run -data-dir=./data2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030
```

- `-init-join` copies genesis from the first node's data directory and creates a new key (this node is not in the validator set unless genesis is edited).
- `-run` with `-p2p-seeds` connects to the first node and syncs from height 1.

**Validator vs. full node**

- **Validator:** A node whose key is in the genesis validator set. It produces a block at height *r* if it is the round-robin leader for *r*. No separate "validator mode" flag is needed.
- **Full node:** Any node running `-run` that is not in the validator set. Non-validators sync and relay only.

**Default ports**

| Service | Default |
|---|---|
| P2P | `:3030` (override with `-p2p-listen`) |
| HTTP API | `:8080` |

Logs go to stdout/stderr. No built-in log rotation — use the OS or a process manager.

---

## Configuration

Parameters are defined in the `config` package (defaults in code) and in **genesis** (written at init). There is no separate config file; node behavior is controlled by flags and genesis.

**Chain parameters**

| Parameter | Default | Description |
|---|---|---|
| `TotalSupply` | 21B × 10⁶ | Total supply in smallest units |
| `Decimals` | 6 | 1 HTC = 10⁶ units |
| `MinFee` | 100 | Minimum fee per tx (smallest units) |
| `MinStake` | 5M × 10⁶ | Minimum stake to be a validator (5M HTC) |
| `MaxBlockSize` | 1,000,000 | Approximate max block size (bytes) |
| `BlockTimeTargetSeconds` | 4 | Target block interval |

**P2P flags**

| Flag | Default | Description |
|---|---|---|
| `-data-dir` | `./data` | Data directory (genesis, state, chain) |
| `-p2p-listen` | `:3030` | P2P listen address |
| `-p2p-seeds` | (none) | Comma-separated seed peers |

Genesis is stored under the data directory (e.g. `./data/genesis.json`). All nodes on the same network must use the same genesis file.

**Example: custom data directory and seeds**

```bash
./hanticoin -run -data-dir=/var/lib/hanticoin -p2p-listen=0.0.0.0:3030 -p2p-seeds=seed1.example.com:3030,seed2.example.com:3030
```

---

## Quick Reference

**Generate a key and send HTC**

```bash
./hanticoin -keygen=./mykey.json
./hanticoin -send-key=./mykey.json -send-to=<addr_hex> -send-amount=1000000 -send-api=http://localhost:8080
```

**HTTP API**

| Endpoint | Description |
|---|---|
| `POST /tx` | Submit a transaction |
| `GET /status` | Chain height and mempool size |
| `GET /account?address=<hex>` | Balance and nonce |
| `GET /blocks?limit=20` | Last N blocks |
| `GET /block?height=N` or `?hash=HEX` | Single block |
| `GET /metrics` | Height, mempool, peers, block_time_s |

**Testnet**

```bash
# Bootstrap
./scripts/testnet-bootstrap.sh

# Load generator
./loadgen -api=http://localhost:8080 -key=./path/to/validator_key.json -total=2000 -c=10
```

Full testnet runbook: [docs/TESTNET.md](docs/TESTNET.md)

---

## Development

```bash
# Run all tests
go test ./...

# Format code
go fmt ./...

# Lint (install golangci-lint if needed)
golangci-lint run
```

**Contributing**

- Open an issue or pull request on the repository.
- Follow existing style (format, naming conventions).
- Add or update tests for any behavior changes.

**Branches**

- `main` — stable, release-ready.
- Feature branches as needed; merge after review.

**Versioning**

Semantic versioning (e.g. `v1.0.0`) for releases.

---

## Security Model

**Slashing**

If a validator produces two different blocks at the same height (equivocation / double-signing), any node that observes both blocks slashes that validator: stake is set to zero and the validator is removed from the set. The penalty is a full stake loss — no partial slash in the base design.

**Validator responsibilities**

- Keep the private key secure.
- Produce at most one block per height; never sign conflicting blocks.
- Run a correct, up-to-date node and follow fork-choice (longest chain by height).

**Protocol upgrades**

Parameter or consensus changes require network-wide coordination. For mainnet, a clear upgrade process is documented in [docs/MAINNET.md](docs/MAINNET.md).

**Vulnerability reporting**

Report security issues privately to a listed maintainer or security contact. Do not disclose publicly before a fix or advisory is agreed upon.

---

## Roadmap

| Phase | Status | Details |
|---|---|---|
| Testnet | Available | Bootstrap script and load generator ready — see [docs/TESTNET.md](docs/TESTNET.md) |
| Security Audit | Planned | Independent review before mainnet — see [docs/SECURITY.md](docs/SECURITY.md) |
| Mainnet | Planned | Audited code, documented parameters and genesis — see [docs/MAINNET.md](docs/MAINNET.md) |
| Validator Decentralization | Future | Expand validator set and geographic distribution over time |

---

## Documentation

| Document | Description |
|---|---|
| [docs/SPEC.md](docs/SPEC.md) | Chain spec: params, block/tx, crypto, consensus |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, algorithms, mechanisms |
| [docs/TESTNET.md](docs/TESTNET.md) | Testnet runbook, load generator, limitations |
| [docs/TWO_NODE_SYNC.md](docs/TWO_NODE_SYNC.md) | Two-node sync: step-by-step usage and testing |
| [docs/MAINNET.md](docs/MAINNET.md) | Mainnet prep, validator guide, node ops |
| [docs/SECURITY.md](docs/SECURITY.md) | Security checklist and recommendations |
| [docs/WHITEPAPER.md](docs/WHITEPAPER.md) | Academic-style protocol specification |
| [docs/CONSENSUS_MODEL.md](docs/CONSENSUS_MODEL.md) | Formal consensus model |

---

## License

This project is licensed under the **MIT License**. See [LICENSE](LICENSE) for the full text.

---

## Disclaimer

Hanticoin Core is **experimental software**. Use at your own risk. Running a node or validator involves operational and financial risk. The authors and contributors do not guarantee correctness, availability, or fitness for any particular purpose, and are not liable for any loss or damage arising from use of this software. This is not financial or legal advice.
