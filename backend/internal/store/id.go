package store

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	idLength   = 10
)

// GenerateID produces a random 10-character ID suitable for pasta URLs.
func GenerateID() (string, error) {
	b := make([]byte, idLength)
	max := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate ID: %w", err)
		}
		b[i] = idAlphabet[n.Int64()]
	}
	return string(b), nil
}
