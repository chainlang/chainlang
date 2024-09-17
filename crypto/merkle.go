package crypto

// MerkleRoot computes the Merkle root of a list of hashes.
// Empty list returns zero hash. Single element is hashed once.
func MerkleRoot(hashes []Hash) Hash {
	if len(hashes) == 0 {
		return Hash{}
	}
	if len(hashes) == 1 {
		return HashBytes(hashes[0][:])
	}
	// Build bottom level: duplicate odd element so we have pairs
	level := make([]Hash, 0, (len(hashes)+1)/2)
	for i := 0; i < len(hashes); i += 2 {
		var h Hash
		if i+1 < len(hashes) {
			h = HashBytes(append(hashes[i][:], hashes[i+1][:]...))
		} else {
			h = HashBytes(append(hashes[i][:], hashes[i][:]...))
		}
		level = append(level, h)
	}
	return MerkleRoot(level)
}
