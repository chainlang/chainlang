package mempool

import (
	"errors"
	"sync"

	"hanticoin/core"
	"hanticoin/crypto"
	"hanticoin/state"
)

var (
	ErrMempoolFull         = errors.New("mempool full")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrFeeTooLow           = errors.New("fee below minimum")
	ErrBadNonce            = errors.New("invalid nonce")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrDuplicate           = errors.New("duplicate transaction")
)

// Mempool holds pending transactions.
type Mempool struct {
	mu    sync.RWMutex
	txs   []*core.Transaction
	max   int
	state *state.State
}

// New creates a mempool with optional state for balance/nonce checks.
func New(maxSize int, st *state.State) *Mempool {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &Mempool{txs: make([]*core.Transaction, 0), max: maxSize, state: st}
}

// Add adds a transaction if valid and not at capacity.
func (m *Mempool) Add(tx *core.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.txs) >= m.max {
		return ErrMempoolFull
	}
	if !tx.VerifySignature() {
		return ErrInvalidSignature
	}
	if !tx.SatisfiesMinFee() {
		return ErrFeeTooLow
	}
	if m.state != nil {
		bal, err := m.state.GetBalance(tx.From)
		if err != nil {
			return err
		}
		nonce, err := m.state.GetNonce(tx.From)
		if err != nil {
			return err
		}
		if tx.Nonce != nonce {
			return ErrBadNonce
		}
		if bal < tx.Amount+tx.Fee {
			return ErrInsufficientBalance
		}
	}
	// Dedupe by tx hash
	h := tx.TxHash()
	for _, t := range m.txs {
		if t.TxHash() == h {
			return ErrDuplicate
		}
	}
	m.txs = append(m.txs, tx)
	return nil
}

// Pending returns up to n transactions (for block building).
func (m *Mempool) Pending(n int) []*core.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n <= 0 || n > len(m.txs) {
		n = len(m.txs)
	}
	out := make([]*core.Transaction, n)
	copy(out, m.txs[:n])
	return out
}

// Remove removes transactions that appear in the given list (by hash).
func (m *Mempool) Remove(included []core.Transaction) {
	if len(included) == 0 {
		return
	}
	hashes := make(map[crypto.Hash]struct{})
	for i := range included {
		hashes[included[i].TxHash()] = struct{}{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	filtered := m.txs[:0]
	for _, tx := range m.txs {
		if _, ok := hashes[tx.TxHash()]; !ok {
			filtered = append(filtered, tx)
		}
	}
	m.txs = filtered
}

// Size returns current pending count.
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.txs)
}