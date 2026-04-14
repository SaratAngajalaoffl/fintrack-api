package auth

import "testing"

func TestIssueAndVerifyOTPTicketPasswordReset(t *testing.T) {
	secret := []byte("sixteencharslong")
	em := "user@example.com"
	res, err := IssueOTPTicket(secret, "uid-1", PurposePasswordReset, &em)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.OTP) != OTPLength {
		t.Fatalf("otp len %d", len(res.OTP))
	}
	sub, ok := VerifyOTPTicket(secret, res.OtpToken, res.OTP, PurposePasswordReset, &em)
	if !ok || sub != "uid-1" {
		t.Fatalf("ok=%v sub=%q", ok, sub)
	}
}

func TestIssueAndVerifyOTPTicketPasswordChange(t *testing.T) {
	secret := []byte("sixteencharslong")
	res, err := IssueOTPTicket(secret, "uid-2", PurposePasswordChange, nil)
	if err != nil {
		t.Fatal(err)
	}
	sub, ok := VerifyOTPTicket(secret, res.OtpToken, res.OTP, PurposePasswordChange, nil)
	if !ok || sub != "uid-2" {
		t.Fatalf("ok=%v sub=%q", ok, sub)
	}
}

func TestVerifyOTPTicketWrongPurpose(t *testing.T) {
	secret := []byte("sixteencharslong")
	em := "a@b.co"
	res, err := IssueOTPTicket(secret, "u", PurposePasswordReset, &em)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := VerifyOTPTicket(secret, res.OtpToken, res.OTP, PurposePasswordChange, &em)
	if ok {
		t.Fatal("expected failure")
	}
}

func TestVerifyOTPTicketWrongEmail(t *testing.T) {
	secret := []byte("sixteencharslong")
	em := "a@b.co"
	res, err := IssueOTPTicket(secret, "u", PurposePasswordReset, &em)
	if err != nil {
		t.Fatal(err)
	}
	other := "b@b.co"
	_, ok := VerifyOTPTicket(secret, res.OtpToken, res.OTP, PurposePasswordReset, &other)
	if ok {
		t.Fatal("expected failure")
	}
}

func TestVerifyOTPTicketWrongOTP(t *testing.T) {
	secret := []byte("sixteencharslong")
	em := "a@b.co"
	res, err := IssueOTPTicket(secret, "u", PurposePasswordReset, &em)
	if err != nil {
		t.Fatal(err)
	}
	_, ok := VerifyOTPTicket(secret, res.OtpToken, "000000", PurposePasswordReset, &em)
	if ok {
		t.Fatal("expected failure")
	}
}

func TestVerifyOTPTicketBadToken(t *testing.T) {
	em := "a@b.co"
	_, ok := VerifyOTPTicket([]byte("sixteencharslong"), "x", "123456", PurposePasswordReset, &em)
	if ok {
		t.Fatal("expected failure")
	}
}
