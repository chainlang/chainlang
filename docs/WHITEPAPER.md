# Hanticoin: A Proof-of-Stake Payment Blockchain

**Formal specification and security analysis**

---

## 1. Abstract

**Problem.** Decentralized payment settlement requires a consistent, live ledger under a fixed trust assumption and without reliance on a single authority.

**Proposed protocol.** Hanticoin is a Layer-1 Proof-of-Stake blockchain with an account-based state model, deterministic round-robin leader selection, and no smart contracts. The ledger is an ordered sequence of blocks; state transitions are deterministic. Validators are selected by index; block acceptance is by leader eligibility and longest-chain fork choice. There is no BFT voting phase.

**Security model.** Partially synchronous network; Byzantine adversary controlling fewer than half of the validators. Safety: no two honest nodes finalize different blocks at the same height. Liveness: under honest majority, the chain grows indefinitely after a global stabilization time.

**Performance.** Target block interval \(T_b = 4\) s; throughput is bounded by block size and \(T_b\). Message complexity per round is \(O(n)\) for block propagation. Confirmation latency is \(O(k_{\mathrm{conf}} \cdot T_b + \Delta)\) where \(k_{\mathrm{conf}} \in \{1,2\}\) and \(\Delta\) is the delay bound after stabilization.

**Economic guarantees.** Fixed total supply \(S = 21\times 10^9\) HTC (6 decimal precision). No minting; validator revenue is transaction fees only. Full slashing (\(\alpha = 1\)) for equivocation. Attack cost exceeds potential gain when detection probability and stake at risk are sufficient; honest behavior is incentive-compatible.

---

## 2. Introduction

### 2.1 Payment settlement problem

We consider the problem of **payment settlement** in a distributed setting: a set of parties wish to maintain a shared ledger of account balances and to update it via **transfers** such that (i) no unit of currency is spent twice (consistency), and (ii) valid transfer requests are eventually reflected in the ledger (liveness). Formally, the ledger is a sequence of state transitions; each transition is induced by a block of transactions. The problem is to choose, in a decentralized way, which blocks are accepted and in which order, so that all honest participants agree on the same sequence and the system makes progress.

### 2.2 Trust assumptions

We assume that **honest participants** follow the protocol and that their fraction is sufficient to enforce safety and liveness (we will require an honest majority of validators). We do **not** assume a trusted central party. We assume **cryptographic hardness**: collision-resistant hashing and existentially unforgeable signatures. The network is **partially synchronous**: after some unknown time, message delays are bounded by an unknown constant.

### 2.3 Motivation for a dedicated payment blockchain

A blockchain restricted to **transfers** (no general-purpose computation) simplifies the state model, reduces implementation and audit surface, and avoids reentrancy and oracle issues inherent in smart contracts. Predictable state transitions and fixed supply allow clear **monetary policy** and **economic analysis**. Targeting **fast, low-fee payments** (e.g. in regions with limited banking infrastructure) favors a minimal protocol with short block times and no heavy consensus overhead (e.g. no multi-round BFT voting in the base design).

### 2.4 Limitations of general-purpose platforms

General-purpose smart-contract platforms optimize for expressiveness and composability; their consensus and execution layers are more complex, and their fee markets are driven by contract execution. For pure payments, a dedicated chain can offer a simpler threat model, lower fees, and formal guarantees tailored to transfers and fixed supply.

---

## 3. System Model

### 3.1 Participants

- **Node set \(N\):** A finite set of participants (nodes). Each node has a unique identity derived from a public key; we identify nodes by their **address** (e.g. 20-byte digest of the public key). The size \(n = |N|\) may change over time (join/leave); the protocol does not assume a fixed \(n\).

- **Validator subset \(V \subseteq N\):** An **ordered** list of \(k\) validators, \(V = (v_0, v_1, \ldots, v_{k-1})\), \(k \geq 1\). Only validators are eligible to produce blocks. Membership and order are defined at genesis and updated only by **slashing** (removal). We treat \(V\) as part of the replicated state so that all honest nodes agreeing on the same chain share the same \(V\).

### 3.2 Adversary \(\mathcal{A}\)

- **Byzantine nodes:** The adversary \(\mathcal{A}\) controls a subset \(B \subseteq N\). Byzantine nodes may deviate arbitrarily (drop, delay, forge, reorder messages).

- **Byzantine validators:** \(\mathcal{V}_B = V \cap B\). We assume **honest majority of validators**:
  \[
  |\mathcal{V}_B| < \frac{k}{2}.
  \]
  So the number of honest validators is at least \(\lfloor k/2 \rfloor + 1\).

- **Maximum Byzantine fraction \(f\):** Among validators, at most \(f = \lceil k/2 \rceil - 1\) may be Byzantine; equivalently, the honest validator fraction is strictly greater than \(1/2\).

### 3.3 Network model

- **Timing:** **Partially synchronous.** There exists an unknown **Global Stabilization Time (GST)** and an unknown delay bound \(\Delta\) such that every message sent after GST is delivered within time \(\Delta\). Before GST, delays may be arbitrary.

- **Communication:** Point-to-point (e.g. TCP). We assume an **authenticated channel** abstraction (sender identity is known). Messages are delivered at most once. Block and transaction propagation is by **best-effort gossip** and request-response (e.g. GetBlocks/Blocks); no reliable broadcast primitive is assumed.

### 3.4 Safety and liveness

- **Safety (consistency):** If two honest nodes have each accepted and applied a block at height \(r\), then those blocks are **identical**. Equivalently, honest nodes do not commit conflicting blocks at the same round.

- **Liveness:** Under partial synchrony and honest majority of validators, from some time onward the chain grows: for every sufficiently large round \(r\), some block at height \(r\) is eventually accepted by all honest nodes (under the fork-choice rule in §8).

---

## 4. Cryptographic Primitives

### 4.1 Hash function

\[
H : \{0,1\}^* \rightarrow \{0,1\}^{256}.
\]

We use a single hash function (e.g. SHA-256) for:
- Transaction digest (input to signing),
- Block digest (linkage and signing),
- Merkle tree construction.

**Assumption (collision resistance):** It is computationally infeasible to find \(x \neq x'\) such that \(H(x) = H(x')\).

### 4.2 Digital signature scheme

A triple of algorithms \((\mathsf{KeyGen}, \mathsf{Sign}, \mathsf{Verify})\):

- \(\mathsf{KeyGen}() \rightarrow (pk, sk)\): key generation.
- \(\mathsf{Sign}(sk, m) \rightarrow \sigma\): signature on message \(m\).
- \(\mathsf{Verify}(pk, m, \sigma) \rightarrow \{0,1\}\): verification.

**Assumption (EUF-CMA):** Existentially unforgeable under chosen-message attack. We use signatures for transactions (signing \(H(\mathrm{tx})\)) and for blocks (validator signs \(H(B)\)). The **address** of an account is derived from the public key (e.g. first 20 bytes of \(H(pk)\)).

### 4.3 Merkle tree

Given a list of hashes \((h_1, h_2, \ldots, h_m)\), the **Merkle root** is the single hash at the root of a binary tree: leaves are the \(h_i\); each internal node is \(H(\mathrm{left} \| \mathrm{right})\); if a level has odd length, the last element is duplicated. Empty list \(\rightarrow\) zero hash; single element \(\rightarrow\) \(H(h_1)\). We use this to commit to the list of transaction hashes in a block; verification is by recomputing the root from the block’s transactions and comparing to the block header.

---

## 5. Ledger Model

### 5.1 Ledger as sequence of blocks

The **ledger** is an ordered sequence of blocks:

\[
\mathcal{L} = (B_0, B_1, \ldots, B_t),
\]

where \(B_0\) is the **genesis block** (fixed by protocol/configuration) and \(t \geq 0\) is the current **tip height**. Each \(B_i\) for \(i \geq 1\) extends \(B_{i-1}\).

### 5.2 Block structure

We define block \(B_i\) (for \(i \geq 1\)) formally as:

\[
B_i = (h_{i-1}, \mathsf{tx}_i, r_i, \mathsf{addr}_L, \mathsf{MR}_i, \sigma_i, h_i),
\]

where:

- \(h_{i-1} \in \{0,1\}^{256}\): hash of the previous block \(B_{i-1}\) (parent pointer).
- \(\mathsf{tx}_i\): ordered list of transactions in the block.
- \(r_i \in \mathbb{Z}_{\geq 0}\): **round index** (height); \(r_i = i\).
- \(\mathsf{addr}_L \in \{0,1\}^{160}\): **validator address** (block producer).
- \(\mathsf{MR}_i = \mathsf{MerkleRoot}(\mathsf{tx}_i)\): Merkle root of transaction hashes.
- \(\sigma_i\): signature over the **block payload** (see below) by the validator’s key.
- \(h_i = H(\mathrm{payload}_i)\): block hash; payload is a canonical encoding of \((r_i, \mathsf{timestamp}_i, h_{i-1}, \mathsf{MR}_i, \mathsf{addr}_L, (H(\mathrm{tx}_1), \ldots, H(\mathrm{tx}_m)))\). Signature and \(h_i\) are not part of the hashed payload.

Genesis \(B_0\) has no parent; it contains initial allocation and validator set and is agreed off-chain (e.g. in a genesis file).

### 5.3 State transition function

State is updated deterministically:

\[
\mathsf{State}_{t+1} = \Gamma(\mathsf{State}_t, B_t).
\]

\(\Gamma\) takes the current state (balances, nonces, validator set, stakes) and block \(B_t\), applies the block’s transactions and fee distribution (see §6), and outputs the new state. **Determinism:** For the same \(\mathsf{State}_t\) and \(B_t\), \(\mathsf{State}_{t+1}\) is uniquely defined. All honest nodes applying the same \(\mathcal{L}\) obtain the same state.

---

## 6. Transaction Model

### 6.1 Transaction structure

A **transaction** is a tuple:

\[
\mathrm{Tx} = (\mathsf{from}, \mathsf{to}, \mathsf{amount}, \mathsf{fee}, \mathsf{nonce}, \mathsf{pk}, \sigma),
\]

where \(\mathsf{from}, \mathsf{to}\) are addresses (20 bytes), \(\mathsf{amount}, \mathsf{fee}, \mathsf{nonce}\) are nonnegative integers, \(\mathsf{pk}\) is the sender’s public key, and \(\sigma = \mathsf{Sign}(sk, H(\mathsf{from}, \mathsf{to}, \mathsf{amount}, \mathsf{fee}, \mathsf{nonce}))\). The **tx hash** used in the Merkle tree is \(H(\mathsf{from}, \mathsf{to}, \mathsf{amount}, \mathsf{fee}, \mathsf{nonce})\).

### 6.2 Validity predicate

We define a predicate \(\mathsf{Valid}(\mathrm{Tx}, \mathsf{State}) \rightarrow \{0,1\}\) that is 1 iff all of the following hold:

1. **Signature:** \(\mathsf{Verify}(\mathsf{pk}, H(\mathsf{from}, \mathsf{to}, \mathsf{amount}, \mathsf{fee}, \mathsf{nonce}), \sigma) = 1\) and \(\mathsf{Addr}(\mathsf{pk}) = \mathsf{from}\).
2. **Nonce:** \(\mathrm{Nonce}(\mathsf{from}) = \mathsf{nonce}\) in \(\mathsf{State}\).
3. **Balance:** \(\mathrm{Balance}(\mathsf{from}) \geq \mathsf{amount} + \mathsf{fee}\) in \(\mathsf{State}\).
4. **Fee:** \(\mathsf{fee} \geq F_{\min}\) (minimum fee, e.g. \(F_{\min} = 100\) in smallest units).

If \(\mathsf{Valid}(\mathrm{Tx}, \mathsf{State}) = 0\), the transaction is **invalid** in that state. In block application, invalid transactions in a block are **skipped** (not applied); the block remains valid. In the mempool, only transactions with \(\mathsf{Valid}(\mathrm{Tx}, \mathsf{State}) = 1\) are admitted.

### 6.3 Balance and nonce updates

For each **applied** transaction in order:

\[
\mathcal{B}(\mathsf{from}) \leftarrow \mathcal{B}(\mathsf{from}) - \mathsf{amount} - \mathsf{fee}, \quad
\mathcal{B}(\mathsf{to}) \leftarrow \mathcal{B}(\mathsf{to}) + \mathsf{amount}, \quad
\mathcal{N}(\mathsf{from}) \leftarrow \mathcal{N}(\mathsf{from}) + 1.
\]

Then **fee redistribution** to the block producer \(\mathsf{addr}_L\):

\[
\mathcal{B}(\mathsf{addr}_L) \leftarrow \mathcal{B}(\mathsf{addr}_L) + \sum_{\mathrm{tx} \in B} \mathsf{fee}(\mathrm{tx}).
\]

(The sum is over applied transactions only.) No other reward or minting.

---

## 7. Proof-of-Stake Consensus Protocol

### 7.1 Total supply and units

- **Total supply (in HTC):** \(S_{\mathrm{HTC}} = 21\times 10^9\) (21 billion HTC).
- **Smallest unit:** 1 HTC = \(10^6\) smallest units. So in smallest units:
  \[
  S = 21 \times 10^9 \times 10^6 = 21 \times 10^{15}.
  \]
  \(S\) is constant; no minting.

### 7.2 Stake distribution and validator weight

- **Stake:** For each validator \(v_i \in V\), \(s_i \geq 0\) denotes its staked amount (in smallest units). Stakes are stored in state; \(s_i\) is updated only by slashing (set to 0 and removal from \(V\)).
- **Minimum stake:** \(v_i \in V\) only if \(s_i \geq S_{\min}\). Default \(S_{\min} = 5\times 10^6 \times 10^6\) (5 million HTC in smallest units).
- **Validator weight:**
  \[
  w_i = \frac{s_i}{\sum_{j \in V} s_j}, \quad \sum_{i \in V} w_i = 1.
  \]
  Used for economic analysis (e.g. cost of corruption). **Leader selection in Hanticoin is not stake-weighted**; it is deterministic by index (see below).

### 7.3 Leader selection function

\[
\mathsf{Leader}(r) = F(r, V, s).
\]

In Hanticoin, \(F\) is **deterministic round-robin** over the ordered set \(V = (v_0, \ldots, v_{k-1})\):

\[
\mathsf{Leader}(r) = v_{r \bmod k}, \quad k = |V|.
\]

So there is **exactly one** allowed leader per round; no randomness and no dependence on \(s\) beyond membership in \(V\). This avoids stake grinding and simplifies analysis.

### 7.4 Round structure and block proposal

- **Round \(r\):** Corresponds to block height \(r\). Round 0 is genesis; round \(r \geq 1\) is the \(r\)-th block.
- **Proposal:** The leader \(\mathsf{Leader}(r)\) builds \(B_r\) (parent \(h_{r-1}\), transactions, timestamp, Merkle root, \(\mathsf{addr}_L = \mathsf{Leader}(r)\), signature), then broadcasts it.
- **Acceptance:** A node accepts \(B_r\) only if (i) \(\mathsf{addr}_L = \mathsf{Leader}(r)\), (ii) \(h_r = H(\mathrm{payload})\), (iii) \(\mathsf{MR}_r\) matches the Merkle root of the block’s transactions, (iv) parent is the node’s current tip (or the node performs reorg under the fork-choice rule).

### 7.5 Voting or attestation

Hanticoin **does not** use a separate voting or attestation phase. There is no pre-vote/pre-commit or quorum certificate. Block acceptance is by **leader eligibility** and **fork choice** only. Therefore there is **no supermajority condition** of the form “\(\geq 2/3\) stake” in the consensus step; safety and liveness rely on the deterministic leader and the longest-chain rule (see §8).

---

## 8. Finality and Fork Choice

### 8.1 Confirmation depth \(k\)

A block at height \(r\) is considered **confirmed** after \(k\) subsequent blocks have been accepted on top of it. We set \(k \in \{1, 2\}\) (1–2 blocks). So finality is **probabilistic / de facto**: once an honest node’s chain has tip at \(r + k\) and no reorg occurs, the block at \(r\) is treated as final for that node.

### 8.2 Fork choice rule \(\mathcal{F}\)

Given the set of known blocks and their parent relation, each node maintains a **canonical chain** by the **longest-chain rule** (by height):

\[
\mathcal{F}(\mathcal{L}_1, \mathcal{L}_2) \Rightarrow \text{choose } \mathcal{L}_1 \text{ iff } \mathsf{height}(\mathsf{tip}(\mathcal{L}_1)) > \mathsf{height}(\mathsf{tip}(\mathcal{L}_2)).
\]

Ties (same height) are broken deterministically (e.g. by tip hash or first-seen). There is **no cumulative difficulty**; only block count (height) matters.

### 8.3 Reorg

If the node’s current tip is at height \(h\) and it receives a chain whose tip is at height \(h' > h\), the node **reorgs**: it fetches the full competing chain from genesis, validates every block (hash, Merkle, leader), then sets state to genesis and applies blocks in order up to the new tip (\(\mathsf{ResetAndReplay}\)), and sets the new tip. Safety is preserved because at each height there is at most one block accepted (leader uniqueness); reorg simply switches to a longer valid chain.

### 8.4 Safety and liveness (recap)

- **Safety:** Honest nodes do not commit two different blocks at the same height; the fork-choice rule and leader rule ensure a single block per height in the canonical chain.
- **Liveness:** Under partial synchrony and honest majority, infinitely many rounds have an honest leader; their blocks are delivered and accepted, so the chain grows. No BFT quorum is required; the condition \(|\mathcal{V}_B| < k/2\) suffices for progress.

---

## 9. Slashing and Accountability

### 9.1 Equivocation (double-signing)

**Equivocation at round \(r\):** A validator \(v\) has **equivocated** if there exist two distinct blocks \(B, B'\) such that:

\[
B \neq B', \quad \mathsf{Height}(B) = \mathsf{Height}(B') = r, \quad \mathsf{ValidatorAddress}(B) = \mathsf{ValidatorAddress}(B') = v.
\]

Equivalently, \(v\) signed two different block hashes at the same height.

### 9.2 Detection

Each node records, for each \((r, v)\), the first block at height \(r\) from validator \(v\) that it accepted. If it later receives another block at height \(r\) from \(v\) with a different hash, it **detects equivocation** and triggers slashing for \(v\).

### 9.3 Penalty function

\[
\mathsf{Penalty}_i = \alpha \cdot s_i.
\]

For Hanticoin: **\(\alpha = 1\)** (full slash). So \(\mathsf{Penalty}_i = s_i\); the validator’s stake is set to zero and the validator is **removed** from \(V\). The slashed stake is not redistributed (effectively burned). No partial slash in the base design.

### 9.4 Unbonding period \(T_u\)

The **unbonding period** \(T_u\) is the number of blocks (or time) after which a validator can leave the set or reduce stake without being subject to slashing for past behavior. In the current specification, stake is “locked” while in \(V\); explicit unbonding is not modeled. We may set \(T_u = 0\) or treat it as a parameter for future extensions (e.g. \(T_u \in \mathbb{Z}_{\geq 0}\) blocks).

---

## 10. Economic Security Analysis

### 10.1 Cost of corruption

To control a **majority of validators** (and thus the sequence of leaders), the adversary must control at least \(\lfloor k/2 \rfloor + 1\) validators. The **minimum stake** required is the sum of the smallest \(\lfloor k/2 \rfloor + 1\) stakes:

\[
\mathsf{AttackStake} \geq \sum_{i \in \text{smallest } \lfloor k/2 \rfloor + 1} s_i.
\]

We can express this in terms of total supply: if a fraction \(\theta\) of \(S\) (or of total staked supply) must be controlled to mount the attack, then:

\[
\mathsf{AttackCost} \geq \theta \cdot S \quad \text{(or } \theta \cdot \text{total staked)}.
\]

\(\theta\) depends on the distribution of stakes; in the worst case for the protocol, \(\theta\) is the minimum fraction that allows control of a majority of validators.

### 10.2 Profitability inequality

We require that for a rational adversary:

\[
\mathsf{AttackCost} > \mathsf{PotentialGain}.
\]

- **AttackCost:** Opportunity cost of capital (stake locked) plus **slashing risk**: if equivocation is detected, \(\mathsf{Penalty}_i = s_i\) per slashed validator. So \(\mathbb{E}[\mathsf{cost}] \geq p \cdot S_{\mathcal{B}} + \text{opportunity cost}\), where \(p\) is detection probability and \(S_{\mathcal{B}}\) is adversary’s staked amount.
- **PotentialGain:** Bounded by double-spend value in the reorg window plus any extracted fees. No block reward; gains are limited.

So a **necessary** condition for security is:

\[
p \cdot S_{\mathcal{B}} + \text{opportunity cost} > \text{max extractable value}.
\]

### 10.3 Incentive compatibility

We require that honest behavior is (in expectation) at least as profitable as attacking:

\[
\mathbb{E}[\mathsf{HonestReward}_i] \geq \mathbb{E}[\mathsf{ExpectedAttackGain}_i].
\]

- **Honest:** Produce the correct block for one’s round → receive fees; avoid slashing.
- **Equivocate:** Duplicate block yields no extra fee (only one chain is accepted) and triggers \(\alpha = 1\) slash. So \(\mathbb{E}[\mathsf{reward} \mid \mathsf{equivocate}] \ll \mathbb{E}[\mathsf{reward} \mid \mathsf{honest}]\) when detection probability is non-negligible.

No explicit staking reward (no emission); validators are motivated by fee revenue and by avoiding removal from \(V\).

---

## 11. Performance Analysis

### 11.1 Block interval \(T_b\)

**Target block time:** \(T_b = 4\) seconds (configurable in a range, e.g. 3–5 s). So under normal conditions, one block is produced every \(T_b\) on average when the leader is active.

### 11.2 Throughput

- **Per-block capacity:** Bounded by block size \(M_b\) (e.g. \(M_b \approx 10^6\) bytes) and average serialized transaction size \(\bar{\ell}\). Maximum transactions per block:
  \[
  \mathsf{TxPerBlock} \leq \left\lfloor \frac{M_b}{\bar{\ell}} \right\rfloor.
  \]
- **Throughput (transactions per second):**
  \[
  \mathsf{Throughput} \leq \frac{\mathsf{TxPerBlock}}{T_b} = O\left( \frac{M_b}{\bar{\ell} \cdot T_b} \right).
  \]

### 11.3 Message complexity per round

- **Block propagation:** One block per round. In a network of \(n\) nodes with best-effort gossip, each node forwards the block to its peers; total messages per round are \(O(n)\) (or \(O(n \cdot d)\) if \(d\) is degree). No multi-phase voting, so **one block message** per round dominates.
- **Sync:** Request-response (GetBlocks/Blocks): \(O(1)\) round-trips per sync session; response size \(O(\mathsf{chainlength})\).

### 11.4 Latency bounds (partial synchrony)

After GST with delay bound \(\Delta\): once the leader broadcasts \(B_r\), all honest nodes receive it within time \(\Delta\) (or a small constant multiple). So:

- **Inclusion latency:** A transaction included in \(B_r\) is first confirmed when \(B_r\) is accepted: at most \(T_b + \Delta\).
- **\(k\)-confirmation latency:** At most \(k \cdot T_b + \Delta\) (e.g. \(k = 1\) or \(2\)).

**Asymptotic:** Latency is \(O(T_b + \Delta)\); throughput is \(O(M_b / (\bar{\ell} \cdot T_b))\).

---

## 12. Security Proof Sketch

### 12.1 Safety (no two conflicting finalized blocks)

**Theorem (informal).** Under the assumptions that (i) at most one validator is allowed to produce a block at each height (\(\mathsf{Leader}(r)\) is unique), (ii) honest nodes accept only blocks with \(\mathsf{ValidatorAddress} = \mathsf{Leader}(r)\), and (iii) honest nodes use the same fork-choice rule (longest chain by height), no two honest nodes finalize different blocks at the same height.

**Sketch.** Suppose two honest nodes finalize blocks \(B \neq B'\) at height \(r\). Then both blocks have \(\mathsf{ValidatorAddress} = \mathsf{Leader}(r)\) (same validator \(v\)). So \(v\) produced two different blocks at \(r\) → equivocation. Honest nodes do not “finalize” both; they accept one and slash \(v\) on seeing the second. Moreover, the fork-choice rule selects a single chain; reorg replaces the chain entirely. So at any time, each honest node has at most one block at height \(r\) in its canonical chain. **Assumptions:** Deterministic leader, consistent \(V\) (same genesis and slashing), authenticated channels (no forged blocks from non-leaders).

### 12.2 Liveness (eventual confirmation)

**Theorem (informal).** Under partial synchrony (after GST) and honest majority of validators (\(|\mathcal{V}_B| < k/2\)), every round \(r\) eventually has a block accepted by all honest nodes (i.e. the chain grows).

**Sketch.** There are infinitely many rounds. In every consecutive \(k\) rounds, at least one round has an **honest** leader (because fewer than \(k/2\) validators are Byzantine). After GST, the honest leader’s block is delivered within time \(\Delta\). Honest nodes receive it, verify leader and hash/Merkle, and accept it (it extends their tip). So from some round onward, at least one block per \(k\) rounds is added. **Assumptions:** Partial synchrony, honest majority of validators, network connectivity so that the leader’s message reaches all honest nodes.

### 12.3 Resistance to long-range attacks

**Long-range attack:** An adversary with old validator keys creates an alternative history from an earlier height. Hanticoin does **not** implement a separate long-range attack mitigation (e.g. checkpointing or key-epoch bounds) in this specification. **Assumption:** We do not model compromise of past validator keys; slashing is defined for **equivocation in the same round** only. For a full treatment, one would add checkpoints or require validators to use key-rotation / epoch bounds so that old keys cannot be used to rewrite history.

### 12.4 Resistance to stake grinding

**Stake grinding:** An adversary tries to bias leader selection by manipulating some quantity (e.g. entropy) that feeds into the leader function. In Hanticoin, **\(\mathsf{Leader}(r) = v_{r \bmod k}\)** depends only on \(r\) and the **ordered set** \(V\). There is no randomness and no stake-weighted sampling. So there is **no stake grinding** in the leader selection itself. The only way to influence who is leader is to change \(V\) (e.g. by slashing), which is protocol-driven and not manipulable by grinding.

---

## 13. Monetary Policy Formalization

### 13.1 Fixed supply invariant

At every state \(\mathsf{State}_t\) resulting from a valid ledger \(\mathcal{L}\):

\[
\sum_{i} \mathcal{B}_t(a_i) = S.
\]

Sum is over all addresses with positive balance. **No minting:** New coins are never created. Genesis allocates a total of \(S\) across addresses; every transaction only moves coins (amount + fee from \(\mathsf{from}\), amount to \(\mathsf{to}\), fee to block producer). So the sum of all balances is invariant and equal to \(S\).

### 13.2 Fee redistribution rule

For each block \(B\) with producer \(\mathsf{addr}_L\) and applied transactions \(\mathsf{tx}_1, \ldots, \mathsf{tx}_m\):

\[
\mathcal{B}(\mathsf{addr}_L) \leftarrow \mathcal{B}(\mathsf{addr}_L) + \sum_{j=1}^{m} \mathsf{fee}(\mathsf{tx}_j).
\]

Fees are the **only** reward to validators. No portion is burned or sent elsewhere in the base design.

### 13.3 Reward emission function

\[
\mathsf{Emission}(r) = 0 \quad \forall r.
\]

There is no block reward or inflation. Validator revenue is exclusively from transaction fees.

---

## 14. Conclusion

### 14.1 Formal guarantees

- **Safety:** Under honest majority of validators and deterministic round-robin leader selection, honest nodes do not commit conflicting blocks at the same height. The longest-chain fork-choice rule and single-leader-per-round ensure a unique canonical chain per node at any time.
- **Liveness:** Under partial synchrony and honest majority, the chain grows indefinitely; at least one honest leader every \(k\) rounds produces a block that is accepted by all honest nodes.
- **Determinism:** State transition \(\Gamma\) is deterministic; all honest nodes applying the same ledger agree on the same state.

### 14.2 Economic stability

- **Fixed supply** \(S = 21\times 10^9\) HTC (6 decimals) with no emission. Fee-only validator revenue and full slashing (\(\alpha = 1\)) for equivocation yield a clear **attack-cost vs. gain** inequality and **incentive compatibility** for honest block production.
- **Applicability:** The design is tailored to **payment settlement**: simple account model, fast block time, low fee floor, and no smart-contract surface. It is suitable for use cases such as domestic and cross-border transfers in regions where predictability and low cost are priorities.

### 14.3 Limitations and future work

- Finality is **probabilistic** (1–2 blocks); no BFT finality gadget. Long-range attacks are out of scope in this document; checkpointing or key-epoch bounds can be added. Unbonding period \(T_u\) is left as a parameter (e.g. 0 in current implementation). Formal verification of the implementation and a full reduction-based security proof are directions for future work.

---

*Document version: 1.0. Hanticoin protocol specification; aligned with the implementation (round-robin leader, longest-chain reorg, full slashing, no BFT voting).*
