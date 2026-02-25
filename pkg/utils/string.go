package utils

import (
	"crypto/rand"
	"math/big"
)

// GenerateCode generates a random alphanumeric code of the specified length.
// The generated code consists of uppercase letters (A-Z) and digits (0-9).
// This function uses cryptographically secure random number generation.
func GenerateCode(length int) string {
	var charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	if length <= 0 {
		return ""
	}

	code := make([]byte, length)
	maxIndex := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, maxIndex)
		if err != nil {
			return ""
		}
		code[i] = charset[num.Int64()]
	}
	return string(code)
}

// SafeSlice returns an empty slice if the input slice is nil.
// This is a generic utility function that works with slices of any type T.
// It prevents potential nil pointer dereferences when working with slices that might be nil.
func SafeSlice[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
