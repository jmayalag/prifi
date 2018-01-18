package dcnet

import (
	"gopkg.in/dedis/crypto.v0/abstract"
)

/**
 * CellCoder is the cell encoding, decoding, and accountability interface.
 * Designed to support multiple alternative cell encoding methods,
 * some for single-owner cells (in which only one key-holder transmits),
 * others for multi-owner cells (for transmit-request bitmaps for example).
 */
type DCNet interface {

	// Computes the client ciphertext size for a given cell payload length,
	// accounting for whatever expansion the cell encoding imposes.
	GetClientCipherSize(payloadSize int) int

	// Computes the trustee ciphertext size for a given cell payload length,
	// accounting for whatever expansion the cell encoding imposes.
	GetTrusteeCipherSize(payloadSize int) int

	UpdateHistory(data []byte)

	ClientSetup(suite abstract.Suite, trusteeciphers []abstract.Cipher)

	RelaySetup(suite abstract.Suite, trusteeinfo [][]byte)

	TrusteeSetup(suite abstract.Suite, clientciphers []abstract.Cipher) []byte

	// Encode a ciphertext slice for the current cell, transmitting the optional payload if non-nil.
	ClientEncode(payload []byte, payloadSize int, history abstract.Cipher) []byte

	// Encode the trustee's ciphertext slice for the current cell. Can be pre-computed for an interval based on a client-set.
	TrusteeEncode(payloadSize int) []byte

	// Initialize per-cell decoding state for the next cell
	DecodeStart(payloadSize int, history abstract.Cipher)

	// Combine a client's ciphertext slice into this cell.
	DecodeClient(slice []byte)

	// Same but to combine a trustee's slice into this cell.
	DecodeTrustee(slice []byte)

	// Combine all client and trustee slices provided via DecodeSlice(), to reveal the anonymized plaintext for this cell.
	DecodeCell() []byte
}

// DCNetFactory is the type of methods providing a CellCoder.
type DCNetFactory func(bool) DCNet
