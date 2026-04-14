package handler

import "testing"

func TestParseFundBucketPriority(t *testing.T) {
	p, ok := parseFundBucketPriority("high")
	if !ok || p != "high" {
		t.Fatalf("got %q ok=%v", p, ok)
	}
	p, ok = parseFundBucketPriority("")
	if !ok || p != "medium" {
		t.Fatalf("empty default: %q ok=%v", p, ok)
	}
	_, ok = parseFundBucketPriority("nope")
	if ok {
		t.Fatal("expected invalid")
	}
}
