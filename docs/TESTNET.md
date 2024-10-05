# Hanticoin Testnet

How to run a testnet (5+ nodes), join, and stress-test.

## Prerequisites

- Built binary: `go build -o hanticoin ./cmd/node`
- Load generator: `go build -o loadgen ./cmd/loadgen`

## 1. Bootstrap first node

```bash
./hanticoin -init -data-dir=./testnet/node1
./hanticoin -run -data-dir=./testnet/node1 -p2p-listen=127.0.0.1:3030
```

API: `http://localhost:8080`, P2P: `127.0.0.1:3030`.

## 2. Join more nodes (same genesis)

In separate terminals, each with its own data dir:

```bash
# Node 2
./hanticoin -init-join=./testnet/node1 -data-dir=./testnet/node2
./hanticoin -run -data-dir=./testnet/node2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030

# Node 3
./hanticoin -init-join=./testnet/node1 -data-dir=./testnet/node3
./hanticoin -run -data-dir=./testnet/node3 -p2p-listen=127.0.0.1:3032 -p2p-seeds=127.0.0.1:3030

# Node 4
./hanticoin -init-join=./testnet/node1 -data-dir=./testnet/node4
./hanticoin -run -data-dir=./testnet/node4 -p2p-listen=127.0.0.1:3033 -p2p-seeds=127.0.0.1:3030,127.0.0.1:3031

# Node 5
./hanticoin -init-join=./testnet/node1 -data-dir=./testnet/node5
./hanticoin -run -data-dir=./testnet/node5 -p2p-listen=127.0.0.1:3034 -p2p-seeds=127.0.0.1:3030,127.0.0.1:3031
```

Each node needs a different `-p2p-listen` port. HTTP API will bind to 8080 on each machine; for local multi-node use different hosts or run on different machines.

## 3. API ports (single machine)

To run 5 nodes on one machine, use different API ports. Add an `-api-port` flag or set env; if not yet implemented, run nodes on different machines or use one node for testing.

(Current node binary uses fixed `:8080`. For local 5-node, run one node per terminal and use the first node’s API for loadgen.)

## 4. Min stake and validators

- **Min stake:** 5,000,000 HTC (see `config.MinStake`).
- Genesis has one validator (node1’s key). Additional nodes from `-init-join` get their own key but are **not** in the validator set until you change genesis to include multiple validators. So only node1 produces blocks unless you create a custom genesis with multiple validator entries.

To run a multi-validator testnet: create a genesis file with multiple (address, stake) entries and distribute it; then init each node with that genesis and the same genesis file.

## 5. Block explorer & metrics

- **Last blocks:** `GET http://localhost:8080/blocks?limit=20`
- **Block by height:** `GET http://localhost:8080/block?height=5`
- **Block by hash:** `GET http://localhost:8080/block?hash=HEX`
- **Metrics:** `GET http://localhost:8080/metrics` (height, mempool, peers, block_time_s)

## 6. Stress test (loadgen)

Use a funded key (e.g. node1’s validator key, or an account that received funds from genesis allocation). Copy node1’s validator key to a temp file for loadgen, or use an address that has balance.

```bash
# Use node1's key (has balance from genesis). Run node1 first.
./loadgen -api=http://localhost:8080 -key=./testnet/node1/validator_key.json -total=2000 -c=10
```

Reported output: `accepted`, `elapsed`, `TPS`. Target: 1000+ TPS in testnet conditions.

## 7. Known limitations (testnet → mainnet)

- **Single validator in default genesis:** `-init` creates one validator; `-init-join` nodes do not join the validator set. For multi-validator testnet, use a custom genesis with multiple validators.
- **HTTP API port:** Fixed to 8080; no per-node API port flag yet for same-machine multi-node.
- **Block time:** Target 4 s; under load, block production still ticks every 4 s (pending txs included).
- **Reorg:** Full-chain reorg replays from genesis; slow for very long chains.
- **P2P:** No TLS; testnet only. Mainnet should use secure transport.
- **Slashing:** Double-sign detection is in-memory per node; no cross-node evidence propagation yet.
