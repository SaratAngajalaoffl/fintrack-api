package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateNumericOTP() (string, error) {
	max := big.NewInt(1_000_000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
