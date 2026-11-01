package api

import (
	"strings"
	"testing"

	"copium/api/internal/ai"
)

func TestSanitizeHistory(t *testing.T) {
	// Leading model turns are dropped so the conversation starts with the user.
	got, err := sanitizeHistory([]ai.Message{
		{Role: "model", Text: "welcome"},
		{Role: "user", Text: "  hi  "},
		{Role: "user", Text: "   "}, // blank, dropped
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Text != "hi" {
		t.Fatalf("unexpected history %+v", got)
	}

	if _, err := sanitizeHistory([]ai.Message{{Role: "system", Text: "x"}}); err == nil {
		t.Error("expected error for invalid role")
	}
	if _, err := sanitizeHistory([]ai.Message{{Role: "user", Text: "q"}, {Role: "model", Text: "a"}}); err == nil {
		t.Error("expected error when the last message is not from the user")
	}

	long := []ai.Message{{Role: "user", Text: strings.Repeat("字", maxMessageRunes+10)}}
	got, _ = sanitizeHistory(long)
	if n := len([]rune(got[0].Text)); n != maxMessageRunes {
		t.Errorf("expected text trimmed to %d runes, got %d", maxMessageRunes, n)
	}

	var many []ai.Message
	for i := 0; i < 30; i++ {
		many = append(many, ai.Message{Role: "user", Text: "q"})
	}
	got, _ = sanitizeHistory(many)
	if len(got) != maxChatMessages {
		t.Errorf("expected %d messages, got %d", maxChatMessages, len(got))
	}
}
