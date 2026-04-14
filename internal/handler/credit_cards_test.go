package handler

import (
	"math"
	"testing"
)

func TestValidBillDayFloatPtr(t *testing.T) {
	v := 15.0
	got, ok := validBillDayFloatPtr(&v)
	if !ok || got != 15 {
		t.Fatalf("got %d ok=%v", got, ok)
	}
	if _, ok := validBillDayFloatPtr(nil); ok {
		t.Fatal("expected false")
	}
	bad := 1.5
	if _, ok := validBillDayFloatPtr(&bad); ok {
		t.Fatal("expected false for non-integer")
	}
	nan := math.NaN()
	if _, ok := validBillDayFloatPtr(&nan); ok {
		t.Fatal("expected false for nan")
	}
	low := 0.0
	if _, ok := validBillDayFloatPtr(&low); ok {
		t.Fatal("expected false")
	}
	high := 32.0
	if _, ok := validBillDayFloatPtr(&high); ok {
		t.Fatal("expected false")
	}
}

func TestNormalizeCategoryNamesSlice(t *testing.T) {
	got := normalizeCategoryNamesSlice([]string{"  a ", "a", "", " b "})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%v", got)
	}
}
