# Hanticoin – Chain specification

Locked specification for the Hanticoin blockchain core. Wallet and app are out of scope.

---

## 1. Network parameters

| Parameter   | Value |
|------------|--------|
| Type       | Public blockchain |
| Consensus  | Proof of Stake (no mining) |
| Supply     | 21,000,000,000 HTC (fixed) |
| Decimals   | 6 (1 HTC = 10⁶ smallest units) |
| Block time | 3–5 s (target 4 s) |
| Finality   | 1–2 blocks |

---

## 2. Block structure

| Field             | Type     | Description |
|-------------------|----------|-------------|
| Height            | uint64   | Block number |
| Timestamp         | int64    | Unix time |
| PreviousHash      | Hash     | 32-byte SHA256 of previous block |
| MerkleRoot        | Hash     | Merkle root of transaction hashes |
| ValidatorAddress  | Address  | 20-byte producer address |
| Transactions      | []Tx     | Ordered list of transactions |
| Signature         | []byte   | ECDSA signature of block hash |
| Hash              | Hash     | SHA256 of canonical block payload |

**Block hash:** SHA256 of encoded (Height, Timestamp, PreviousHash, MerkleRoot, ValidatorAddress, list of tx hashes). Signature and Hash are not part of the hashed payload.

**Merkle root:** Root of Merkle tree over transaction hashes (each tx hash = SHA256 of canonical tx payload).

---

## 3. Transaction structure

| Field        | Type   | Description |
|-------------|--------|-------------|
| From        | Address| Sender (20 bytes) |
| To          | Address| Recipient |
| Amount      | uint64 | Smallest units |
| Fee         | uint64 | ≥ MinFee (smallest units) |
| Nonce       | uint64 | Replay protection |
| SenderPubkey| []byte | Compressed ECDSA public key (for verification) |
| Signature   | []byte | ECDSA signature of tx hash |

**Tx hash (for signing):** SHA256 of encoded (From, To, Amount, Fee, Nonce).

**Rules:**

- Signature valid (ECDSA P-256); `PubkeyToAddress(SenderPubkey) == From`.
- Balance(From) ≥ Amount + Fee.
- Nonce(From) == tx.Nonce (then incremented).
- Fee ≥ MinFee.

---

## 4. Crypto

| Component   | Choice |
|-------------|--------|
| Signatures  | ECDSA P-256 (64 bytes R\|S) |
| Hashing     | SHA256 (block hash, tx hash, Merkle) |
| Address     | 20 bytes = first 20 bytes of SHA256(compressed_pubkey) |

---

## 5. Consensus (PoS)

- **Validator set:** Ordered list (address, stake). From genesis; stake ≥ MinStake.
- **Producer:** For height H, producer = Validators[H % len(Validators)] (round-robin).
- **Block:** Must be signed by the producer for that height; others reject.
- **Rewards:** No new minting. Tx fees in a block go to block.ValidatorAddress.
- **Slashing:** Double-sign (same height, two blocks from same validator) → stake set to 0, validator removed from set.

---

## 6. Config defaults

Defined in package `config`.

| Parameter        | Value | Meaning |
|------------------|-------|---------|
| TotalSupply      | 21B × 10⁶ | Smallest units |
| Decimals         | 6 | 1 HTC = 10⁶ units |
| MinFee           | 100 | Smallest units per tx |
| MinStake         | 5M × 10⁶ | Per validator (5M HTC) |
| MaxBlockSize     | 1_000_000 | Bytes (approx) |
| BlockTimeTarget  | 4 | Seconds |
| Validator count  | len(GenesisValidators) | Set in genesis (e.g. 21 or 51) |

**Genesis allocation (default):**

| Address (hex suffix) | Label        | Amount (HTC) |
|----------------------|-------------|--------------|
| ...0001              | Treasury    | 10,000,000,000 |
| ...0002              | Staking     | 5,000,000,000 |
| ...0003              | Distribution| 6,000,000,000 |

**P2P defaults:** ListenAddr `:3030`, MaxPeers 50, Seeds nil (set at runtime).

---

## 7. State model

- **Account:** address → (balance, nonce). Persisted in LevelDB.
- **Validators:** Stored as ordered list (validator_list) + per-address stake (vs:addr).
- **Blocks:** Stored by hash in `blocks/`; chain metadata (height, last hash) in `chain.json`.
- **Genesis:** `genesis.json` (block + allocation + validators). Applied once; then blocks applied in order.

---

## 8. Wire / API

- **P2P:** TCP, newline-delimited JSON. Messages: Hello, NewTx, NewBlock, GetBlocks, Blocks, GetPeers, Peers.
- **HTTP API:** POST /tx, GET /status, GET /account, GET /blocks, GET /block, GET /metrics. See README and TESTNET.md.
