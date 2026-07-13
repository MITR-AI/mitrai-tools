// Package trace defines and reads the local Agent Trace Viewer trace format.
package trace

import "time"

// Meta identifies the producer of a trace. Tenant is optional.
type Meta struct {
	Agent  string `json:"agent"`
	Model  string `json:"model"`
	Tenant string `json:"tenant,omitempty"`
}

// Tokens holds input and output token counts for a span.
type Tokens struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

// Span is one timed operation in an agent run.
type Span struct {
	SpanID       string         `json:"span_id"`
	ParentSpanID string         `json:"parent_span_id,omitempty"`
	Name         string         `json:"name"`
	Kind         string         `json:"kind"`
	Status       string         `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      time.Time      `json:"ended_at"`
	Attributes   map[string]any `json:"attributes,omitempty"`
	Tokens       *Tokens        `json:"tokens,omitempty"`
	CostUSD      float64        `json:"cost_usd,omitempty"`
	Error        string         `json:"error,omitempty"`
}

// Duration returns the elapsed duration of a span.
func (s Span) Duration() time.Duration { return s.EndedAt.Sub(s.StartedAt) }

// Trace is one complete agent run.
type Trace struct {
	SchemaVersion string    `json:"schema_version"`
	TraceID       string    `json:"trace_id"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
	Meta          Meta      `json:"meta"`
	Spans         []Span    `json:"spans"`
}

// Duration returns the wall-clock duration of a trace.
func (t Trace) Duration() time.Duration { return t.EndedAt.Sub(t.StartedAt) }
