package state

import (
	"testing"

	"hanticoin/config"
	"hanticoin/core"
	"hanticoin/crypto"
)

const (
	addr1 = "0000000000000000000000000000000000000001"
	addr2 = "0000000000000000000000000000000000000002"
)

func mustAddr(s string) crypto.Address {
	a, err := crypto.AddressFromHex(s)
	if err != nil {
		panic(err)
	}
	return a
}

func TestSlashValidator_RemovesFromSet(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultChainConfig()
	st, err := Open(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	allocation := config.GenesisAllocation{
		addr1: 10 * 1_000_000 * 1_000_000,
		addr2: 10 * 1_000_000 * 1_000_000,
	}
	validators := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
		{Address: addr2, Stake: config.MinStake},
	}
	if err := st.ApplyGenesis(allocation, validators); err != nil {
		t.Fatal(err)
	}

	list, err := st.GetValidatorSet()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 validators, got %d", len(list))
	}

	if err := st.SlashValidator(mustAddr(addr1)); err != nil {
		t.Fatal(err)
	}

	list, err = st.GetValidatorSet()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("after slash: expected 1 validator, got %d", len(list))
	}
	if list[0].Address != mustAddr(addr2) {
		t.Errorf("remaining validator should be addr2, got %s", list[0].Address.Hex())
	}
}

func TestApplyBlock_FeeDistribution(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultChainConfig()
	st, err := Open(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	producer := mustAddr(addr1)
	allocation := config.GenesisAllocation{
		addr1: 1000 * 1_000_000, // 1000 HTC
		addr2: 500 * 1_000_000,
	}
	validators := []config.ValidatorGenesis{
		{Address: addr1, Stake: config.MinStake},
	}
	if err := st.ApplyGenesis(allocation, validators); err != nil {
		t.Fatal(err)
	}

	balBefore, _ := st.GetBalance(producer)

	// Block with two txs: fee 100 and 200 -> producer gets 300
	txs := []core.Transaction{
		{From: producer, To: mustAddr(addr2), Amount: 100, Fee: 100, Nonce: 0},
		{From: producer, To: mustAddr(addr2), Amount: 50, Fee: 200, Nonce: 1},
	}
	b := &core.Block{
		Height:           1,
		ValidatorAddress: producer,
		Transactions:     txs,
	}
	b.SetHash()

	if err := st.ApplyBlock(b); err != nil {
		t.Fatal(err)
	}

	balAfter, _ := st.GetBalance(producer)
	totalFees := uint64(100 + 200)
	// Producer spent 100+100 + 50+200 = 450, received 300 in fees -> net -150
	expected := balBefore - (100+100+50+200) + totalFees
	if balAfter != expected {
		t.Errorf("producer balance: got %d, want %d (before %d, spent 450, +fees %d)", balAfter, expected, balBefore, totalFees)
	}
}

func TestApplyBlock_NoFees_NoChangeToProducer(t *testing.T) {
	dir := t.TempDir()
	cfg := config.DefaultChainConfig()
	st, err := Open(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	producer := mustAddr(addr1)
	allocation := config.GenesisAllocation{addr1: 1000 * 1_000_000}
	validators := []config.ValidatorGenesis{{Address: addr1, Stake: config.MinStake}}
	if err := st.ApplyGenesis(allocation, validators); err != nil {
		t.Fatal(err)
	}

	balBefore, _ := st.GetBalance(producer)
	b := &core.Block{
		Height:           1,
		ValidatorAddress: producer,
		Transactions:     nil,
	}
	b.SetHash()
	if err := st.ApplyBlock(b); err != nil {
		t.Fatal(err)
	}
	balAfter, _ := st.GetBalance(producer)
	if balAfter != balBefore {
		t.Errorf("empty block: producer balance changed from %d to %d", balBefore, balAfter)
	}
}
