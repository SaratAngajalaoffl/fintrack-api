package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), PasswordBcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyPassword(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func HashOTP(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), OTPBcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyOTP(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
