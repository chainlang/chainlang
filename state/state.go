package state

import (
	"encoding/binary"
	"encoding/hex"
	"sync"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	prefixBalance = "b:"
	prefixNonce   = "n:"
)

// State holds account balances and nonces in LevelDB.
type State struct {
	mu   sync.RWMutex
	db   *leveldb.DB
	cfg  *config.ChainConfig
}

// Open opens or creates the state DB at path.
func Open(path string, cfg *config.ChainConfig) (*State, error) {
	if cfg == nil {
		cfg = config.DefaultChainConfig()
	}
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &State{db: db, cfg: cfg}, nil
}

// Close closes the database.
func (s *State) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func balanceKey(addr crypto.Address) []byte {
	return append([]byte(prefixBalance), addr[:]...)
}

func nonceKey(addr crypto.Address) []byte {
	return append([]byte(prefixNonce), addr[:]...)
}

// GetBalance returns balance for address (0 if not found).
func (s *State) GetBalance(addr crypto.Address) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := balanceKey(addr)
	val, err := s.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(val) < 8 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(val), nil
}

// GetNonce returns next nonce for address (0 if never used).
func (s *State) GetNonce(addr crypto.Address) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := nonceKey(addr)
	val, err := s.db.Get(key, nil)
	if err == leveldb.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if len(val) < 8 {
		return 0, nil
	}
	return binary.BigEndian.Uint64(val), nil
}

// setBalance and setNonce are used only during ApplyBlock (under batch).
func setBalance(batch *leveldb.Batch, addr crypto.Address, amount uint64) {
	key := balanceKey(addr)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, amount)
	batch.Put(key, buf)
}

func setNonce(batch *leveldb.Batch, addr crypto.Address, nonce uint64) {
	key := nonceKey(addr)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, nonce)
	batch.Put(key, buf)
}

// ApplyGenesis applies genesis allocation. Call once when initializing chain.
func (s *State) ApplyGenesis(allocation config.GenesisAllocation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch := new(leveldb.Batch)
	for addrHex, amount := range allocation {
		b, err := hex.DecodeString(addrHex)
		if err != nil || len(b) != crypto.AddressSize {
			continue
		}
		var addr crypto.Address
		copy(addr[:], b)
		setBalance(batch, addr, amount)
		setNonce(batch, addr, 0)
	}
	return s.db.Write(batch, nil)
}

// ApplyBlock applies all transactions in the block and updates state.
// Caller must have verified block and txs.
func (s *State) ApplyBlock(b *core.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Load current balances/nonces for all involved addresses
	balances := make(map[crypto.Address]uint64)
	nonces := make(map[crypto.Address]uint64)
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		if _, ok := balances[tx.From]; !ok {
			bal, _ := s.getBalanceLocked(tx.From)
			balances[tx.From] = bal
			nonce, _ := s.getNonceLocked(tx.From)
			nonces[tx.From] = nonce
		}
		if _, ok := balances[tx.To]; !ok {
			bal, _ := s.getBalanceLocked(tx.To)
			balances[tx.To] = bal
		}
	}
	// Apply txs in order
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		total := tx.Amount + tx.Fee
		if balances[tx.From] < total {
			continue // skip invalid
		}
		balances[tx.From] -= total
		nonces[tx.From]++
		balances[tx.To] += tx.Amount
	}
	batch := new(leveldb.Batch)
	for addr, bal := range balances {
		setBalance(batch, addr, bal)
	}
	for addr, n := range nonces {
		setNonce(batch, addr, n)
	}
	return s.db.Write(batch, nil)
}

func (s *State) getBalanceLocked(addr crypto.Address) (uint64, error) {
	val, err := s.db.Get(balanceKey(addr), nil)
	if err == leveldb.ErrNotFound {
		return 0, nil
	}
	if err != nil || len(val) < 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(val), nil
}

func (s *State) getNonceLocked(addr crypto.Address) (uint64, error) {
	val, err := s.db.Get(nonceKey(addr), nil)
	if err == leveldb.ErrNotFound {
		return 0, nil
	}
	if err != nil || len(val) < 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(val), nil
}

// ResetAndReplay clears state, applies genesis, then applies blocks in order (for reorg).
func (s *State) ResetAndReplay(allocation config.GenesisAllocation, blocks []*core.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Clear all balance/nonce keys by iterating (LevelDB has no clear; we iterate and delete)
	iter := s.db.NewIterator(nil, nil)
	batch := new(leveldb.Batch)
	for iter.Next() {
		batch.Delete(iter.Key())
	}
	iter.Release()
	if err := s.db.Write(batch, nil); err != nil {
		return err
	}
	if err := s.applyGenesisLocked(allocation); err != nil {
		return err
	}
	for _, b := range blocks {
		if err := s.applyBlockLocked(b); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) applyGenesisLocked(allocation config.GenesisAllocation) error {
	batch := new(leveldb.Batch)
	for addrHex, amount := range allocation {
		b, err := hex.DecodeString(addrHex)
		if err != nil || len(b) != crypto.AddressSize {
			continue
		}
		var addr crypto.Address
		copy(addr[:], b)
		setBalance(batch, addr, amount)
		setNonce(batch, addr, 0)
	}
	return s.db.Write(batch, nil)
}

func (s *State) applyBlockLocked(b *core.Block) error {
	balances := make(map[crypto.Address]uint64)
	nonces := make(map[crypto.Address]uint64)
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		if _, ok := balances[tx.From]; !ok {
			bal, _ := s.getBalanceLocked(tx.From)
			balances[tx.From] = bal
			nonce, _ := s.getNonceLocked(tx.From)
			nonces[tx.From] = nonce
		}
		if _, ok := balances[tx.To]; !ok {
			bal, _ := s.getBalanceLocked(tx.To)
			balances[tx.To] = bal
		}
	}
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		total := tx.Amount + tx.Fee
		if balances[tx.From] < total {
			continue
		}
		balances[tx.From] -= total
		nonces[tx.From]++
		balances[tx.To] += tx.Amount
	}
	batch := new(leveldb.Batch)
	for addr, bal := range balances {
		setBalance(batch, addr, bal)
	}
	for addr, n := range nonces {
		setNonce(batch, addr, n)
	}
	return s.db.Write(batch, nil)
}
