package trace

import (
	"strings"
	"testing"
	"time"
)

const validTrace = `{"schema_version":"1.0","trace_id":"run","started_at":"2026-01-01T00:00:00Z","ended_at":"2026-01-01T00:00:03Z","meta":{"agent":"a","model":"m"},"spans":[{"span_id":"one","name":"one","kind":"llm","status":"ok","started_at":"2026-01-01T00:00:00Z","ended_at":"2026-01-01T00:00:01Z"}]}`

func TestParseJSONFailures(t *testing.T) {
	tests := []struct{ name, input, want string }{
		{"malformed", `{`, "malformed trace JSON"},
		{"missing trace id", strings.Replace(validTrace, `"trace_id":"run",`, "", 1), "trace_id"},
		{"missing trace start", strings.Replace(validTrace, `"started_at":"2026-01-01T00:00:00Z",`, "", 1), "started_at"},
		{"missing trace end", strings.Replace(validTrace, `,"ended_at":"2026-01-01T00:00:03Z","meta"`, `,"meta"`, 1), "ended_at"},
		{"missing span id", strings.Replace(validTrace, `"span_id":"one",`, "", 1), "span_id"},
		{"missing name", strings.Replace(validTrace, `"name":"one",`, "", 1), "name"},
		{"missing kind", strings.Replace(validTrace, `"kind":"llm",`, "", 1), "kind"},
		{"missing status", strings.Replace(validTrace, `"status":"ok",`, "", 1), "status"},
		{"missing span start", strings.Replace(validTrace, `"started_at":"2026-01-01T00:00:00Z",`, "", 2), "started_at"},
		{"missing span end", strings.Replace(validTrace, `,"ended_at":"2026-01-01T00:00:01Z"}`, `}`, 1), "ended_at"},
		{"bad kind", strings.Replace(validTrace, `"kind":"llm"`, `"kind":"wat"`, 1), "unknown kind"},
		{"bad status", strings.Replace(validTrace, `"status":"ok"`, `"status":"waiting"`, 1), "unknown status"},
		{"span backwards", strings.Replace(validTrace, `"ended_at":"2026-01-01T00:00:01Z"`, `"ended_at":"2025-12-31T23:59:59Z"`, 1), "before started_at"},
		{"duplicate span id", strings.Replace(validTrace, `]}`, `,{"span_id":"one","name":"two","kind":"tool","status":"ok","started_at":"2026-01-01T00:00:01Z","ended_at":"2026-01-01T00:00:02Z"}]}`, 1), "duplicate span_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON(strings.NewReader(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestParseJSONLFailure(t *testing.T) {
	_, err := ParseJSONL(strings.NewReader(`{"span_id":"one"} not-json`))
	if err == nil || !strings.Contains(err.Error(), "malformed JSONL at line 1") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseFixturesAndEdges(t *testing.T) {
	simple, err := ParseFile("testdata/simple.json")
	if err != nil {
		t.Fatal(err)
	}
	if simple.TraceID != "simple-run" || len(simple.Spans) != 2 {
		t.Fatalf("unexpected simple trace: %#v", simple)
	}
	if got := simple.Duration(); got != 3*time.Second {
		t.Fatalf("duration = %s", got)
	}
	multi, err := ParseFile("testdata/multiagent.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if multi.TraceID != "jsonl" || len(multi.Spans) != 5 || multi.Duration() != 6*time.Second {
		t.Fatalf("unexpected jsonl trace: %#v", multi)
	}
	empty := strings.Replace(validTrace, `[{"span_id":"one","name":"one","kind":"llm","status":"ok","started_at":"2026-01-01T00:00:00Z","ended_at":"2026-01-01T00:00:01Z"}]`, `[]`, 1)
	if _, err := ParseJSON(strings.NewReader(empty)); err != nil {
		t.Fatalf("empty spans: %v", err)
	}
	zero := strings.Replace(validTrace, `"ended_at":"2026-01-01T00:00:01Z"`, `"ended_at":"2026-01-01T00:00:00Z"`, 1)
	got, err := ParseJSON(strings.NewReader(zero))
	if err != nil || got.Spans[0].Duration() != 0 {
		t.Fatalf("zero duration: %v, %s", err, got.Spans[0].Duration())
	}
	orphan := strings.Replace(validTrace, `"name":"one",`, `"parent_span_id":"missing","name":"one",`, 1)
	got, err = ParseJSON(strings.NewReader(orphan))
	if err != nil {
		t.Fatal(err)
	}
	tree := BuildTree(got)
	if len(tree.Orphans) != 1 || !tree.Roots[0].Orphan {
		t.Fatalf("orphan was not reported: %#v", tree)
	}
}

func TestBuildTree(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	span := func(id, parent string, sec int) Span {
		return Span{SpanID: id, ParentSpanID: parent, Name: id, Kind: "other", Status: "ok", StartedAt: base.Add(time.Duration(sec) * time.Second), EndedAt: base.Add(time.Duration(sec+1) * time.Second)}
	}
	tree := BuildTree(Trace{Spans: []Span{span("child", "root", 2), span("root", "", 1), span("other", "", 0), span("grand", "child", 3), span("lost", "gone", 4)}})
	if len(tree.Roots) != 3 || tree.Roots[0].Span.SpanID != "other" || tree.Roots[1].Span.SpanID != "root" {
		t.Fatalf("roots: %#v", tree.Roots)
	}
	if tree.Roots[1].Children[0].Span.SpanID != "child" || tree.Roots[1].Children[0].Children[0].Span.SpanID != "grand" {
		t.Fatalf("nesting: %#v", tree.Roots[1])
	}
	if len(tree.Orphans) != 1 || tree.Orphans[0].Span.SpanID != "lost" {
		t.Fatalf("orphans: %#v", tree.Orphans)
	}
}

func TestSummary(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	span := func(name, kind, status string, seconds int, in, out int, cost float64) Span {
		return Span{Name: name, SpanID: name, Kind: kind, Status: status, StartedAt: base, EndedAt: base.Add(time.Duration(seconds) * time.Second), Tokens: &Tokens{In: in, Out: out}, CostUSD: cost}
	}
	tr := Trace{StartedAt: base, EndedAt: base.Add(10 * time.Second), Spans: []Span{span("two", "tool", "ok", 2, 2, 3, .1), span("five", "llm", "error", 5, 5, 7, .2), span("one", "tool", "ok", 1, 11, 13, .3), span("three", "verify", "ok", 3, 17, 19, .4)}}
	s := Summarize(tr)
	if s.WallDuration != int64(10*time.Second) || s.SpanCount != 4 || s.ByKind["tool"] != 2 || s.TokensIn != 35 || s.TokensOut != 42 || s.CostUSD != 1 || s.ErrorCount != 1 {
		t.Fatalf("bad summary: %#v", s)
	}
	if len(s.Slowest) != 3 || s.Slowest[0].Name != "five" || s.Slowest[1].Name != "three" || s.Slowest[2].Name != "two" {
		t.Fatalf("slowest: %#v", s.Slowest)
	}
}
