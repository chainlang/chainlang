# Hanticoin – Security

Internal review checklist and recommendations before mainnet. An external security review is recommended for production.

## Threat model (summary)

- **Consensus:** PoS round-robin; validators are identified by address and stake. Attack requires stake or validator key compromise.
- **Transactions:** ECDSA-signed; replay protected by nonce. Double-spend prevented by ordering and balance checks.
- **P2P:** Unauthenticated TCP; messages are validated (block hash, Merkle, validator, signatures). Malicious peers can send bad blocks/txs but they are rejected.

## Review checklist

### Transactions

- [x] Signature verification (ECDSA, From matches pubkey).
- [x] Nonce enforced (mempool and state).
- [x] Balance check before adding to mempool and in block application.
- [x] Min fee enforced (anti-spam).
- [ ] Consider rate limiting on `/tx` API (currently none).

### Blocks

- [x] Block hash and Merkle root verified.
- [x] Validator for height matches block producer (PoS).
- [x] Tx signatures and rules applied in state.
- [ ] Block timestamp not validated (could allow slight time drift).

### Consensus / validators

- [x] Validator set from genesis; round-robin by height.
- [x] Slashing: double-sign detection (in-memory per node), stake zeroed, removed from set.
- [ ] No evidence propagation: if one node sees double-sign, others may not until they see both blocks.
- [ ] Validator key stored as JSON on disk; ensure file permissions (0600) and no key in logs.

### P2P

- [x] Genesis hash in Hello (wrong network rejected by design if we check; currently not enforced on receive).
- [x] Blocks and txs validated before apply/broadcast.
- [ ] No TLS: traffic is plaintext. Mainnet should use secure transport or run in private network.
- [ ] No peer authentication: any peer can connect and send messages (validation still applies).

### State and replay

- [x] Reorg replays from genesis; no arbitrary state injection.
- [x] Genesis allocation and validators fixed in genesis file; tampering requires replacing genesis.

### Key handling

- [x] Validator key: PEM/JSON on disk, not logged.
- [ ] No HSM or hardware key support.
- [ ] Send command: key loaded for signing; ensure process memory is not dumped to disk.

## Recommendations before mainnet

1. **External audit:** Have consensus, crypto, and P2P reviewed by a third party.
2. **Genesis audit:** Verify allocation sums to 21B, addresses and validator set are correct and intended.
3. **Parameter review:** Confirm block time, max block size, min fee, min stake for production load and security.
4. **Operational security:** Validator keys on dedicated machines; restrict API and P2P to trusted networks or add auth/TLS.
5. **Monitoring:** Use `/metrics` and alerts for chain growth, peer count, and block time.
