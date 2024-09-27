package config

// Decimals for HTC (1 HTC = 10^6 smallest units).
const Decimals = 6

// TotalSupply is 21 billion HTC in smallest units.
const TotalSupply = 21_000_000_000 * 1_000_000

// MinFee is minimum fee per transaction (anti-spam), in smallest units.
const MinFee = 100

// MaxBlockSize is maximum number of bytes per block (approximate).
const MaxBlockSize = 1_000_000

// BlockTimeTargetSeconds is target block interval (3–5 seconds).
const BlockTimeTargetSeconds = 4

// MinStake is minimum HTC to be a validator (e.g. 5M HTC).
const MinStake = 5_000_000 * 1_000_000

// GenesisAllocation defines initial balances (address -> amount in smallest units).
type GenesisAllocation map[string]uint64

// ValidatorGenesis is one validator in genesis (address hex, stake in smallest units).
type ValidatorGenesis struct {
	Address string `json:"address"`
	Stake   uint64 `json:"stake"`
}

// ChainConfig holds chain and genesis parameters.
type ChainConfig struct {
	TotalSupply       uint64
	Decimals          uint8
	MinFee            uint64
	MinStake          uint64
	MaxBlockSize      int
	Genesis           GenesisAllocation
	GenesisValidators []ValidatorGenesis `json:"genesis_validators,omitempty"`
}

// P2PConfig holds network parameters.
type P2PConfig struct {
	ListenAddr string   // e.g. ":3030"
	Seeds      []string // bootstrap peers
	MaxPeers   int
}

// DefaultP2PConfig returns default P2P config (no seeds; add via flag or config file).
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddr: ":3030",
		Seeds:      nil,
		MaxPeers:   50,
	}
}

// Well-known genesis address hexes (40 hex chars = 20 bytes).
const (
	TreasuryAddrHex     = "0000000000000000000000000000000000000001"
	StakingAddrHex      = "0000000000000000000000000000000000000002"
	DistributionAddrHex = "0000000000000000000000000000000000000003"
)

// DefaultChainConfig returns the default Hanticoin chain config.
// Genesis keys are address hex (40 chars).
func DefaultChainConfig() *ChainConfig {
	return &ChainConfig{
		TotalSupply:  TotalSupply,
		Decimals:     Decimals,
		MinFee:       MinFee,
		MinStake:     MinStake,
		MaxBlockSize: MaxBlockSize,
		Genesis: GenesisAllocation{
			TreasuryAddrHex:     10_000_000_000 * 1_000_000, // 10B HTC
			StakingAddrHex:      5_000_000_000 * 1_000_000,  // 5B HTC
			DistributionAddrHex: 6_000_000_000 * 1_000_000,  // 6B HTC
		},
		GenesisValidators: nil, // set when initializing for PoS
	}
}
