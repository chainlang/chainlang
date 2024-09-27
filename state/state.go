package state

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"sync"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	prefixBalance    = "b:"
	prefixNonce      = "n:"
	keyValidatorList = "validator_list"
	prefixValidator  = "vs:"
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

func validatorStakeKey(addr crypto.Address) []byte {
	return append([]byte(prefixValidator), addr[:]...)
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

func setValidatorStake(batch *leveldb.Batch, addr crypto.Address, stake uint64) {
	key := validatorStakeKey(addr)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, stake)
	batch.Put(key, buf)
}

// ApplyGenesis applies genesis allocation and optional validator set.
func (s *State) ApplyGenesis(allocation config.GenesisAllocation, validators []config.ValidatorGenesis) error {
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
	if len(validators) > 0 {
		var addrs []string
		for _, v := range validators {
			b, err := hex.DecodeString(v.Address)
			if err != nil || len(b) != crypto.AddressSize || v.Stake < s.cfg.MinStake {
				continue
			}
			var addr crypto.Address
			copy(addr[:], b)
			setValidatorStake(batch, addr, v.Stake)
			addrs = append(addrs, v.Address)
		}
		if len(addrs) > 0 {
			list, _ := json.Marshal(addrs)
			batch.Put([]byte(keyValidatorList), list)
		}
	}
	return s.db.Write(batch, nil)
}

// Validator is address + stake (for GetValidatorSet).
type Validator struct {
	Address crypto.Address
	Stake   uint64
}

// GetValidatorSet returns the ordered validator set from state.
func (s *State) GetValidatorSet() ([]Validator, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, err := s.db.Get([]byte(keyValidatorList), nil)
	if err == leveldb.ErrNotFound || len(val) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var addrs []string
	if err := json.Unmarshal(val, &addrs); err != nil {
		return nil, err
	}
	var out []Validator
	for _, a := range addrs {
		b, err := hex.DecodeString(a)
		if err != nil || len(b) != crypto.AddressSize {
			continue
		}
		var addr crypto.Address
		copy(addr[:], b)
		stake, _ := s.getValidatorStakeLocked(addr)
		if stake > 0 {
			out = append(out, Validator{Address: addr, Stake: stake})
		}
	}
	return out, nil
}

func (s *State) getValidatorStakeLocked(addr crypto.Address) (uint64, error) {
	val, err := s.db.Get(validatorStakeKey(addr), nil)
	if err == leveldb.ErrNotFound {
		return 0, nil
	}
	if err != nil || len(val) < 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(val), nil
}

// SlashValidator sets stake to 0 and removes from validator list.
func (s *State) SlashValidator(addr crypto.Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	listVal, err := s.db.Get([]byte(keyValidatorList), nil)
	if err == leveldb.ErrNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	var addrs []string
	if err := json.Unmarshal(listVal, &addrs); err != nil {
		return err
	}
	addrHex := addr.Hex()
	var newList []string
	for _, a := range addrs {
		if a != addrHex {
			newList = append(newList, a)
		}
	}
	batch := new(leveldb.Batch)
	setValidatorStake(batch, addr, 0)
	if len(newList) > 0 {
		data, _ := json.Marshal(newList)
		batch.Put([]byte(keyValidatorList), data)
	} else {
		batch.Delete([]byte(keyValidatorList))
	}
	return s.db.Write(batch, nil)
}

// ApplyBlock applies all transactions and distributes fees to block producer.
func (s *State) ApplyBlock(b *core.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	var totalFees uint64
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		total := tx.Amount + tx.Fee
		totalFees += tx.Fee
		if balances[tx.From] < total {
			continue
		}
		balances[tx.From] -= total
		nonces[tx.From]++
		balances[tx.To] += tx.Amount
	}
	if totalFees > 0 {
		// Use balance after txs (from map), not initial DB balance
		if _, ok := balances[b.ValidatorAddress]; !ok {
			bal, _ := s.getBalanceLocked(b.ValidatorAddress)
			balances[b.ValidatorAddress] = bal
		}
		balances[b.ValidatorAddress] += totalFees
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

// ResetAndReplay clears state, applies genesis (with validators), then applies blocks (for reorg).
func (s *State) ResetAndReplay(allocation config.GenesisAllocation, validators []config.ValidatorGenesis, blocks []*core.Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	iter := s.db.NewIterator(nil, nil)
	batch := new(leveldb.Batch)
	for iter.Next() {
		batch.Delete(iter.Key())
	}
	iter.Release()
	if err := s.db.Write(batch, nil); err != nil {
		return err
	}
	if err := s.applyGenesisLocked(allocation, validators); err != nil {
		return err
	}
	for _, b := range blocks {
		if err := s.applyBlockLocked(b); err != nil {
			return err
		}
	}
	return nil
}

func (s *State) applyGenesisLocked(allocation config.GenesisAllocation, validators []config.ValidatorGenesis) error {
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
	if len(validators) > 0 {
		var addrs []string
		for _, v := range validators {
			b, err := hex.DecodeString(v.Address)
			if err != nil || len(b) != crypto.AddressSize || v.Stake < s.cfg.MinStake {
				continue
			}
			var addr crypto.Address
			copy(addr[:], b)
			setValidatorStake(batch, addr, v.Stake)
			addrs = append(addrs, v.Address)
		}
		if len(addrs) > 0 {
			list, _ := json.Marshal(addrs)
			batch.Put([]byte(keyValidatorList), list)
		}
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
	var totalFees uint64
	for i := range b.Transactions {
		tx := &b.Transactions[i]
		total := tx.Amount + tx.Fee
		totalFees += tx.Fee
		if balances[tx.From] < total {
			continue
		}
		balances[tx.From] -= total
		nonces[tx.From]++
		balances[tx.To] += tx.Amount
	}
	if totalFees > 0 {
		if _, ok := balances[b.ValidatorAddress]; !ok {
			bal, _ := s.getBalanceLocked(b.ValidatorAddress)
			balances[b.ValidatorAddress] = bal
		}
		balances[b.ValidatorAddress] += totalFees
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
