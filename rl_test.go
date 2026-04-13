package rl

import "testing"

func TestShouldReturnEOFOnCtrlD(t *testing.T) {
	tests := []struct {
		name       string
		input      []rune
		eofOnCtrlD bool
		want       bool
	}{
		{
			name:  "empty input returns eof",
			input: []rune{},
			want:  true,
		},
		{
			name:  "non empty input keeps current behavior by default",
			input: []rune("abc"),
			want:  false,
		},
		{
			name:       "non empty input can be configured to return eof",
			input:      []rune("abc"),
			eofOnCtrlD: true,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldReturnEOFOnCtrlD(tt.input, tt.eofOnCtrlD)
			if got != tt.want {
				t.Fatalf("shouldReturnEOFOnCtrlD(%q, %v) = %v, want %v", string(tt.input), tt.eofOnCtrlD, got, tt.want)
			}
		})
	}
}

func TestApplyCompletionPreservesSuffix(t *testing.T) {
	got, cursor, ok := applyCompletion([]rune("say he world"), 4, 6, []string{"hello", "help"})
	if !ok {
		t.Fatal("applyCompletion returned !ok")
	}
	if string(got) != "say hel world" {
		t.Fatalf("applyCompletion = %q, want %q", string(got), "say hel world")
	}
	if cursor != 7 {
		t.Fatalf("applyCompletion cursor = %d, want 7", cursor)
	}
}

func TestApplyCompletionHandlesUTF8Prefix(t *testing.T) {
	got, _, ok := applyCompletion([]rune("こん"), 0, 2, []string{"こんにちは", "こんばんは"})
	if !ok {
		t.Fatal("applyCompletion returned !ok")
	}
	if string(got) != "こん" {
		t.Fatalf("applyCompletion = %q, want %q", string(got), "こん")
	}
}

func TestDeleteWordBeforeCursor(t *testing.T) {
	got, cursor, ok := deleteWordBeforeCursor([]rune("abc def ghi"), 8)
	if !ok {
		t.Fatal("deleteWordBeforeCursor returned !ok")
	}
	if string(got) != "abc ghi" {
		t.Fatalf("deleteWordBeforeCursor = %q, want %q", string(got), "abc ghi")
	}
	if cursor != 4 {
		t.Fatalf("deleteWordBeforeCursor cursor = %d, want 4", cursor)
	}
}

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
