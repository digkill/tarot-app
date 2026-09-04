package yookassa

import "testing"

func TestFormatRUB(t *testing.T) {
	if got := FormatRUB(59900); got != "599.00" {
		t.Fatalf("got %s", got)
	}
	if got := FormatRUB(4990); got != "49.90" {
		t.Fatalf("got %s", got)
	}
}
