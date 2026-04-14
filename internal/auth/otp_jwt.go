package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	PurposePasswordReset = "password_reset"
	PurposePasswordChange  = "password_change"
)

type IssueOTPTicketResult struct {
	OTP       string
	OtpToken  string
	ExpiresAt time.Time
}

// IssueOTPTicket builds a stateless OTP ticket (HS256), matching web otp.ts.
func IssueOTPTicket(secret []byte, userID, purpose string, emailNorm *string) (*IssueOTPTicketResult, error) {
	otp, err := GenerateNumericOTP()
	if err != nil {
		return nil, err
	}
	otpHash, err := HashOTP(otp)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(time.Duration(OTPTTL) * time.Millisecond)

	claims := jwt.MapClaims{
		"sub":     userID,
		"purpose": purpose,
		"oh":      otpHash,
		"iat":     time.Now().Unix(),
		"exp":     expiresAt.Unix(),
	}
	if emailNorm != nil {
		claims["em"] = *emailNorm
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	otpToken, err := tok.SignedString(secret)
	if err != nil {
		return nil, err
	}
	return &IssueOTPTicketResult{OTP: otp, OtpToken: otpToken, ExpiresAt: expiresAt}, nil
}

func VerifyOTPTicket(secret []byte, otpToken, otp, purpose string, emailNorm *string) (sub string, ok bool) {
	parsed, err := jwt.Parse(otpToken, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected alg")
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return "", false
	}
	c, mapOK := parsed.Claims.(jwt.MapClaims)
	if !mapOK {
		return "", false
	}
	if p, _ := c["purpose"].(string); p != purpose {
		return "", false
	}
	oh, _ := c["oh"].(string)
	if oh == "" || !VerifyOTP(otp, oh) {
		return "", false
	}
	if emailNorm != nil {
		em, _ := c["em"].(string)
		if em != *emailNorm {
			return "", false
		}
	}
	s, _ := c["sub"].(string)
	if s == "" {
		return "", false
	}
	return s, true
}
