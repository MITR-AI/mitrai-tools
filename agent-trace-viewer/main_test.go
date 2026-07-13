package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/MITR-AI/agent-trace-viewer/trace"
)

func TestCLIHelpers(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tc := trace.Trace{StartedAt: start, EndedAt: start.Add(2 * time.Second), Spans: []trace.Span{
		{SpanID: "root", Name: "root", Kind: "plan", Status: "ok", StartedAt: start, EndedAt: start.Add(time.Second), Tokens: &trace.Tokens{In: 1, Out: 2}},
		{SpanID: "lost", ParentSpanID: "missing", Name: "lost", Kind: "tool", Status: "error", StartedAt: start.Add(time.Second), EndedAt: start.Add(2 * time.Second), Error: "a very useful failure"},
	}}
	show := renderTree(tc, false)
	for _, want := range []string{"root [plan] ok", "[ORPHAN] lost [tool] error", "a very useful failure"} {
		if !strings.Contains(show, want) {
			t.Errorf("show missing %q:\n%s", want, show)
		}
	}
	if strings.Contains(show, "\x1b[") {
		t.Error("plain render contains ANSI")
	}
	if !strings.Contains(renderTree(tc, true), "\x1b[32m") {
		t.Error("colored render lacks ANSI")
	}
	summary := renderSummary(trace.Summarize(tc))
	for _, want := range []string{"Wall-clock duration: 2s", "Spans: 2", "Errors: 1", "Slowest spans:"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary missing %q", want)
		}
	}
}

func TestRunUsageAndErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"-h"}, {"show", "-h"}, {"summary", "--help"}, {"serve", "-h"}, {"unknown"}} {
		var out, errOut bytes.Buffer
		err := run(args, &out, &errOut)
		if !strings.Contains(out.String()+errOut.String(), "Usage:") {
			t.Fatalf("args %v did not print usage", args)
		}
		if len(args) < 1 && err != nil {
			t.Fatalf("empty args: %v", err)
		}
	}
}
