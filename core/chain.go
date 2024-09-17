package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"hanticoin/config"
	"hanticoin/crypto"
)

const (
	blocksDir = "blocks"
	chainMeta = "chain.json"
)

// Chain holds the canonical block list and metadata on disk.
type Chain struct {
	mu       sync.RWMutex
	dir      string
	cfg      *config.ChainConfig
	Height   uint64    `json:"height"`
	LastHash crypto.Hash `json:"last_hash"`
}

// ChainMeta is persisted to chain.json.
type ChainMeta struct {
	Height   uint64      `json:"height"`
	LastHash crypto.Hash `json:"last_hash"`
}

// NewChain creates or loads a chain from dir.
func NewChain(dir string, cfg *config.ChainConfig) (*Chain, error) {
	if cfg == nil {
		cfg = config.DefaultChainConfig()
	}
	c := &Chain{dir: dir, cfg: cfg}
	if err := os.MkdirAll(filepath.Join(dir, blocksDir), 0755); err != nil {
		return nil, err
	}
	metaPath := filepath.Join(dir, chainMeta)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	var meta ChainMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	c.Height = meta.Height
	c.LastHash = meta.LastHash
	return c, nil
}

// Save persists chain metadata.
func (c *Chain) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	meta := ChainMeta{c.Height, c.LastHash}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, chainMeta), data, 0644)
}

// AppendBlock persists a block and updates chain metadata.
func (c *Chain) AppendBlock(b *Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	path := filepath.Join(c.dir, blocksDir, b.Hash.Hex()+".json")
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	c.Height = b.Height
	c.LastHash = b.Hash
	return c.Save()
}

// GetBlock loads a block by hash.
func (c *Chain) GetBlock(h crypto.Hash) (*Block, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	path := filepath.Join(c.dir, blocksDir, h.Hex()+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b Block
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// GetGenesis returns the genesis block from genesis.json if present.
func (c *Chain) GetGenesis() (*Block, error) {
	g, _, err := LoadGenesisFromFile(c.dir)
	return g, err
}

// GetBlockByHeight loads block by height (scan directory or maintain index; for Phase 1 we scan).
func (c *Chain) GetBlockByHeight(height uint64) (*Block, error) {
	if height == 0 {
		return c.GetGenesis()
	}
	// Linear scan; can add height->hash index later
	entries, err := os.ReadDir(filepath.Join(c.dir, blocksDir))
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.dir, blocksDir, e.Name()))
		if err != nil {
			continue
		}
		var b Block
		if err := json.Unmarshal(data, &b); err != nil {
			continue
		}
		if b.Height == height {
			return &b, nil
		}
	}
	return nil, os.ErrNotExist
}

// LastBlock returns the latest block.
func (c *Chain) LastBlock() (*Block, error) {
	c.mu.RLock()
	last := c.LastHash
	c.mu.RUnlock()
	if last.IsZero() {
		return nil, nil
	}
	return c.GetBlock(last)
}

// Config returns the chain config.
func (c *Chain) Config() *config.ChainConfig { return c.cfg }
