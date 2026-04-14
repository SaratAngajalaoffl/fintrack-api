package handler

import (
	"math"
	"testing"
)

func TestIsFinite(t *testing.T) {
	if !isFinite(0) || !isFinite(-1.5) {
		t.Fatal("expected finite")
	}
	if isFinite(math.NaN()) || isFinite(math.Inf(1)) {
		t.Fatal("expected non-finite")
	}
}
