package p2p

import (
	"encoding/json"

	"hanticoin/core"
	"hanticoin/crypto"
)

// Kind is the message type.
type Kind string

const (
	KindNewTx     Kind = "new_tx"
	KindNewBlock  Kind = "new_block"
	KindGetBlocks Kind = "get_blocks"
	KindBlocks    Kind = "blocks"
	KindGetPeers  Kind = "get_peers"
	KindPeers     Kind = "peers"
	KindHello     Kind = "hello"
)

// Message is the wire format (newline-delimited JSON).
type Message struct {
	Kind Kind `json:"kind"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// HelloPayload is sent on connect (genesis hash + our listen addr).
type HelloPayload struct {
	GenesisHash crypto.Hash `json:"genesis_hash"`
	ListenAddr  string      `json:"listen_addr"`
	Height      uint64      `json:"height"`
}

// GetBlocksPayload requests blocks from a height.
type GetBlocksPayload struct {
	FromHeight uint64 `json:"from_height"`
	Max        int    `json:"max"`
}

// BlocksPayload sends a batch of blocks.
type BlocksPayload struct {
	Blocks []*core.Block `json:"blocks"`
}

// PeersPayload sends peer addresses.
type PeersPayload struct {
	Peers []string `json:"peers"`
}

// EncodeMessage encodes msg to JSON (one line, for framing).
func EncodeMessage(kind Kind, payload interface{}) ([]byte, error) {
	var raw json.RawMessage
	if payload != nil {
		var err error
		raw, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(Message{Kind: kind, Payload: raw})
}

// DecodeMessage decodes a message line.
func DecodeMessage(data []byte) (Kind, json.RawMessage, error) {
	var m Message
	if err := json.Unmarshal(data, &m); err != nil {
		return "", nil, err
	}
	return m.Kind, m.Payload, nil
}
