package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MITR-AI/conversation-replay/conversation"
)

func fixture(t *testing.T, name string) conversation.Conversation {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	c, err := conversation.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRenderAt(t *testing.T) {
	c := fixture(t, "simple.json")
	tests := []struct {
		name string
		turn int
		want string
		err  string
	}{
		{"resolves latest state", 2, `"stage": "looked-up"`, ""},
		{"before state", 0, "(no state snapshot)", ""},
		{"out of range", 9, "", "out of range"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			err := renderAt(&b, c, tt.turn)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(b.String(), tt.want) {
				t.Fatalf("output %q does not contain %q", b.String(), tt.want)
			}
		})
	}
}

func TestRenderDiff(t *testing.T) {
	a := fixture(t, "simple.json")
	tests := []struct {
		name      string
		other     conversation.Conversation
		different bool
		wants     []string
	}{
		{"identical", a, false, []string{"No differences."}},
		{"divergences", fixture(t, "diverging.jsonl"), true, []string{"turn 1 content differs", "turn 1 tool calls differ", "turn 3 only in B"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			if got := renderDiff(&b, a, tt.other); got != tt.different {
				t.Fatalf("different=%v, want %v", got, tt.different)
			}
			for _, want := range tt.wants {
				if !strings.Contains(b.String(), want) {
					t.Errorf("output missing %q: %s", want, b.String())
				}
			}
		})
	}
}

func TestRunDiffExitCode(t *testing.T) {
	var out, errs bytes.Buffer
	code := run([]string{"diff", "testdata/simple.json", "testdata/diverging.jsonl"}, &out, &errs)
	if code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
}

func TestRunAtFileThenFlag(t *testing.T) {
	var out, errs bytes.Buffer
	code := run([]string{"at", "testdata/simple.json", "--turn", "2"}, &out, &errs)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errs.String())
	}
	if !strings.Contains(out.String(), "Transcript through turn 2") {
		t.Fatalf("output=%s", out.String())
	}
}

func TestRunShow(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run([]string{"show", "testdata/simple.json"}, &out, &errs); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errs.String())
	}
	for _, want := range []string{"Conversation demo-001 (3 turns)", "[0] user:", "tools: search", "tokens: in=12"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRunExitCodesAndErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"no args → help", nil, 0},
		{"help", []string{"help"}, 0},
		{"unknown subcommand", []string{"bogus"}, 2},
		{"show wrong arg count", []string{"show"}, 2},
		{"diff wrong arg count", []string{"diff", "a"}, 2},
		{"at bad flag (negative)", []string{"at", "testdata/simple.json", "--turn", "-1"}, 2},
		{"show missing file", []string{"show", "testdata/nope.json"}, 1},
		{"at missing file", []string{"at", "testdata/nope.json", "--turn", "0"}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errs bytes.Buffer
			if code := run(tc.args, &out, &errs); code != tc.want {
				t.Fatalf("code=%d, want %d (stderr=%s)", code, tc.want, errs.String())
			}
		})
	}
}

func TestRenderConversationColorAndPreview(t *testing.T) {
	c := fixture(t, "simple.json")
	// long content → preview truncates; color=true → ANSI role wrapping.
	c.Turns[0].Content = strings.Repeat("word ", 40)
	var b bytes.Buffer
	renderConversation(&b, c, true)
	out := b.String()
	if !strings.Contains(out, "\x1b[36m") {
		t.Fatalf("color role not applied: %q", out)
	}
	if !strings.Contains(out, "...") {
		t.Fatalf("long content not truncated by preview: %q", out)
	}
	if !strings.Contains(out, "tools: search") {
		t.Fatalf("tool names not rendered: %q", out)
	}
}

func TestPreview(t *testing.T) {
	if got := preview("  a   b\tc  "); got != "a b c" {
		t.Fatalf("whitespace collapse: %q", got)
	}
	long := strings.Repeat("x", 200)
	if got := preview(long); len([]rune(got)) != 80 || !strings.HasSuffix(got, "...") {
		t.Fatalf("truncation: len=%d suffix=%q", len([]rune(got)), got[len(got)-3:])
	}
}

func TestReplayHandler(t *testing.T) {
	c := fixture(t, "simple.json")
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(replayHandler(data))
	defer srv.Close()

	get := func(path string) (*http.Response, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp, string(body)
	}

	if resp, body := get("/"); resp.StatusCode != 200 || !strings.Contains(body, "Conversation Replay") {
		t.Fatalf("index page: status=%d", resp.StatusCode)
	}
	if resp, body := get("/api/conversation"); resp.StatusCode != 200 || !strings.Contains(body, "demo-001") {
		t.Fatalf("api: status=%d body=%s", resp.StatusCode, body)
	}
	if resp, _ := get("/nope"); resp.StatusCode != 404 {
		t.Fatalf("unknown path should 404, got %d", resp.StatusCode)
	}
}
