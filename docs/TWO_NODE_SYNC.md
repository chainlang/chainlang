# Two-node setup: step-by-step usage and testing

This guide walks through running two Hanticoin nodes that sync with each other on one machine, and how to verify they stay in sync.

---

## Prerequisites

- Go 1.21+
- Terminal 1 and Terminal 2 (two windows or tabs)

---

## Step 1: Build the binary

In one terminal, from the repo root:

```bash
cd /path/to/hanticoin
go build -o hanticoin ./cmd/node
```

You should have a `hanticoin` executable.

---

## Step 2: First node — initialize

**Terminal 1:**

```bash
./hanticoin -init -data-dir=./data1
```

Expected output (in order):

- Creating data directory...
- Generating validator key...
- Writing genesis...
- Opening state database...
- Applying genesis (allocation + validators)...
- Opening chain storage...
- Writing genesis block to chain...
- Initialization complete.
- **Initialized. Run with -run to start the node.**

---

## Step 3: First node — start (validator)

**Terminal 1:**

```bash
./hanticoin -run -data-dir=./data1 -p2p-listen=127.0.0.1:3030
```

Expected:

- `p2p listening on 127.0.0.1:3030`
- `API listening on :8080`

Leave this terminal running. Node 1 is the **validator** (only node in genesis) and will produce blocks. API: `http://localhost:8080`.

---

## Step 4: Second node — initialize from first (join same chain)

**Terminal 2** (new terminal, same machine):

```bash
./hanticoin -init-join=./data1 -data-dir=./data2
```

This copies genesis from `data1` and creates a new key for node 2. Node 2 is **not** in the validator set; it will sync blocks from node 1.

Expected:

- Same init progress lines as step 2.
- **Initialized from existing genesis. Run with -run -p2p-seeds=<first node> to sync.**

---

## Step 5: Second node — start with seed

**Terminal 2:**

```bash
./hanticoin -run -data-dir=./data2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030 -api-port=8081
```

- `-p2p-listen=127.0.0.1:3031` — different P2P port so it doesn’t clash with node 1.
- `-p2p-seeds=127.0.0.1:3030` — connect to node 1 for sync.
- `-api-port=8081` — different API port so both nodes can run on one machine.

Expected:

- `p2p listening on 127.0.0.1:3031`
- `API listening on :8081`

Leave this running. Node 2’s API: `http://localhost:8081`.

---

## Step 6: Verify both nodes see the same chain

**From any terminal (or browser):**

**Node 1 (validator):**

```bash
curl -s http://localhost:8080/status
```

**Node 2 (syncing):**

```bash
curl -s http://localhost:8081/status
```

Both should return JSON with the same `height` (after a few seconds). Example:

```json
{"height":0,"mempool":0,"peers":1,"block_time_s":0}
```

Node 2 should show `peers: 1` (connected to node 1). Heights should match.

---

## Step 7: Wait for a new block (optional)

Node 1 produces a block about every 4 seconds. After ~5–10 seconds:

**Node 1:**

```bash
curl -s http://localhost:8080/status
```

**Node 2:**

```bash
curl -s http://localhost:8081/status
```

Both should show the same `height` (e.g. 1, 2, 3…). That confirms node 2 is syncing blocks from node 1.

---

## Step 8: List blocks on both nodes

**Node 1:**

```bash
curl -s "http://localhost:8080/blocks?limit=5"
```

**Node 2:**

```bash
curl -s "http://localhost:8081/blocks?limit=5"
```

The list of blocks (heights, hashes) should be the same on both.

---

## Step 9: Submit a transaction and see it on both nodes

**Get the validator address (node 1):**

```bash
grep -A1 '"address"' ./data1/genesis.json | head -2
```

Or from the genesis block:

```bash
curl -s "http://localhost:8080/block?height=0" | grep -o '"validator_address":"[^"]*"'
```

**Generate a second key (for “recipient”):**

```bash
./hanticoin -keygen=./mykey.json
```

Note the printed address (hex).

**Send HTC from node 1’s validator to the new address:**

```bash
./hanticoin -send-key=./data1/validator_key.json -send-to=RECIPIENT_HEX -send-amount=1000000 -send-api=http://localhost:8080
```

Replace `RECIPIENT_HEX` with the address from `-keygen`. Use the full 40-character hex string.

**Check mempool then blocks:**

- `curl -s http://localhost:8080/status` — mempool may show 0 after the tx is in a block.
- `curl -s "http://localhost:8080/blocks?limit=3"` — latest block should include the tx.
- `curl -s "http://localhost:8081/blocks?limit=3"` — same block should appear on node 2.

**Check balance on node 2:**

```bash
curl -s "http://localhost:8081/account?address=RECIPIENT_HEX"
```

You should see the updated balance and nonce. This confirms the same state after sync.

---

## Step 10: Metrics (optional)

**Node 1:**

```bash
curl -s http://localhost:8080/metrics
```

**Node 2:**

```bash
curl -s http://localhost:8081/metrics
```

You can compare height, peers, and block time.

---

## Summary: ports and roles

| Node   | Data dir | P2P listen    | API port | Role                    |
|--------|----------|---------------|----------|-------------------------|
| Node 1 | `./data1`| 127.0.0.1:3030| 8080     | Validator (produces blocks) |
| Node 2 | `./data2`| 127.0.0.1:3031| 8081     | Full node (syncs from node 1) |

**Commands at a glance**

- **Terminal 1:**  
  `./hanticoin -run -data-dir=./data1 -p2p-listen=127.0.0.1:3030`

- **Terminal 2:**  
  `./hanticoin -run -data-dir=./data2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030 -api-port=8081`

---

## Two nodes on two machines

- **Machine A (validator):**  
  `./hanticoin -init -data-dir=./data` then  
  `./hanticoin -run -data-dir=./data -p2p-listen=0.0.0.0:3030`  
  (Use the machine’s IP or hostname for seeds.)

- **Machine B:**  
  `./hanticoin -init-join=/path/to/data/from/A -data-dir=./data` then  
  `./hanticoin -run -data-dir=./data -p2p-listen=0.0.0.0:3031 -p2p-seeds=MACHINE_A_IP:3030`  

Replace `MACHINE_A_IP` with A’s IP. Both can use default API port 8080.

---

## Troubleshooting

- **Node 2 height stays 0** — Check that node 1 is running and that node 2 has `-p2p-seeds=127.0.0.1:3030`. Give it a few seconds and check `curl http://localhost:8081/status` again (peers should be 1).
- **“address already in use”** — Another process is using that port. Use different `-p2p-listen` or `-api-port` (e.g. 3032, 8082).
- **Send fails** — Ensure the recipient address is 40 hex characters and the sender key has balance (validator key from data1 has genesis allocation).
- **Different genesis** — Both nodes must use the same genesis. Always use `-init-join=./data1` for node 2 so it copies node 1’s genesis.
