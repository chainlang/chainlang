# Hanticoin – Mainnet prep

Parameters, genesis audit, economics, validator guide, and node operations.

---

## 1. Mainnet parameters (recommended)

Tune in `config` or via config file before mainnet launch.

| Parameter | Current default | Notes |
|-----------|-----------------|--------|
| Block time | 4 s | Keep 3–5 s for finality. |
| Max block size | 1,000,000 bytes | Increase if needed for higher TPS; test first. |
| Min fee | 100 (smallest units) | Anti-spam; raise if needed. |
| Min stake | 5,000,000 HTC | Per validator; 21 or 51 validators typical. |
| P2P max peers | 50 | Increase for well-connected nodes. |

No new coin minting; supply is fixed at 21B HTC.

---

## 2. Genesis allocation audit

Before mainnet, confirm genesis balances and validator set.

**Default allocation (config):**

| Address | Label | Amount (HTC) | Smallest units |
|---------|--------|-------------|----------------|
| `0x...0001` | Treasury | 10,000,000,000 | 10B × 10⁶ |
| `0x...0002` | Staking pool | 5,000,000,000 | 5B × 10⁶ |
| `0x...0003` | Distribution | 6,000,000,000 | 6B × 10⁶ |
| **Total** | | **21,000,000,000** | 21B × 10⁶ |

**Checklist:**

- [ ] Sum of allocation = 21,000,000,000 HTC (no more, no less).
- [ ] Addresses (hex) are correct and controlled as intended.
- [ ] Validator set (genesis validators) is decided; each has stake ≥ MinStake (5M HTC).
- [ ] Genesis block and `genesis.json` are identical across all launch participants.

---

## 3. Economics

- **Supply:** 21 billion HTC, fixed. No minting after genesis.
- **Fees:** Every tx pays a fee (≥ MinFee). Fees go to the block producer (validator for that height).
- **Staking:** Validators stake HTC; round-robin block production. No separate “staking reward” mint; rewards = fees only.
- **Slashing:** Double-sign or invalid block → stake set to 0 and validator removed from set.

---

## 4. Validator guide

**Requirements:**

- Stake ≥ MinStake (5,000,000 HTC).
- Run a node with the validator key that holds the staked balance (or is the designated validator address in genesis).
- Stable network and clock; correct genesis and chain data.

**Setup:**

1. Generate key: `./hanticoin -keygen=./validator_key.json`. Back up the file (and optionally the address).
2. Include this address and stake in **genesis validators** (genesis file or init tooling). Ensure genesis is shared with the network.
3. Init node with that genesis: either first node `-init` (and write genesis with your validator) or `-init-join` from an existing genesis.
4. Run: `./hanticoin -run -data-dir=./data -p2p-listen=HOST:3030 -p2p-seeds=SEED1,SEED2`.
5. Your node produces blocks when `height % num_validators == your_index` (round-robin).

**Slashing:**

- Do not run the same validator key on two different chains or with different genesis.
- Double-sign (same height, two blocks) is detected; your stake is zeroed and you are removed from the set. Keys and backups must be single-use per chain.

**Key safety:**

- Store `validator_key.json` with permissions 0600; do not commit or expose.
- Prefer a dedicated machine or restricted access for the validator process.

---

## 5. Node operations

**Run a node (non-validator or validator):**

```bash
./hanticoin -run -data-dir=/var/lib/hanticoin -p2p-listen=:3030 -p2p-seeds=seed1.example.com:3030,seed2.example.com:3030
```

- **data-dir:** Holds `genesis.json`, `chain.json`, `blocks/`, `state/`, `validator_key.json` (if validator). Back this up for recovery.
- **P2P:** Listen on a public or DMZ interface; seeds are bootstrap peers.

**API (default :8080):**

- `POST /tx` – submit transaction.
- `GET /status`, `GET /account?address=`, `GET /blocks`, `GET /block?height=`, `GET /metrics` – see TESTNET.md.

**Backup:**

- Copy `data-dir` (or at least `blocks/`, `state/`, `chain.json`, `genesis.json`, `validator_key.json`) regularly. Restore by placing back and restarting.

**Upgrade:**

- Replace binary; restart. State and chain are forward-compatible. If a hard fork is ever needed, coordination and new genesis/versioning would be required.

**Monitoring:**

- Use `GET /metrics` (height, mempool, peers, block_time_s). Alert on stalled height or zero peers.
