package store

import (
	"crypto/rand"
	"math/big"
)

const (
	idAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	idLength   = 10
)

// GenerateID produces a random 10-character ID suitable for pasta URLs.
func GenerateID() string {
	b := make([]byte, idLength)
	max := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		b[i] = idAlphabet[n.Int64()]
	}
	return string(b)
}
