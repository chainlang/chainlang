package core

import (
	"encoding/json"
	"time"

	"hanticoin/crypto"
	"hanticoin/config"
)

// Transaction represents a transfer of HTC.
type Transaction struct {
	From        crypto.Address `json:"from"`
	To          crypto.Address `json:"to"`
	Amount      uint64        `json:"amount"`
	Fee         uint64        `json:"fee"`
	Nonce       uint64        `json:"nonce"`
	SenderPubkey []byte       `json:"sender_pubkey"` // compressed ECDSA pubkey for verification
	Signature   []byte       `json:"signature"`
}

// TxHash returns the hash of the tx payload (for signing and Merkle).
func (t *Transaction) TxHash() crypto.Hash {
	payload := struct {
		From   crypto.Address `json:"from"`
		To     crypto.Address `json:"to"`
		Amount uint64        `json:"amount"`
		Fee    uint64        `json:"fee"`
		Nonce  uint64        `json:"nonce"`
	}{t.From, t.To, t.Amount, t.Fee, t.Nonce}
	b, _ := json.Marshal(payload)
	return crypto.HashBytes(b)
}

// VerifySignature checks ECDSA signature and that From matches sender pubkey.
func (t *Transaction) VerifySignature() bool {
	pub := crypto.UnmarshalPubkey(t.SenderPubkey)
	if pub == nil || len(t.Signature) < 64 {
		return false
	}
	if crypto.PubkeyToAddress(pub) != t.From {
		return false
	}
	return crypto.Verify(pub, t.TxHash(), crypto.Signature(t.Signature))
}

// SatisfiesMinFee returns true if fee >= config.MinFee.
func (t *Transaction) SatisfiesMinFee() bool {
	return t.Fee >= config.MinFee
}

// Block represents a single block in the chain.
type Block struct {
	Height           uint64        `json:"height"`
	Timestamp        int64         `json:"timestamp"`
	PreviousHash     crypto.Hash   `json:"previous_hash"`
	MerkleRoot       crypto.Hash   `json:"merkle_root"`
	ValidatorAddress crypto.Address `json:"validator_address"`
	Transactions     []Transaction `json:"transactions"`
	Signature        []byte        `json:"signature"`
	Hash             crypto.Hash   `json:"hash"`
}

// BlockHash computes the hash of the block (excluding Hash and Signature for the hashed payload).
func (b *Block) BlockHash() crypto.Hash {
	payload := struct {
		Height           uint64          `json:"height"`
		Timestamp        int64           `json:"timestamp"`
		PreviousHash     crypto.Hash     `json:"previous_hash"`
		MerkleRoot       crypto.Hash     `json:"merkle_root"`
		ValidatorAddress crypto.Address  `json:"validator_address"`
		TxHashes         []crypto.Hash   `json:"tx_hashes"`
	}{
		b.Height,
		b.Timestamp,
		b.PreviousHash,
		b.MerkleRoot,
		b.ValidatorAddress,
		txHashes(b.Transactions),
	}
	data, _ := json.Marshal(payload)
	return crypto.HashBytes(data)
}

func txHashes(txs []Transaction) []crypto.Hash {
	out := make([]crypto.Hash, len(txs))
	for i := range txs {
		out[i] = txs[i].TxHash()
	}
	return out
}

// SetHash sets block Hash from current fields.
func (b *Block) SetHash() {
	b.Hash = b.BlockHash()
}

// NewBlock creates a block (MerkleRoot and Hash must be set by caller or BuildBlock).
func NewBlock(height uint64, prevHash crypto.Hash, validator crypto.Address, txs []Transaction) *Block {
	b := &Block{
		Height:           height,
		Timestamp:        time.Now().Unix(),
		PreviousHash:     prevHash,
		ValidatorAddress: validator,
		Transactions:     txs,
	}
	hashes := make([]crypto.Hash, len(txs))
	for i := range txs {
		hashes[i] = txs[i].TxHash()
	}
	b.MerkleRoot = crypto.MerkleRoot(hashes)
	b.SetHash()
	return b
}
