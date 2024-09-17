package core

import (
	"encoding/json"
	"os"
	"path/filepath"

	"hanticoin/config"
	"hanticoin/crypto"
)

const genesisFilename = "genesis.json"

// GenesisBlock creates the genesis block (height 0, no txs, zero hashes).
// Allocations are applied in state; the block itself has no transactions.
func GenesisBlock(validator crypto.Address) *Block {
	b := &Block{
		Height:           0,
		Timestamp:        0,
		PreviousHash:     crypto.Hash{},
		MerkleRoot:       crypto.Hash{},
		ValidatorAddress: validator,
		Transactions:     nil,
	}
	b.SetHash()
	return b
}

// WriteGenesisFile writes genesis block and allocation to dir for replay.
func WriteGenesisFile(dir string, cfg *config.ChainConfig, validator crypto.Address) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	g := GenesisBlock(validator)
	type genesisFile struct {
		Block       *Block                    `json:"block"`
		Allocation  config.GenesisAllocation  `json:"allocation"`
	}
	data, err := json.MarshalIndent(genesisFile{Block: g, Allocation: cfg.Genesis}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, genesisFilename), data, 0644)
}

// LoadGenesisFromFile loads genesis block and allocation from dir.
func LoadGenesisFromFile(dir string) (block *Block, allocation config.GenesisAllocation, err error) {
	data, err := os.ReadFile(filepath.Join(dir, genesisFilename))
	if err != nil {
		return nil, nil, err
	}
	var out struct {
		Block      *Block                   `json:"block"`
		Allocation config.GenesisAllocation `json:"allocation"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, nil, err
	}
	return out.Block, out.Allocation, nil
}
