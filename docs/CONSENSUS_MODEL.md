# Hanticoin Consensus: Mathematical Model

Formal specification of the Hanticoin Proof-of-Stake consensus protocol. This document is structured for academic review and uses mathematical notation for precision.

**Scope:** Hanticoin is a standalone Layer-1 blockchain for fast, low-fee payments. Fixed supply 21,000,000,000 HTC; 6 decimal precision (1 HTC = 10⁶ smallest units). No smart contracts.

---

## 1. System Model

### 1.1 Notation

| Symbol | Meaning |
|--------|--------|
| \(\mathbb{N}\) | Set of all nodes in the network |
| \(\mathcal{V}\) | Validator set, \(\mathcal{V} \subseteq \mathbb{N}\) |
| \(n\) | \(|\mathbb{N}|\) (total nodes) |
| \(k\) | \(|\mathcal{V}|\) (number of validators), \(k \geq 1\) |
| \(r \in \mathbb{Z}_{\geq 0}\) | Round index (block height) |
| \(\sigma_r\) | State after applying blocks up to and including round \(r\) |

### 1.2 Network Model

- **Timing:** Partially synchronous. Message delays are bounded by some unknown \(\Delta\) after some unknown Global Stabilization Time (GST). Before GST, delays may be arbitrary.
- **Communication:** Point-to-point over TCP. We assume an authenticated channel abstraction (in practice: TCP + optional future TLS). Messages are delivered at most once; no reliable broadcast primitive is assumed—blocks and transactions are propagated by best-effort gossip and request-response (GetBlocks/Blocks).
- **Node set \(\mathbb{N}\):** Finite set of participants. Each node has a unique identity (derived from public key / address). Nodes may join or leave; the protocol does not assume a fixed \(n\).
- **Validator subset \(\mathcal{V}\):** Ordered list of \(k\) validators. \(\mathcal{V} = (v_0, v_1, \ldots, v_{k-1})\) where each \(v_i\) is an address (20-byte identifier). Membership and order are defined by genesis and updated only by slashing (removal). \(\mathcal{V}\) is stored in the replicated state and is consistent for all honest nodes that agree on the same chain.

### 1.3 Adversarial Model

- **Byzantine nodes:** An adversary may control a subset \(\mathcal{B} \subseteq \mathbb{N}\). Byzantine nodes may deviate arbitrarily from the protocol (drop, delay, forge, or reorder messages).
- **Validator adversary:** The adversary may control a subset of validators \(\mathcal{V}_{\mathcal{B}} = \mathcal{V} \cap \mathcal{B}\). We assume **honest majority of validators**: \(|\mathcal{V}_{\mathcal{B}}| < \frac{k}{2}\) (strictly less than half of the validator set).
- **No long-range attacks in scope:** We do not model key compromise of past validator keys; slashing is defined for equivocation within the same round only.

### 1.4 Safety and Liveness

- **Safety (consistency):** If two honest nodes have finalized (accepted and applied) blocks at the same height \(r\), then those blocks are identical. Equivalently: honest nodes do not commit conflicting blocks at the same round.
- **Liveness:** If the network is partially synchronous and a strict majority of validators is honest, then from some time onward the chain grows: for every sufficiently large \(r\), some block at height \(r\) is eventually accepted by all honest nodes (under the fork-resolution rule below).

### 1.5 Maximum Tolerated Byzantine Fraction

- **Validators:** At most \(\lceil k/2 \rceil - 1\) validators may be Byzantine; i.e. the number of honest validators is at least \(\lfloor k/2 \rfloor + 1\). The protocol does not use explicit voting; safety is achieved by the deterministic leader rule and the longest-chain rule. Liveness requires that the leader for round \(r\) is honest infinitely often (which is guaranteed if more than half of the validators are honest and rounds are assigned in a fixed order).

---

## 2. Stake Model

### 2.1 Total Supply and Units

- **Total supply (in smallest units):**  
  \[
  S = 21\,000\,000\,000 \times 10^6 = 21\times 10^{15}.
  \]  
  No minting; \(S\) is constant. All quantities below are in smallest units unless stated otherwise.

- **Decimals:** 6. One HTC = \(10^6\) smallest units.

### 2.2 Stake Distribution

- **Validator stake:** For each validator \(v_i \in \mathcal{V}\), \(s_i \geq 0\) denotes its staked amount. Stakes are stored in state and updated only by slashing (set to zero and removal from \(\mathcal{V}\)).
- **Minimum stake:** A node can be in \(\mathcal{V}\) only if \(s_i \geq S_{\min}\). By default \(S_{\min} = 5\,000\,000 \times 10^6\) (5 million HTC).
- **Validator weight (for reference):**  
  \[
  w_i = \frac{s_i}{\sum_{j \in \mathcal{V}} s_j}, \quad \sum_{i \in \mathcal{V}} w_i = 1.
  \]  
  In Hanticoin’s current design, **leader selection is not probability-weighted by stake**; it is deterministic round-robin by index (see §3). The weight \(w_i\) is still well-defined for economic analysis (e.g. cost of corruption).

### 2.3 Stake Lock and Unbonding

- **Stake lock:** Validator stake is implicitly locked while the node is in \(\mathcal{V}\). Reducing stake or leaving \(\mathcal{V}\) is not modeled in the current protocol (no explicit unbonding transaction).
- **Unbonding period:** Not specified in the current implementation; can be set to zero or to a parameter \(U\) (number of blocks) in a future extension. Here we take \(U = 0\) unless otherwise stated.

---

## 3. Block Production Algorithm

### 3.1 Round Index

- **Round:** Each round \(r \in \mathbb{Z}_{\geq 0}\) corresponds to one block height. Round \(0\) is the genesis block; round \(r \geq 1\) is the \(r\)-th block after genesis.

### 3.2 Leader Selection Function

Leader selection is **deterministic** and **index-based** (round-robin over the ordered validator set):

\[
\mathrm{Leader}(r) = v_{r \bmod k},
\quad
\mathcal{V} = (v_0, v_1, \ldots, v_{k-1}),\quad k = |\mathcal{V}|.
\]

So the leader for round \(r\) is the validator at index \(r \bmod k\). There is no randomness and no dependence on stake weights; only membership and order of \(\mathcal{V}\) matter. This yields exactly one allowed leader per round for a given \(\mathcal{V}\).

### 3.3 Block Creation and Acceptance

- **Creation:** The leader of round \(r\) builds a block \(B_r\) containing: height \(r\), timestamp, previous block hash, Merkle root of transactions, its own address as \(\mathrm{ValidatorAddress}\), list of transactions, and a signature over the block hash.
- **Acceptance:** A node accepts \(B_r\) only if:
  1. \(\mathrm{ValidatorAddress}(B_r) = \mathrm{Leader}(r)\),
  2. Block hash and Merkle root are valid,
  3. Parent is the node’s current tip (or the node performs reorg under the fork rule below).

No voting or multi-step BFT phase; single-block proposal and deterministic eligibility.

---

## 4. State Transition Function

### 4.1 State Space

State at round \(r\) is \(\sigma_r = (\mathcal{B}_r, \mathcal{A}_r, \mathcal{N}_r, \mathcal{V}_r, \mathcal{S}_r)\) where:

- \(\mathcal{B}_r\): balances (address \(\mapsto\) nonnegative integer),
- \(\mathcal{N}_r\): nonces (address \(\mapsto\) nonnegative integer),
- \(\mathcal{V}_r\): ordered validator set (list of addresses),
- \(\mathcal{S}_r\): validator stakes (address \(\mapsto\) nonnegative integer).

We require \(\sum_{a} \mathcal{B}_r(a) \leq S\) and that \(\mathcal{V}_r\) and \(\mathcal{S}_r\) are consistent (only addresses in \(\mathcal{V}_r\) have \(\mathcal{S}_r(\cdot) \geq S_{\min}\) and are used in \(\mathrm{Leader}(\cdot)\)).

### 4.2 Transition

\[
\sigma_{r+1} = \Gamma(\sigma_r, B_r).
\]

\(\Gamma\) is **deterministic**: given \(\sigma_r\) and block \(B_r\), the next state is uniquely defined.

### 4.3 Transaction Validation (within \(\Gamma\))

For each transaction \(\mathrm{tx}\) in \(B_r\) (in order):

- **Signature:** \(\mathrm{Verify}(\mathrm{tx}) = \mathrm{true}\) (ECDSA over \(\mathrm{Hash}(\mathrm{tx})\); \(\mathrm{From}(\mathrm{tx})\) equals the address of the signer).
- **Replay:** \(\mathrm{Nonce}(\mathrm{tx}) = \mathcal{N}_r(\mathrm{From}(\mathrm{tx}))\).
- **Balance:** \(\mathcal{B}_r(\mathrm{From}(\mathrm{tx})) \geq \mathrm{Amount}(\mathrm{tx}) + \mathrm{Fee}(\mathrm{tx})\).
- **Fee:** \(\mathrm{Fee}(\mathrm{tx}) \geq F_{\min}\) (e.g. \(F_{\min} = 100\)).

If any check fails, that transaction is **skipped** (not applied); the block is still valid. (In the mempool, only transactions satisfying these checks are admitted.)

### 4.4 Balance and Nonce Updates

Let \(\mathrm{addr}_L = \mathrm{ValidatorAddress}(B_r)\). For each applied \(\mathrm{tx}\):

\[
\mathcal{B}'(\mathrm{From}) \leftarrow \mathcal{B}'(\mathrm{From}) - \mathrm{Amount}(\mathrm{tx}) - \mathrm{Fee}(\mathrm{tx}),
\]
\[
\mathcal{B}'(\mathrm{To}) \leftarrow \mathcal{B}'(\mathrm{To}) + \mathrm{Amount}(\mathrm{tx}),
\]
\[
\mathcal{N}'(\mathrm{From}) \leftarrow \mathcal{N}(\mathrm{From}) + 1.
\]

Then fee distribution:

\[
\mathcal{B}'(\mathrm{addr}_L) \leftarrow \mathcal{B}'(\mathrm{addr}_L) + \sum_{\mathrm{tx} \in B_r} \mathrm{Fee}(\mathrm{tx}).
\]

(\(\mathcal{B}'\) is the balance after applying all transactions in \(B_r\); the sum is over applied transactions only.) Final state: \(\mathcal{B}_{r+1} = \mathcal{B}'\), \(\mathcal{N}_{r+1} = \mathcal{N}'\), \(\mathcal{V}_{r+1}\) and \(\mathcal{S}_{r+1}\) as in \(\sigma_r\) unless updated by slashing (see §6).

### 4.5 Determinism

For the same \(\sigma_r\) and the same ordered list of transactions (and same applicability decisions), \(\Gamma(\sigma_r, B_r)\) is uniquely defined. All honest nodes applying the same \(B_r\) from the same \(\sigma_r\) obtain the same \(\sigma_{r+1}\).

---

## 5. Finality Model

### 5.1 No BFT Voting

Hanticoin does **not** use pre-vote/pre-commit or supermajority quorum certificates. Finality is **probabilistic/de facto** based on depth and fork resolution.

### 5.2 Block Confirmation Depth

- **Confirmation depth \(k_{\mathrm{conf}}\):** A block at height \(r\) is considered confirmed after \(k_{\mathrm{conf}}\) subsequent blocks have been accepted on top of it. In the spec we take \(k_{\mathrm{conf}} = 1\) or \(2\) (1–2 blocks).
- **Safety condition (practical):** Once an honest node has accepted a chain whose tip is at height \(r + k_{\mathrm{conf}}\), and no reorg has occurred, the block at height \(r\) is treated as final for that node. Reorgs are possible only if a **longer** chain (by height) is received and validated.

### 5.3 Fork Resolution Rule

- **Longest chain (by height):** Given two chains \(C_1, C_2\) with tips at heights \(h_1, h_2\), the node adopts the chain with **larger height** (no cumulative difficulty). Ties (same height) are broken by the first-received or by a deterministic rule (e.g. by tip hash).
- **Reorg:** If the node currently has tip at height \(h\) and receives a chain whose tip is at height \(h' > h\), it fetches the full competing chain from genesis, validates every block (hash, Merkle, leader), then replaces state by \(\sigma_0\) (genesis) and applies blocks in order up to the new tip (e.g. \(\mathrm{ResetAndReplay}\)), and sets the new tip. No extra “finality gadget”; safety relies on the uniqueness of the leader per round and the longest-chain rule.

### 5.4 Liveness Condition

Under partial synchrony and honest majority of validators, the leader for infinitely many rounds is honest. Those rounds produce blocks that are propagated and accepted by all honest nodes, so the chain grows indefinitely after GST.

---

## 6. Slashing Conditions

### 6.1 Equivocation (Double-Signing)

**Equivocation at round \(r\):** A validator \(v\) has **equivocated** if two distinct blocks \(B_r^{(1)}, B_r^{(2)}\) are observed such that:

\[
\mathrm{ValidatorAddress}(B_r^{(1)}) = \mathrm{ValidatorAddress}(B_r^{(2)}) = v,
\quad
\mathrm{Hash}(B_r^{(1)}) \neq \mathrm{Hash}(B_r^{(2)}).
\]

(Height and validator address are the same; block hashes differ.)

### 6.2 Detection

Each node maintains a local structure: for each round \(r\), the first accepted block from validator \(v\) at round \(r\) is recorded. If a second block at the same \(r\) from the same \(v\) with a different hash is received, the node **detects equivocation** and triggers slashing for \(v\).

### 6.3 Slashing Penalty Function

\[
\mathrm{Penalty}(v) = \alpha \cdot s_v.
\]

For Hanticoin: **\(\alpha = 1\)** (full slash). So:

\[
\mathcal{S}_{r+1}(v) \leftarrow 0,
\quad
v \text{ removed from } \mathcal{V}.
\]

The validator is removed from the ordered set \(\mathcal{V}\); subsequent rounds use a reduced set (and \(k\) decreases until/unless new validators are added by a future mechanism). The slashed stake is not redistributed; it is effectively burned (balance of \(v\) for staking is zeroed in the validator state; any separate “stake account” balance is set to zero as per implementation).

### 6.4 Slashing Coefficient

\[
\alpha = 1.
\]

---

## 7. Economic Security Model

### 7.1 Minimum Stake to Participate

A validator must have stake \(s_i \geq S_{\min}\). Default: \(S_{\min} = 5\,000\,000 \times 10^6\) smallest units (5M HTC).

### 7.2 Cost of Corruption (Byzantine Majority)

To control the sequence of leaders (e.g. to censor or reorg), an adversary would need to control **more than half** of the validator set. With \(k\) validators and stakes \(s_1,\ldots,s_k\), the **minimum stake** that must be controlled to have a majority (by count) is more than half of the validators. If the adversary controls a set \(\mathcal{V}_{\mathcal{B}}\) of validators, the cost is at least the sum of their stakes; to have \(|\mathcal{V}_{\mathcal{B}}| \geq \lfloor k/2 \rfloor + 1\), that sum is at least the sum of the smallest \(\lfloor k/2 \rfloor + 1\) stakes. So:

\[
\mathrm{Attack\_Stake} \geq \sum_{i \in \mathrm{smallest}\; \lfloor k/2 \rfloor + 1} s_i.
\]

### 7.3 Attack Profitability (Inequality)

We require that for any rational adversary:

\[
\mathrm{Attack\_Cost} > \mathrm{Potential\_Gain}.
\]

- **Attack cost:** Opportunity cost of stake (locked capital) plus slashing risk: if equivocation is detected, \(\mathrm{Penalty} = s_i\) for each slashed validator. So the expected cost includes the probability of detection times the slashed amount.
- **Potential gain:** Short-term gain from double-spend or censorship (bounded by the value at risk in the reorg window and the cost of mounting the attack). No block reward (only fees); so gain is limited to extracted fees and double-spend value.

Formally, if the adversary controls validators with total stake \(S_{\mathcal{B}}\), and detection probability is \(p\), then a necessary condition for security is:

\[
p \cdot S_{\mathcal{B}} + \text{(opportunity cost)} > \text{(max extractable value)}.
\]

### 7.4 Long-Term Incentive Compatibility

- **Honest behavior:** Producing the correct block for one’s round yields fee income and avoids slashing.
- **Equivocation:** Yields zero fee for the duplicate block and leads to \(\alpha = 1\) slash. So \(\mathbb{E}[\mathrm{reward}|\mathrm{equivocate}] \ll \mathbb{E}[\mathrm{reward}|\mathrm{honest}]\) when detection is likely.
- **No explicit staking reward:** Only fees. Validators are motivated by fee revenue and by avoiding removal from \(\mathcal{V}\).

---

## 8. Network Latency and Throughput Model

### 8.1 Block Time

- **Target block interval:** \(T_b = 4\) seconds (configurable; e.g. 3–5 s in spec).
- **Interpretation:** The block producer creates at most one block per \(T_b\); if the leader is honest and the network is live, a new block is expected every \(T_b\) on average.

### 8.2 Throughput

- **Per-block transaction capacity:** Bounded by block size (e.g. \(M_b \approx 10^6\) bytes) and average tx size. Let \(\bar{\ell}\) be average serialized tx size. Then:
  \[
  \mathrm{TxPerBlock} \leq \left\lfloor \frac{M_b}{\bar{\ell}} \right\rfloor.
  \]
- **Expected transaction throughput:**
  \[
  \mathrm{Throughput} \leq \frac{\mathrm{TxPerBlock}}{T_b} \quad \text{[tx/s]}.
  \]

### 8.3 Message Complexity per Round

- **Normal operation:** One new block per round. Propagation is best-effort: each node that receives a block may forward it to its peers. In a network of degree \(d\), order of messages per block is \(O(n)\) or \(O(nd)\) depending on whether we count sends or total message hops. No multi-phase voting, so **one block message per round** per link in the propagation tree.
- **Sync:** GetBlocks/Blocks: one request and one response per peer per sync session; response size is \(O(\mathrm{chain length})\).

### 8.4 Latency Bound (Normal Conditions)

Under partial synchrony, after GST with delay bound \(\Delta\): once the leader broadcasts \(B_r\), all honest nodes receive it within time \(\Delta\) (or a small multiple thereof). So **confirmation latency** for a transaction included in \(B_r\) is at most about \(T_b + \Delta\) for one block, and \(k_{\mathrm{conf}} \cdot T_b + \Delta\) for \(k_{\mathrm{conf}}\)-deep confirmation.

---

## 9. Security Proof Sketch

### 9.1 Safety

- **Claim:** Two honest nodes do not commit different blocks at the same height.
- **Idea:** For each height \(r\), at most one validator is allowed to produce a block (\(\mathrm{Leader}(r)\)). Honest nodes accept only blocks whose \(\mathrm{ValidatorAddress}\) equals \(\mathrm{Leader}(r)\). So if two different blocks at \(r\) exist, at least one is from a Byzantine validator (equivocation). Honest nodes that see equivocation slash that validator; they do not “commit” both blocks. Any honest node that has already accepted one block at \(r\) will not switch to another block at \(r\) unless it performs a reorg to a **longer** chain. The longest-chain rule is applied to **heights**; so the first time an honest node commits to a block at \(r\), it is the only block at \(r\) that it will accept unless it reorgs to a chain that includes a different block at \(r\). After reorg, state is deterministic and the new chain has a single block per height. So at any moment, for any height, an honest node has at most one block at that height in its view of the canonical chain.
- **Assumptions:** Deterministic leader, honest nodes use the same \(\mathcal{V}\) (from same genesis and same slashing events), and message authentication (no forged blocks from non-leaders).

### 9.2 Liveness

- **Claim:** Under partial synchrony and honest majority of validators, the chain grows indefinitely.
- **Idea:** There are infinitely many rounds. In at least one out of every \(k\) consecutive rounds, the leader is honest (because \(|\mathcal{V}_{\mathcal{B}}| < k/2\)). After GST, the honest leader’s block is received by all honest nodes within bounded time. Honest nodes accept it (it extends their tip) and update their state. So from some round onward, every \(k\) rounds at least one new block is added by an honest leader. Hence the chain grows.
- **Assumptions:** Partial synchrony after GST, honest majority of validators, and that the network (graph) eventually delivers messages from the leader to all honest nodes.

### 9.3 Failure Scenarios

- **Byzantine majority of validators:** Safety can be violated (two different blocks at same height accepted by different honest nodes if they see different order of blocks). Liveness can be violated (Byzantine leaders can withhold blocks).
- **Network partition:** If partition lasts longer than GST assumption, nodes in different partitions may follow different chains; after partition heals, longest-chain rule reconciles.
- **Slashing false positives:** If equivocation detection is wrong (e.g. bug), an honest validator could be slashed; the model assumes correct implementation of the equivocation condition.

---

## 10. Conclusion

### 10.1 Security Guarantees

- **Safety:** Under honest majority of validators and deterministic leader rule, honest nodes do not commit conflicting blocks at the same height; reorgs are limited to adopting a longer chain and replaying state.
- **Liveness:** Under partial synchrony and honest majority, the chain grows indefinitely (at least one honest leader every \(k\) rounds).
- **Slashing:** Full penalty (\(\alpha = 1\)) for equivocation; no BFT voting, so guarantees are with respect to the longest-chain and single-leader-per-round rules.

### 10.2 Economic Stability

- Fixed supply \(S\); no minting. Validator revenue is fees only.
- Cost of mounting a majority attack is at least the stake of the smallest majority set of validators; slashing makes equivocation unprofitable when detection is likely.
- Incentive compatibility: honest production is rewarded (fees), equivocation is penalized (full slash).

### 10.3 Scalability Assumptions

- Throughput is bounded by block size and block interval (\(M_b\), \(T_b\)). No sharding or L2 in scope.
- Message complexity is linear in the number of nodes for block propagation. Latency is modeled as bounded after GST.

---

*Document version: 1.0. Aligned with Hanticoin implementation (round-robin leader, full slashing, longest-chain reorg, no BFT voting).*
