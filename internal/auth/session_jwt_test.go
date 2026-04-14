package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestSignAndVerifySessionToken(t *testing.T) {
	secret := []byte("sixteencharslong")
	tok, err := SignSessionToken(secret, "user-id-1", "u@example.com")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := VerifySessionToken(secret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Sub != "user-id-1" || payload.Email != "u@example.com" {
		t.Fatalf("payload: %+v", payload)
	}
}

func TestVerifySessionTokenWrongSecret(t *testing.T) {
	secret := []byte("sixteencharslong")
	tok, err := SignSessionToken(secret, "sub", "e@x.com")
	if err != nil {
		t.Fatal(err)
	}
	_, err = VerifySessionToken([]byte("othersecret12345"), tok)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifySessionTokenMissingClaims(t *testing.T) {
	secret := []byte("sixteencharslong")
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = VerifySessionToken(secret, s)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifySessionTokenGarbage(t *testing.T) {
	_, err := VerifySessionToken([]byte("sixteencharslong"), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifySessionTokenExpired(t *testing.T) {
	secret := []byte("sixteencharslong")
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "u",
		"email": "a@b.co",
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	s, err := tok.SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = VerifySessionToken(secret, s)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}
