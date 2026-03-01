# Hanticoin documentation

Index of documentation in this repository. The root [../README.md](../README.md) covers build, quick start, and API.

---

## Documents

| File | Purpose |
|------|---------|
| [SPEC.md](SPEC.md) | **Chain specification** — Network parameters, block and transaction structures, crypto (ECDSA, SHA256, Merkle), consensus (PoS, round-robin), config defaults, state model, wire/API summary. |
| [ARCHITECTURE.md](ARCHITECTURE.md) | **System design** — Component diagram, data flow, repo layout, algorithms (tx/block hash, Merkle root, validator selection, block application, fork resolution), mechanisms (PoS, fee distribution, slashing, tx/block validation, P2P, genesis). |
| [CONSENSUS_MODEL.md](CONSENSUS_MODEL.md) | **Consensus mathematical model** — Formal PoS spec: system model (nodes, validators, adversary, safety/liveness), stake model, block production (round-robin), state transition, finality, slashing, economic security, performance, security proof sketch. |
| [WHITEPAPER.md](WHITEPAPER.md) | **Academic whitepaper** — Full protocol specification in research-paper style: abstract, introduction, system model, cryptographic primitives, ledger and transaction model, PoS consensus, finality and fork choice, slashing, economic security, performance, security proofs, monetary policy, conclusion. |
| [TESTNET.md](TESTNET.md) | **Testnet runbook** — Bootstrap and join nodes, block explorer API, load generator, metrics, known limitations. |
| [TWO_NODE_SYNC.md](TWO_NODE_SYNC.md) | **Two-node sync** — Step-by-step: run two nodes on one machine, verify sync, send tx, check both APIs. |
| [MAINNET.md](MAINNET.md) | **Mainnet preparation** — Parameters, genesis allocation audit, economics, validator guide, node operations. |
| [SECURITY.md](SECURITY.md) | **Security** — Checklist, threat model, mainnet recommendations, vulnerability reporting. |

---

## By topic

- **Get started:** Root [../README.md](../README.md) → build, `-init`, `-run`, API.
- **Chain rules and data:** [SPEC.md](SPEC.md).
- **How it works (design and algorithms):** [ARCHITECTURE.md](ARCHITECTURE.md).
- **Formal consensus and proofs:** [CONSENSUS_MODEL.md](CONSENSUS_MODEL.md), [WHITEPAPER.md](WHITEPAPER.md).
- **Running testnet:** [TESTNET.md](TESTNET.md).
- **Two-node sync (step-by-step):** [TWO_NODE_SYNC.md](TWO_NODE_SYNC.md).
- **Running mainnet / validators:** [MAINNET.md](MAINNET.md).
- **Security and audits:** [SECURITY.md](SECURITY.md).
