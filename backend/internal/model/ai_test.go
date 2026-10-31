package model

import "testing"

func TestEmptyTranscriptRemainsEmpty(t *testing.T) {
	if got := Transcribe(""); got != "" {
		t.Fatalf("empty transcript became %q", got)
	}
}
