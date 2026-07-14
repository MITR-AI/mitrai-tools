// Package conversation defines and reads recorded AI-agent conversations.
package conversation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Conversation is a deterministic recording of dialogue turns.
type Conversation struct {
	SchemaVersion  string    `json:"schema_version"`
	ConversationID string    `json:"conversation_id"`
	StartedAt      time.Time `json:"started_at"`
	Meta           Meta      `json:"meta"`
	Turns          []Turn    `json:"turns"`
}

type Meta struct {
	Agent  string `json:"agent"`
	Model  string `json:"model"`
	Tenant string `json:"tenant,omitempty"`
}

type Turn struct {
	Index     int            `json:"index"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []ToolCall     `json:"tool_calls,omitempty"`
	Model     string         `json:"model,omitempty"`
	Tokens    *Tokens        `json:"tokens,omitempty"`
	State     map[string]any `json:"state,omitempty"`
	At        time.Time      `json:"at"`
}

type ToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
}

type Tokens struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

// Parse reads either one Conversation JSON object or JSONL containing one Turn
// object per non-empty line. JSONL has no envelope, so its ConversationID is
// "jsonl" unless the optional conversation_id field is repeated on a turn.
func Parse(r io.Reader) (Conversation, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return Conversation{}, fmt.Errorf("read conversation: %w", err)
	}
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return Conversation{}, fmt.Errorf("malformed JSON: empty input")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(b, &object) == nil {
		if _, isConversation := object["turns"]; isConversation {
			return parseConversation(b)
		}
	}
	return parseJSONL(b)
}

func parseConversation(b []byte) (Conversation, error) {
	var raw struct {
		SchemaVersion  string            `json:"schema_version"`
		ConversationID string            `json:"conversation_id"`
		StartedAt      time.Time         `json:"started_at"`
		Meta           Meta              `json:"meta"`
		Turns          []json.RawMessage `json:"turns"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Conversation{}, fmt.Errorf("malformed JSON: %w", err)
	}
	if strings.TrimSpace(raw.ConversationID) == "" {
		return Conversation{}, fmt.Errorf("missing required field conversation_id")
	}
	if raw.Turns == nil {
		return Conversation{}, fmt.Errorf("missing required field turns")
	}
	c := Conversation{SchemaVersion: raw.SchemaVersion, ConversationID: raw.ConversationID, StartedAt: raw.StartedAt, Meta: raw.Meta}
	for line, turnRaw := range raw.Turns {
		turn, err := parseTurn(turnRaw)
		if err != nil {
			return Conversation{}, fmt.Errorf("turn %d: %w", line, err)
		}
		c.Turns = append(c.Turns, turn)
	}
	if err := Validate(c); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

func parseJSONL(b []byte) (Conversation, error) {
	lines := bytes.Split(b, []byte{'\n'})
	c := Conversation{SchemaVersion: "1.0", ConversationID: "jsonl"}
	for i, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var envelope struct {
			ConversationID string `json:"conversation_id"`
			SchemaVersion  string `json:"schema_version"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return Conversation{}, fmt.Errorf("malformed JSONL at line %d: %w", i+1, err)
		}
		turn, err := parseTurn(line)
		if err != nil {
			return Conversation{}, fmt.Errorf("turn at line %d: %w", i+1, err)
		}
		if envelope.ConversationID != "" {
			if c.ConversationID != "jsonl" && c.ConversationID != envelope.ConversationID {
				return Conversation{}, fmt.Errorf("JSONL has inconsistent conversation_id")
			}
			c.ConversationID = envelope.ConversationID
		}
		if envelope.SchemaVersion != "" {
			c.SchemaVersion = envelope.SchemaVersion
		}
		c.Turns = append(c.Turns, turn)
	}
	if err := Validate(c); err != nil {
		return Conversation{}, err
	}
	return c, nil
}

func parseTurn(raw []byte) (Turn, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Turn{}, fmt.Errorf("malformed JSON: %w", err)
	}
	for _, key := range []string{"index", "role", "content"} {
		if _, ok := fields[key]; !ok {
			return Turn{}, fmt.Errorf("missing required field %s", key)
		}
	}
	var t Turn
	if err := json.Unmarshal(raw, &t); err != nil {
		return Turn{}, fmt.Errorf("malformed turn: %w", err)
	}
	return t, nil
}

// Validate enforces the format invariants after decoding.
func Validate(c Conversation) error {
	if strings.TrimSpace(c.ConversationID) == "" {
		return fmt.Errorf("missing required field conversation_id")
	}
	seen := make(map[int]bool, len(c.Turns))
	previous := 0
	for pos, t := range c.Turns {
		if !validRole(t.Role) {
			return fmt.Errorf("turn %d: unknown role %q", t.Index, t.Role)
		}
		if seen[t.Index] {
			return fmt.Errorf("duplicate turn index %d", t.Index)
		}
		seen[t.Index] = true
		if pos > 0 && t.Index <= previous {
			return fmt.Errorf("turns not in ascending index order: %d follows %d", t.Index, previous)
		}
		previous = t.Index
		for _, call := range t.ToolCalls {
			if strings.TrimSpace(call.Name) == "" {
				return fmt.Errorf("turn %d: tool_call has empty name", t.Index)
			}
		}
	}
	return nil
}

func validRole(role string) bool {
	switch role {
	case "user", "assistant", "tool", "system":
		return true
	default:
		return false
	}
}

// StateAt returns the most recent state snapshot at or before index.
func (c Conversation) StateAt(index int) (map[string]any, bool) {
	var state map[string]any
	found := false
	for _, t := range c.Turns {
		if t.Index > index {
			break
		}
		if t.State != nil {
			state, found = t.State, true
		}
	}
	return state, found
}

// TurnAt finds a turn by its recorded index.
func (c Conversation) TurnAt(index int) (Turn, bool) {
	i := sort.Search(len(c.Turns), func(i int) bool { return c.Turns[i].Index >= index })
	if i < len(c.Turns) && c.Turns[i].Index == index {
		return c.Turns[i], true
	}
	return Turn{}, false
}
