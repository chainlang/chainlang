# Hanticoin – Architecture and repo layout

High-level design and Go package structure.

---

## 1. Repo layout (Go)

```
hanticoin/
├── cmd/
│   ├── node/          # Node daemon: init, run, keygen, send
│   └── loadgen/       # Stress-test: send many txs, report TPS
├── core/              # Block, Transaction, Chain, Genesis
├── consensus/         # Validator set, round-robin, block verification
├── p2p/               # TCP transport, messages, gossip, peer set
├── mempool/           # Transaction pool, validation
├── state/             # Account state (LevelDB), validators, ApplyBlock, Slash
├── crypto/            # ECDSA, SHA256, Merkle, keyfile
├── config/            # ChainConfig, P2PConfig, defaults
├── docs/              # Spec, architecture, testnet, mainnet, security
└── scripts/           # testnet-bootstrap.sh, etc.
```

---

## 2. Package roles

| Package    | Role |
|-----------|------|
| **cmd/node** | Entrypoint: `-init` (genesis + state), `-run` (block producer + P2P + HTTP API), `-keygen`, `-send`, `-init-join`. Wires chain, state, mempool, consensus, p2p. |
| **cmd/loadgen** | Load generator: funded key, POST /tx in loop, report TPS. |
| **core** | Block and Transaction types; Chain (append, get by hash/height, sync helpers); Genesis (file read/write). |
| **consensus** | Validator set (from genesis or state); ValidatorForHeight (round-robin); VerifyBlockValidator. |
| **p2p** | TCP server and client; message types (Hello, NewTx, NewBlock, GetBlocks, Blocks, Peers); PeerSet (add, broadcast, connect seeds). |
| **mempool** | Pending txs; Add (validate sig, fee, nonce, balance); Pending(n); Remove(included). |
| **state** | LevelDB: balance, nonce, validator list and stake. ApplyGenesis, ApplyBlock (txs + fee to producer), GetValidatorSet, SlashValidator, ResetAndReplay. |
| **crypto** | ECDSA (sign, verify, keygen), SHA256, Merkle root, Address (from pubkey), keyfile (save/load JSON). |
| **config** | ChainConfig (supply, decimals, min fee, min stake, max block size, genesis allocation, genesis validators); P2PConfig (listen, seeds, max peers). Defaults. |

---

## 3. Data flow (node)

**Startup:** Load/create genesis → open state, chain, mempool → load validator set from state → start P2P listener → connect to seeds → optional sync (GetBlocks).

**Block production (ticker):** If we are validator for next height → take txs from mempool → build block → sign → ApplyBlock → AppendBlock → broadcast NewBlock → remove txs from mempool.

**Incoming P2P:** NewTx → mempool.Add → broadcast. NewBlock → verify (hash, Merkle, validator) → apply → append → broadcast. GetBlocks → reply with Blocks. Blocks (batch) → apply in order or reorg if longer chain.

**HTTP:** POST /tx → mempool.Add → broadcast. GET /status, /account, /blocks, /block, /metrics read from chain/state/mempool/peerSet.

---

## 4. Key files

| Concern        | Where |
|----------------|-------|
| Chain params   | config/config.go |
| Genesis file   | core/genesis.go |
| Block/chain DB | core/chain.go (blocks/*.json, chain.json) |
| State DB       | state/state.go (LevelDB) |
| Validator set  | state (storage), consensus (round-robin) |
| P2P messages   | p2p/message.go, transport.go, peers.go |
| Node wiring    | cmd/node/main.go |

---

## 5. Dependencies (conceptual)

- **node** → core, consensus, p2p, mempool, state, crypto, config
- **state** → core, config, crypto
- **consensus** → core, config, crypto
- **mempool** → core, state, crypto, config
- **core** → config, crypto
- **p2p** → core, crypto
- **loadgen** → core, crypto, config

No circular imports; config and crypto are leaves.
