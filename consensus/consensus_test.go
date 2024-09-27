package consensus

import (
	"testing"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
)

// 40-char hex addresses for testing
const (
	addr1 = "0000000000000000000000000000000000000001"
	addr2 = "0000000000000000000000000000000000000002"
	addr3 = "0000000000000000000000000000000000000003"
)

func mustAddr(s string) crypto.Address {
	a, err := crypto.AddressFromHex(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestValidatorForHeight_RoundRobin(t *testing.T) {
	genesis := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
		{Address: addr2, Stake: config.MinStake},
		{Address: addr3, Stake: config.MinStake},
	}
	set := NewSet(genesis)
	if set.Size() != 3 {
		t.Fatalf("expected 3 validators, got %d", set.Size())
	}

	// height % 3
	if got := set.ValidatorForHeight(0); got != mustAddr(addr1) {
		t.Errorf("height 0: got %s want %s", got.Hex(), addr1)
	}
	if got := set.ValidatorForHeight(1); got != mustAddr(addr2) {
		t.Errorf("height 1: got %s want %s", got.Hex(), addr2)
	}
	if got := set.ValidatorForHeight(2); got != mustAddr(addr3) {
		t.Errorf("height 2: got %s want %s", got.Hex(), addr3)
	}
	if got := set.ValidatorForHeight(3); got != mustAddr(addr1) {
		t.Errorf("height 3: got %s want %s", got.Hex(), addr1)
	}
	if got := set.ValidatorForHeight(100); got != mustAddr(addr2) {
		t.Errorf("height 100: got %s want %s", got.Hex(), addr2)
	}
}

func TestValidatorForHeight_EmptySet(t *testing.T) {
	set := NewSet(nil)
	if set.Size() != 0 {
		t.Fatalf("expected 0 validators, got %d", set.Size())
	}
	got := set.ValidatorForHeight(0)
	if got != (crypto.Address{}) {
		t.Errorf("empty set: expected zero address, got %s", got.Hex())
	}
}

func TestVerifyBlockValidator_Accept(t *testing.T) {
	genesis := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
		{Address: addr2, Stake: config.MinStake},
	}
	set := NewSet(genesis)

	b := &core.Block{Height: 1, ValidatorAddress: mustAddr(addr2)}
	if !set.VerifyBlockValidator(b) {
		t.Error("expected VerifyBlockValidator true for correct validator at height 1")
	}
}

func TestVerifyBlockValidator_Reject(t *testing.T) {
	genesis := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
		{Address: addr2, Stake: config.MinStake},
	}
	set := NewSet(genesis)

	// height 1 should be addr2; wrong validator
	b := &core.Block{Height: 1, ValidatorAddress: mustAddr(addr1)}
	if set.VerifyBlockValidator(b) {
		t.Error("expected VerifyBlockValidator false for wrong validator")
	}

	b2 := &core.Block{Height: 0, ValidatorAddress: mustAddr(addr2)}
	if set.VerifyBlockValidator(b2) {
		t.Error("expected VerifyBlockValidator false for height 0 with addr2 (should be addr1)")
	}
}

func TestIsValidator(t *testing.T) {
	genesis := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
		{Address: addr2, Stake: config.MinStake},
	}
	set := NewSet(genesis)

	if !set.IsValidator(mustAddr(addr1)) {
		t.Error("addr1 should be validator")
	}
	if !set.IsValidator(mustAddr(addr2)) {
		t.Error("addr2 should be validator")
	}
	if set.IsValidator(mustAddr(addr3)) {
		t.Error("addr3 should not be validator")
	}
}
