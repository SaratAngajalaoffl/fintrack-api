package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("verify failed")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("verify should fail")
	}
}

func TestHashAndVerifyOTP(t *testing.T) {
	hash, err := HashOTP("123456")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyOTP("123456", hash) {
		t.Fatal("verify failed")
	}
	if VerifyOTP("000000", hash) {
		t.Fatal("verify should fail")
	}
}
