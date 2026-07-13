// trace-view is a local, dependency-free viewer for AI agent run traces.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MITR-AI/agent-trace-viewer/trace"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		usage(out)
		return nil
	}
	switch args[0] {
	case "show":
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			usage(out)
			return nil
		}
		if len(args) != 2 {
			usage(errOut)
			return errors.New("show requires one trace file")
		}
		t, err := trace.ParseFile(args[1])
		if err != nil {
			return err
		}
		_, err = io.WriteString(out, renderTree(t, isTTY(out)))
		return err
	case "summary":
		if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
			usage(out)
			return nil
		}
		if len(args) != 2 {
			usage(errOut)
			return errors.New("summary requires one trace file")
		}
		t, err := trace.ParseFile(args[1])
		if err != nil {
			return err
		}
		_, err = io.WriteString(out, renderSummary(trace.Summarize(t)))
		return err
	case "serve":
		return serve(args[1:], out, errOut)
	default:
		usage(errOut)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `Agent Trace Viewer

Usage:
  trace-view show <file>
  trace-view summary <file>
  trace-view serve <file> [--port 8099]
`)
}

func serve(args []string, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		usage(out)
		return nil
	}
	if len(args) == 2 && (args[1] == "-h" || args[1] == "--help") {
		usage(out)
		return nil
	}
	t, err := trace.ParseFile(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(errOut)
	port := fs.Int("port", 8099, "HTTP port")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return errors.New("serve accepts only one trace file")
	}
	if *port < 1 || *port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, viewerHTML)
	})
	mux.HandleFunc("/api/trace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := writeJSON(w, t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	addr := ":" + strconv.Itoa(*port)
	fmt.Fprintf(out, "Agent Trace Viewer: http://localhost:%d\n", *port)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w io.Writer, value any) error {
	return json.NewEncoder(w).Encode(value)
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func renderTree(t trace.Trace, color bool) string {
	var b strings.Builder
	for _, root := range trace.BuildTree(t).Roots {
		writeNode(&b, root, 0, color)
	}
	return b.String()
}

func writeNode(b *strings.Builder, node *trace.Node, depth int, color bool) {
	s := node.Span
	b.WriteString(strings.Repeat("  ", depth))
	if node.Orphan {
		b.WriteString(label("[ORPHAN] ", "33", color))
	}
	b.WriteString(s.Name)
	fmt.Fprintf(b, " [%s] %s %s", s.Kind, statusText(s.Status, color), s.Duration().Round(time.Microsecond))
	in, out := 0, 0
	if s.Tokens != nil {
		in, out = s.Tokens.In, s.Tokens.Out
	}
	fmt.Fprintf(b, " tokens=%d/%d cost=$%.6f", in, out, s.CostUSD)
	if s.Status == "error" && s.Error != "" {
		fmt.Fprintf(b, " — %s", shortError(s.Error))
	}
	b.WriteByte('\n')
	for _, child := range node.Children {
		writeNode(b, child, depth+1, color)
	}
}

func statusText(status string, color bool) string {
	if status == "error" {
		return label(status, "31", color)
	}
	return label(status, "32", color)
}

func label(s, code string, color bool) string {
	if !color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func shortError(err string) string {
	err = strings.Join(strings.Fields(err), " ")
	const max = 80
	if len(err) > max {
		return err[:max-1] + "…"
	}
	return err
}

func renderSummary(s trace.Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Wall-clock duration: %s\n", time.Duration(s.WallDuration).Round(time.Microsecond))
	fmt.Fprintf(&b, "Spans: %d\n", s.SpanCount)
	b.WriteString("By kind:")
	kinds := make([]string, 0, len(s.ByKind))
	for kind := range s.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		fmt.Fprintf(&b, " %s=%d", kind, s.ByKind[kind])
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Tokens: in=%d out=%d\n", s.TokensIn, s.TokensOut)
	fmt.Fprintf(&b, "Cost: $%.6f\n", s.CostUSD)
	fmt.Fprintf(&b, "Errors: %d\n", s.ErrorCount)
	b.WriteString("Slowest spans:\n")
	for i, slow := range s.Slowest {
		fmt.Fprintf(&b, "  %d. %s — %s\n", i+1, slow.Name, time.Duration(slow.Duration).Round(time.Microsecond))
	}
	return b.String()
}
