package handler

import "testing"

func TestIsCatppuccinMochaExpenseColor(t *testing.T) {
	if !isCatppuccinMochaExpenseColor("mauve") {
		t.Fatal("mauve should be valid")
	}
	if isCatppuccinMochaExpenseColor("not-a-token") {
		t.Fatal("invalid should be false")
	}
}
