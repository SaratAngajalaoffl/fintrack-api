package auth

import (
	"regexp"
	"strings"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

var supportedCurrencies = map[string]struct{}{
	"USD": {}, "EUR": {}, "GBP": {}, "JPY": {}, "AUD": {}, "CAD": {}, "INR": {}, "BRL": {},
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateEmail(email string) string {
	n := NormalizeEmail(email)
	if !emailRe.MatchString(n) {
		return "Invalid email address"
	}
	return ""
}

func ValidatePassword(password string) string {
	if len(password) < 8 {
		return "Password must be at least 8 characters"
	}
	return ""
}

func ValidateOTP(otp string) string {
	s := strings.TrimSpace(otp)
	if len(s) != OTPLength {
		return "OTP must be 6 digits"
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return "OTP must be 6 digits"
		}
	}
	return ""
}

func NormalizeProfileName(name string) string {
	s := strings.TrimSpace(name)
	return regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
}

func ValidateProfileName(name string) string {
	n := NormalizeProfileName(name)
	if len(n) < 2 {
		return "Name must be at least 2 characters"
	}
	if len(n) > 80 {
		return "Name must be at most 80 characters"
	}
	return ""
}

func ParsePreferredCurrency(value string) string {
	n := strings.ToUpper(strings.TrimSpace(value))
	if _, ok := supportedCurrencies[n]; ok {
		return n
	}
	return "USD"
}

func ValidatePreferredCurrency(value string) string {
	n := strings.ToUpper(strings.TrimSpace(value))
	if _, ok := supportedCurrencies[n]; !ok {
		return "Select a supported currency"
	}
	return ""
}
