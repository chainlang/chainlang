# Hanticoin

Blockchain core for Hanticoin: public PoS chain, 21B HTC fixed supply, account-based state, P2P sync, and testnet tooling.

**Features:** Single/multi-node, Proof of Stake (round-robin validators), fee distribution, slashing, block explorer API, stress-test loadgen, testnet runbook, mainnet prep docs.

## Build

```bash
go build -o hanticoin ./cmd/node
go build -o loadgen ./cmd/loadgen   # optional: stress-test tool
```

## Quick start

**Init and run one node:**

```bash
./hanticoin -init -data-dir=./data
./hanticoin -run -data-dir=./data
```

API: `http://localhost:8080`, P2P: `:3030`.

**Join a second node** (same genesis):

```bash
./hanticoin -init-join=./data1 -data-dir=./data2
./hanticoin -run -data-dir=./data2 -p2p-listen=127.0.0.1:3031 -p2p-seeds=127.0.0.1:3030
```

**Create key and send HTC** (node running):

```bash
./hanticoin -keygen=./mykey.json
./hanticoin -send-key=./mykey.json -send-to=<addr_hex> -send-amount=1000000 -send-api=http://localhost:8080
```

## API

| Endpoint | Description |
|----------|-------------|
| `POST /tx` | Submit transaction (JSON) |
| `GET /status` | Chain height, mempool size |
| `GET /account?address=<hex>` | Balance and nonce |
| `GET /blocks?limit=20` | Last N blocks |
| `GET /block?height=N` or `?hash=HEX` | One block |
| `GET /metrics` | height, mempool, peers, block_time_s |

## Testnet and stress test

**Bootstrap 5 nodes:** `./scripts/testnet-bootstrap.sh`

**Load generator:** `./loadgen -api=http://localhost:8080 -key=./path/to/validator_key.json -total=2000 -c=10`

See [docs/TESTNET.md](docs/TESTNET.md) for full testnet runbook.

## Docs

| Doc | Description |
|-----|-------------|
| [docs/SPEC.md](docs/SPEC.md) | Chain spec: params, block/tx, crypto, consensus |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Repo layout, packages, data flow |
| [docs/TESTNET.md](docs/TESTNET.md) | Testnet runbook, loadgen, limitations |
| [docs/MAINNET.md](docs/MAINNET.md) | Mainnet prep, validator guide, node ops |
| [docs/SECURITY.md](docs/SECURITY.md) | Security checklist and recommendations |

## License

See repository.
