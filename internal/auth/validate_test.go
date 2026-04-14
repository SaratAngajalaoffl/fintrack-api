package auth

import "testing"

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Foo@BAR.com "); got != "foo@bar.com" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateEmail(t *testing.T) {
	if msg := ValidateEmail("not-an-email"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateEmail("a@b.co"); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}

func TestValidatePassword(t *testing.T) {
	if msg := ValidatePassword("short"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidatePassword("longenough"); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}

func TestValidateOTP(t *testing.T) {
	if msg := ValidateOTP("12345"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateOTP("1234567"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateOTP("12345a"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateOTP("123456"); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}

func TestNormalizeProfileName(t *testing.T) {
	if got := NormalizeProfileName("  hello   world  "); got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateProfileName(t *testing.T) {
	if msg := ValidateProfileName("a"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateProfileName(string(make([]byte, 81))); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidateProfileName("ab"); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}

func TestParsePreferredCurrency(t *testing.T) {
	if got := ParsePreferredCurrency(""); got != "USD" {
		t.Fatalf("got %q", got)
	}
	if got := ParsePreferredCurrency("inr"); got != "INR" {
		t.Fatalf("got %q", got)
	}
}

func TestValidatePreferredCurrency(t *testing.T) {
	if msg := ValidatePreferredCurrency("XXX"); msg == "" {
		t.Fatal("expected error")
	}
	if msg := ValidatePreferredCurrency("USD"); msg != "" {
		t.Fatalf("unexpected: %s", msg)
	}
}
