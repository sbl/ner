package ner_test

import (
	"testing"

	"github.com/sbl/ner"
)

func TestTokenize(t *testing.T) {
	txt := "I am a precious snowflake"
	ts := ner.Tokenize(txt)
	got := len(ts)
	if got != 5 {
		t.Errorf("Expected 5 tokens, have: %d", got)
	}
}
