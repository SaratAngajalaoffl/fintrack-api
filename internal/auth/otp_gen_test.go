package auth

import "testing"

func TestGenerateNumericOTP(t *testing.T) {
	for range 20 {
		s, err := GenerateNumericOTP()
		if err != nil {
			t.Fatal(err)
		}
		if len(s) != OTPLength {
			t.Fatalf("len %d: %q", len(s), s)
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				t.Fatalf("non-digit: %q", s)
			}
		}
	}
}
