package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type SessionPayload struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// SignSessionToken matches jose SignJWT({ email }).setSubject(sub). HS256, exp JWT_TTL_SEC.
func SignSessionToken(secret []byte, sub, email string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": email,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Duration(JWT_TTL_SEC) * time.Second).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret)
}

func VerifySessionToken(secret []byte, token string) (*SessionPayload, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return nil, err
	}
	c, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	subv, _ := c["sub"].(string)
	em, _ := c["email"].(string)
	if subv == "" || em == "" {
		return nil, fmt.Errorf("missing sub/email")
	}
	return &SessionPayload{Sub: subv, Email: em}, nil
}
