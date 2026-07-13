package trace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ParseFile reads a .json trace or a .jsonl file containing one Span per line.
func ParseFile(path string) (Trace, error) {
	f, err := os.Open(path)
	if err != nil {
		return Trace{}, fmt.Errorf("open trace file: %w", err)
	}
	defer f.Close()
	if strings.EqualFold(filepath.Ext(path), ".jsonl") {
		return ParseJSONL(f)
	}
	return ParseJSON(f)
}

// ParseJSON parses and strictly validates one Trace JSON object.
func ParseJSON(r io.Reader) (Trace, error) {
	var t Trace
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return Trace{}, fmt.Errorf("malformed trace JSON: %w", err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return Trace{}, fmt.Errorf("malformed trace JSON: multiple JSON values")
	}
	if err := validateTrace(t); err != nil {
		return Trace{}, err
	}
	return t, nil
}

// ParseJSONL parses one strictly validated Span object per non-empty line.
// JSONL has no run-level envelope, so its trace ID is "jsonl" and its bounds
// are derived from the earliest and latest span.
func ParseJSONL(r io.Reader) (Trace, error) {
	t := Trace{SchemaVersion: "1.0", TraceID: "jsonl", Spans: []Span{}}
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 4096), 4*1024*1024)
	line := 0
	for s.Scan() {
		line++
		data := bytes.TrimSpace(s.Bytes())
		if len(data) == 0 {
			continue
		}
		var span Span
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&span); err != nil {
			return Trace{}, fmt.Errorf("malformed JSONL at line %d: %w", line, err)
		}
		if dec.Decode(&struct{}{}) != io.EOF {
			return Trace{}, fmt.Errorf("malformed JSONL at line %d: multiple JSON values", line)
		}
		if err := validateSpan(span); err != nil {
			return Trace{}, fmt.Errorf("invalid span at line %d: %w", line, err)
		}
		t.Spans = append(t.Spans, span)
	}
	if err := s.Err(); err != nil {
		return Trace{}, fmt.Errorf("read JSONL: %w", err)
	}
	if len(t.Spans) > 0 {
		t.StartedAt, t.EndedAt = t.Spans[0].StartedAt, t.Spans[0].EndedAt
		for _, span := range t.Spans[1:] {
			if span.StartedAt.Before(t.StartedAt) {
				t.StartedAt = span.StartedAt
			}
			if span.EndedAt.After(t.EndedAt) {
				t.EndedAt = span.EndedAt
			}
		}
	}
	if err := validateUniqueIDs(t.Spans); err != nil {
		return Trace{}, err
	}
	return t, nil
}

func validateTrace(t Trace) error {
	if t.TraceID == "" {
		return fmt.Errorf("missing required field: trace_id")
	}
	if t.StartedAt.IsZero() {
		return fmt.Errorf("missing required field: started_at")
	}
	if t.EndedAt.IsZero() {
		return fmt.Errorf("missing required field: ended_at")
	}
	if t.EndedAt.Before(t.StartedAt) {
		return fmt.Errorf("trace ended_at is before started_at")
	}
	for i, span := range t.Spans {
		if err := validateSpan(span); err != nil {
			return fmt.Errorf("invalid span %d: %w", i, err)
		}
	}
	return validateUniqueIDs(t.Spans)
}

func validateUniqueIDs(spans []Span) error {
	seen := make(map[string]struct{}, len(spans))
	for _, span := range spans {
		if _, ok := seen[span.SpanID]; ok {
			return fmt.Errorf("duplicate span_id: %s", span.SpanID)
		}
		seen[span.SpanID] = struct{}{}
	}
	return nil
}

func validateSpan(s Span) error {
	if s.SpanID == "" {
		return fmt.Errorf("missing required field: span_id")
	}
	if s.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if s.Kind == "" {
		return fmt.Errorf("missing required field: kind")
	}
	if !validKind(s.Kind) {
		return fmt.Errorf("unknown kind: %q", s.Kind)
	}
	if s.Status == "" {
		return fmt.Errorf("missing required field: status")
	}
	if s.Status != "ok" && s.Status != "error" {
		return fmt.Errorf("unknown status: %q", s.Status)
	}
	if s.StartedAt.IsZero() {
		return fmt.Errorf("missing required field: started_at")
	}
	if s.EndedAt.IsZero() {
		return fmt.Errorf("missing required field: ended_at")
	}
	if s.EndedAt.Before(s.StartedAt) {
		return fmt.Errorf("span %q ended_at is before started_at", s.SpanID)
	}
	return nil
}

func validKind(kind string) bool {
	switch kind {
	case "llm", "tool", "plan", "verify", "memory", "other":
		return true
	}
	return false
}
