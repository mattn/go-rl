//go:build !windows
// +build !windows

package rl

import "testing"

func TestDecodeRunesKeepsIncompleteUTF8(t *testing.T) {
	rs, pending := decodeRunes([]byte{0xe3, 0x81})
	if len(rs) != 0 {
		t.Fatalf("decodeRunes returned runes %q, want none", string(rs))
	}
	if len(pending) != 2 {
		t.Fatalf("decodeRunes pending length = %d, want 2", len(pending))
	}

	rs, pending = decodeRunes(append(pending, 0x82))
	if string(rs) != "あ" {
		t.Fatalf("decodeRunes = %q, want %q", string(rs), "あ")
	}
	if len(pending) != 0 {
		t.Fatalf("decodeRunes pending length = %d, want 0", len(pending))
	}
}
