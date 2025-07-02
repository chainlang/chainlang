# Hanticoin – Architecture

System design, algorithms, and mechanisms for the Hanticoin blockchain core.

---

## 1. System design

### 1.1 High-level components

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                      NODE (cmd/node)                     │
                    │  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐  │
                    │  │  Chain   │  │  State   │  │ Consensus (validator  │  │
                    │  │ (blocks/ │  │(LevelDB) │  │ set, round-robin)    │  │
                    │  │ chain.json)│  │          │  └──────────┬───────────┘  │
                    │  └────┬─────┘  └────┬─────┘             │               │
                    │       │              │                    │               │
                    │  ┌────▼──────────────▼────┐  ┌───────────▼───────────┐  │
                    │  │       Mempool          │  │   Block producer       │  │
                    │  │ (pending txs, validate) │  │ (ticker: build+sign+   │  │
                    │  └────┬───────────────────┘  │    apply+broadcast)    │  │
                    │       │                      └───────────┬────────────┘  │
                    │  ┌────▼─────┐  ┌────────────────────────▼─────────────┐  │
                    │  │ HTTP API │  │              P2P layer                │  │
                    │  │ /tx,     │  │ (TCP, messages, PeerSet, gossip)     │  │
                    │  │ /blocks..│  └───────────────────────────────────────┘  │
                    │  └──────────┘                                              │
                    └─────────────────────────────────────────────────────────┘
                                         │
                    ─────────────────────┼─────────────────────
                                         │ TCP, newline-JSON
                            ┌────────────▼────────────┐
                            │   Other nodes (peers)   │
                            └────────────────────────┘
```

- **Chain:** Persists blocks by hash (`blocks/<hash>.json`) and tip metadata (`chain.json`). Append, get by hash/height, sync helpers (GetBlocksFrom), reorg (SetTip, WriteBlock).
- **State:** LevelDB for balances, nonces, validator list and stakes. ApplyGenesis, ApplyBlock (txs + fee to producer), GetValidatorSet, SlashValidator, ResetAndReplay for reorg.
- **Consensus:** Validator set (from state or genesis); round-robin selection; block producer verification.
- **Mempool:** In-memory pending txs; validation on Add; Pending(n) for block building; Remove when txs are included.
- **Block producer:** Ticker (e.g. 4 s). If we are the validator for next height: take txs from mempool → build block → sign → ApplyBlock → AppendBlock → broadcast → remove txs from mempool.
- **P2P:** TCP server and client; newline-delimited JSON messages; PeerSet (add, remove, broadcast, connect to seeds). Handlers: NewTx → mempool + broadcast; NewBlock / Blocks → validate → apply → broadcast; GetBlocks → reply with Blocks.
- **HTTP API:** POST /tx (mempool + broadcast); GET /status, /account, /blocks, /block, /metrics (read from chain/state/mempool/peers).

### 1.2 Data flow summary

| Flow | Path |
|------|------|
| Tx submission | HTTP or P2P → Mempool.Add (validate) → broadcast NewTx |
| Block production | Ticker → Consensus.ValidatorForHeight → build block → State.ApplyBlock → Chain.AppendBlock → broadcast NewBlock → Mempool.Remove |
| Block receive | P2P NewBlock/Blocks → validate (hash, Merkle, validator) → State.ApplyBlock → Chain.WriteBlock + SetTip → broadcast → Mempool.Remove |
| Sync | Send GetBlocks(fromHeight) → peer replies Blocks → apply in order or reorg if longer chain |
| Reorg | Receive longer chain (e.g. full chain from genesis) → State.ResetAndReplay(genesis, validators, blocks) → Chain.SetTip(last) |

### 1.3 Repo layout (Go packages)

```
cmd/node       → init, run, keygen, send, init-join; wires all components
cmd/loadgen    → stress-test: POST /tx in loop, report TPS
core           → Block, Transaction, Chain, Genesis
consensus      → Validator set, ValidatorForHeight, VerifyBlockValidator
p2p            → Transport, messages, PeerSet
mempool        → Pending txs, Add (validate), Pending, Remove
state          → LevelDB: balance, nonce, validators; ApplyGenesis, ApplyBlock, Slash, ResetAndReplay
crypto         → ECDSA, SHA256, Merkle, Address, keyfile
config         → ChainConfig, P2PConfig, defaults
```

---

## 2. Algorithms

### 2.1 Transaction hash

Used for signing, Merkle tree, and deduplication.

1. Build canonical payload: `(From, To, Amount, Fee, Nonce)` (e.g. JSON).
2. `TxHash = SHA256(payload)`.

Signer signs `TxHash`; verifier recomputes `TxHash` and checks ECDSA signature and `From == PubkeyToAddress(SenderPubkey)`.

### 2.2 Block hash

Used for chain linkage and block identity.

1. For each tx in the block, compute `TxHash`.
2. Build canonical payload: `(Height, Timestamp, PreviousHash, MerkleRoot, ValidatorAddress, [TxHashes])`.
3. `BlockHash = SHA256(payload)`.

`Hash` and `Signature` are not part of the hashed payload. Validator signs `BlockHash`.

### 2.3 Merkle root

Commitment to the list of transaction hashes in the block.

1. Input: list of hashes `H[0..n-1]` (each 32 bytes).
2. If empty → zero hash. If single element → `SHA256(H[0])`.
3. Otherwise, build next level: for `i = 0, 2, 4, ...`, set `next[j] = SHA256(H[i] || H[i+1])`; if odd length, duplicate last: `next[j] = SHA256(H[i] || H[i])`.
4. Recursively compute Merkle root of `next` until one hash remains.

Result is the single root hash stored in the block. Verification: recompute from block’s txs and check equality with `block.MerkleRoot`.

### 2.4 Validator selection (round-robin)

Deterministic producer per height.

- **Input:** Ordered validator set `V[0..k-1]` (from genesis/state), block height `H`.
- **Output:** Producer address = `V[H % k].Address`.

So heights 0, k, 2k, ... are produced by the same validator; no randomness. Ensures exactly one allowed producer per height (by construction).

### 2.5 Block application (state transition)

Apply a block’s transactions and fee distribution.

1. **Load:** For every address that appears as `From` or `To` in any tx, load current balance and (for `From`) nonce from state.
2. **Apply txs in order:** For each tx:
   - `total = Amount + Fee`.
   - If `balances[From] < total` → skip (invalid).
   - `balances[From] -= total`; `nonces[From]++`; `balances[To] += Amount`.
   - Accumulate `totalFees += Fee`.
3. **Fee distribution:** `balances[ValidatorAddress] += totalFees` (use balance after step 2 so producer can be a tx sender in the same block).
4. **Persist:** Write all updated balances and nonces to LevelDB in one batch.

Invalid txs are skipped; balance and nonce updates are in-memory then committed once.

### 2.6 Fork resolution (longest chain, reorg)

When a node sees a chain that is longer than its current chain:

1. **Detect:** Receive a block or Blocks batch that does not extend current tip but whose last block has height > current height (or receive full chain from genesis).
2. **Fetch full chain:** If needed, request `GetBlocks(0, max)` to get the full competing chain from genesis.
3. **Validate:** Check each block: hash, Merkle root, correct validator for height. Check chain linkage: `block[i].PreviousHash == block[i-1].Hash`.
4. **Reorg:** Clear state DB (or iterate and delete), then:
   - `ApplyGenesis(allocation, validators)`.
   - For each block in order, `applyBlockLocked(block)` (same logic as ApplyBlock).
   - `Chain.SetTip(lastBlock)` and ensure all blocks are written (WriteBlock) if not already.

Result: state and tip reflect the longer valid chain. “Longest” is by height (no cumulative difficulty).

---

## 3. Mechanisms

### 3.1 Proof of Stake (PoS)

- **Validator set:** Stored in state (ordered list of addresses + stake). Initial set from genesis; stake ≥ MinStake. Can be updated by slashing (removal).
- **Producer:** For each height, producer = round-robin (see 2.4). Only that address may produce the block; others reject blocks with a different `ValidatorAddress` for that height.
- **No mining:** No proof-of-work; blocks are created by the designated validator on a timer (e.g. every 4 s when there are txs).
- **Finality:** De facto 1–2 blocks (after a block is extended, reorg requires a longer chain and full replay).

### 3.2 Fee distribution

- **Source:** Sum of all `Tx.Fee` in the block. No new coin minting.
- **Recipient:** `Block.ValidatorAddress` (the block producer).
- **When:** During `ApplyBlock` / `applyBlockLocked`, after applying all tx balance changes, add `totalFees` to the producer’s balance (so producer balance is “after txs” + fees).

### 3.3 Slashing

- **Trigger:** Double-sign: two different blocks at the same height from the same validator (same `ValidatorAddress`, different block hash).
- **Detection:** Per-node in-memory map: `height → (validator_address → block_hash)`. On receiving a block, if we already have a block at that height from that validator and the hash differs → slash.
- **Action:** `SlashValidator(addr)`: set validator stake to 0 in state, remove address from validator list. The new block is still rejected (we do not apply the double-signed block).
- **Scope:** Slashing is local (no evidence propagation to other nodes in current design). Other nodes will slash only if they also see both blocks.

### 3.4 Transaction validation (mempool)

Before adding a tx to the mempool:

1. **Signature:** ECDSA verify on `TxHash`; `From == PubkeyToAddress(SenderPubkey)`.
2. **Fee:** `Fee >= MinFee`.
3. **Nonce:** `tx.Nonce == State.GetNonce(From)` (next expected nonce).
4. **Balance:** `State.GetBalance(From) >= Amount + Fee`.
5. **Dedupe:** Tx hash not already in mempool.
6. **Capacity:** Mempool not full.

If any check fails, the tx is rejected. Block application may skip invalid txs but mempool only holds valid ones.

### 3.5 Block validation (acceptance)

Before applying a received block:

1. **Structure:** Block hash recomputed from payload (height, timestamp, previousHash, merkleRoot, validatorAddress, tx hashes) must equal `block.Hash`.
2. **Merkle:** Recompute Merkle root from block’s transactions; must equal `block.MerkleRoot`.
3. **Producer:** `Consensus.VerifyBlockValidator(block)` → `ValidatorForHeight(block.Height) == block.ValidatorAddress`.
4. **Chain link:** If we have a current tip, `block.PreviousHash == tip.Hash` (for extending); or for reorg, full chain is validated and applied (see 2.6).
5. **Double-sign:** If we already have a block at this height from this validator with a different hash → slash and reject.

### 3.6 P2P protocol

- **Transport:** TCP; each message is one line (newline-delimited JSON). Message: `{ "kind": "<Kind>", "payload": <json> }`.
- **On connect:** Server sends **Hello:** `{ genesis_hash, listen_addr, height }`. Used for peer discovery (TryConnect to listen_addr) and optional chain-id check.
- **Gossip:** **NewTx** – broadcast to all peers; **NewBlock** – broadcast after accepting a block.
- **Sync:** **GetBlocks** `{ from_height, max }` → responder sends **Blocks** `{ blocks: [...] }` (ordered from `from_height`). Requester applies or reorgs (see 2.6).
- **Discovery:** **GetPeers** / **Peers** (optional); **Hello** carries `listen_addr` so receivers can connect back.
- **No TLS** in current implementation; suitable for testnet. Mainnet should consider secure transport.

### 3.7 Genesis and bootstrap

- **Genesis file:** `genesis.json` contains genesis block, allocation (address → balance), and optional validators (address, stake). All nodes must use the same genesis for the same chain.
- **Init:** First node: generate validator key, write genesis (with that validator), create state and apply genesis allocation + validators, append genesis block to chain.
- **Join:** New node: copy genesis from existing node (`-init-join`), generate new key, apply same genesis to state, append genesis block. New node syncs from height 1 via GetBlocks from peers. New node is not in the validator set unless genesis is edited to include it.

---

## 4. Key files reference

| Concern | Location |
|--------|----------|
| Chain params, defaults | config/config.go |
| Genesis read/write | core/genesis.go |
| Block/tx types, hash, Merkle check | core/types.go |
| Chain storage, GetBlocksFrom, reorg | core/chain.go |
| State, ApplyBlock, Slash, ResetAndReplay | state/state.go |
| Round-robin, VerifyBlockValidator | consensus/consensus.go |
| Mempool Add (validation), Pending, Remove | mempool/mempool.go |
| P2P messages, transport, PeerSet | p2p/message.go, transport.go, peers.go |
| Node wiring, block producer, P2P/HTTP handlers | cmd/node/main.go |

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
