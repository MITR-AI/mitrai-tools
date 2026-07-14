package conversation

import (
	"strings"
	"testing"
)

const validObject = `{"schema_version":"1.0","conversation_id":"c1","started_at":"2026-01-01T00:00:00Z","meta":{"agent":"a","model":"m"},"turns":[{"index":0,"role":"user","content":"hello","at":"2026-01-01T00:00:00Z"},{"index":1,"role":"assistant","content":"hi","tool_calls":[{"name":"lookup"}],"state":{"x":"y"},"at":"2026-01-01T00:00:01Z"}]}`

func TestParse(t *testing.T) {
	tests := []struct {
		name, input, want string
		turns             int
	}{
		{"single object", validObject, "", 2},
		{"jsonl", `{"index":0,"role":"user","content":"hello"}` + "\n" + `{"index":1,"role":"assistant","content":"hi"}`, "", 2},
		{"malformed", `{`, "malformed JSONL", 0},
		{"missing conversation id", `{"turns":[]}`, "missing required field conversation_id", 0},
		{"missing index", `{"conversation_id":"c","turns":[{"role":"user","content":"x"}]}`, "missing required field index", 0},
		{"missing role", `{"conversation_id":"c","turns":[{"index":0,"content":"x"}]}`, "missing required field role", 0},
		{"missing content", `{"conversation_id":"c","turns":[{"index":0,"role":"user"}]}`, "missing required field content", 0},
		{"bad role", `{"conversation_id":"c","turns":[{"index":0,"role":"robot","content":"x"}]}`, "unknown role", 0},
		{"duplicate index", `{"conversation_id":"c","turns":[{"index":0,"role":"user","content":"x"},{"index":0,"role":"assistant","content":"y"}]}`, "duplicate turn index", 0},
		{"out of order", `{"conversation_id":"c","turns":[{"index":2,"role":"user","content":"x"},{"index":1,"role":"assistant","content":"y"}]}`, "not in ascending", 0},
		{"empty tool name", `{"conversation_id":"c","turns":[{"index":0,"role":"assistant","content":"x","tool_calls":[{}]}]}`, "empty name", 0},
		{"empty turns", `{"conversation_id":"c","turns":[]}`, "", 0},
		{"no state unicode", `{"conversation_id":"c","turns":[{"index":0,"role":"user","content":"नमस्ते 👋"}]}`, "", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(strings.NewReader(tt.input))
			if tt.want != "" {
				if err == nil || !strings.Contains(err.Error(), tt.want) {
					t.Fatalf("error = %v, want substring %q", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Turns) != tt.turns {
				t.Fatalf("turns = %d, want %d", len(got.Turns), tt.turns)
			}
		})
	}
}

func TestParsePreservesUnicodeContent(t *testing.T) {
	c, err := Parse(strings.NewReader(`{"conversation_id":"c","turns":[{"index":0,"role":"user","content":"नमस्ते 👋"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := c.Turns[0].Content, "नमस्ते 👋"; got != want {
		t.Fatalf("content = %q, want %q", got, want)
	}
}

func TestStateAt(t *testing.T) {
	c, err := Parse(strings.NewReader(validObject))
	if err != nil {
		t.Fatal(err)
	}
	if state, ok := c.StateAt(1); !ok || state["x"] != "y" {
		t.Fatalf("state = %#v, ok = %v", state, ok)
	}
	if _, ok := c.StateAt(0); ok {
		t.Fatal("unexpected state before snapshot")
	}
}
