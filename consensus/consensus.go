package consensus

import (
	"encoding/hex"
	"sync"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
)

// Validator is one validator (address + stake).
type Validator struct {
	Address crypto.Address
	Stake   uint64
}

// Set is the ordered validator set (round-robin by index).
type Set struct {
	mu   sync.RWMutex
	list []Validator
}

// NewSet builds a set from genesis validators (address hex, stake).
func NewSet(genesis []config.ValidatorGenesis) *Set {
	s := &Set{}
	for _, g := range genesis {
		b, err := hex.DecodeString(g.Address)
		if err != nil || len(b) != crypto.AddressSize {
			continue
		}
		if g.Stake < config.MinStake {
			continue
		}
		var addr crypto.Address
		copy(addr[:], b)
		s.list = append(s.list, Validator{Address: addr, Stake: g.Stake})
	}
	return s
}

// NewSetFromValidators builds from already-parsed validators (e.g. from state).
func NewSetFromValidators(list []Validator) *Set {
	return &Set{list: list}
}

// ValidatorForHeight returns the validator address for the given height (round-robin).
func (s *Set) ValidatorForHeight(height uint64) crypto.Address {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.list) == 0 {
		return crypto.Address{}
	}
	idx := height % uint64(len(s.list))
	return s.list[idx].Address
}

// Size returns the number of validators.
func (s *Set) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.list)
}

// List returns a copy of the validator list.
func (s *Set) List() []Validator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Validator, len(s.list))
	copy(out, s.list)
	return out
}

// VerifyBlockValidator returns true if block.ValidatorAddress is the correct validator for block.Height.
func (s *Set) VerifyBlockValidator(b *core.Block) bool {
	return s.ValidatorForHeight(b.Height) == b.ValidatorAddress
}

// IsValidator returns true if addr is in the set.
func (s *Set) IsValidator(addr crypto.Address) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.list {
		if v.Address == addr {
			return true
		}
	}
	return false
}
