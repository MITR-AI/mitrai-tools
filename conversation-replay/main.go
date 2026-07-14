package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/MITR-AI/conversation-replay/conversation"
)

const usageText = `Conversation Replay — inspect recorded AI agent conversations

Usage:
  convreplay show <file>
  convreplay at <file> --turn N
  convreplay diff <a> <b>
  convreplay serve <file> [--port 8110]
`

// pf/pl/ps write to a CLI output writer, ignoring the (unrecoverable) write error to stdout/stderr —
// so call sites stay readable and errcheck stays green.
func pf(w io.Writer, format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
func pl(w io.Writer, a ...any)                { _, _ = fmt.Fprintln(w, a...) }
func ps(w io.Writer, s string)                { _, _ = fmt.Fprint(w, s) }

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		ps(out, usageText)
		return 0
	}
	switch args[0] {
	case "show":
		if len(args) != 2 {
			return usageError(errOut)
		}
		c, err := readConversation(args[1])
		if err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		renderConversation(out, c, isTTY(out))
		return 0
	case "at":
		fs := flag.NewFlagSet("at", flag.ContinueOnError)
		fs.SetOutput(errOut)
		turn := fs.Int("turn", -1, "turn index")
		if err := fs.Parse(flagsFirst(args[1:], "--turn")); err != nil || fs.NArg() != 1 || *turn < 0 {
			return usageError(errOut)
		}
		c, err := readConversation(fs.Arg(0))
		if err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		if err := renderAt(out, c, *turn); err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		return 0
	case "diff":
		if len(args) != 3 {
			return usageError(errOut)
		}
		a, err := readConversation(args[1])
		if err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		b, err := readConversation(args[2])
		if err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		if renderDiff(out, a, b) {
			return 1
		}
		return 0
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		fs.SetOutput(errOut)
		port := fs.Int("port", 8110, "HTTP port")
		if err := fs.Parse(flagsFirst(args[1:], "--port")); err != nil || fs.NArg() != 1 || *port < 1 || *port > 65535 {
			return usageError(errOut)
		}
		c, err := readConversation(fs.Arg(0))
		if err != nil {
			pl(errOut, "error:", err)
			return 1
		}
		return serve(out, c, *port)
	default:
		pf(errOut, "error: unknown subcommand %q\n", args[0])
		return usageError(errOut)
	}
}

func usageError(w io.Writer) int { ps(w, usageText); return 2 }

// flagsFirst supports the familiar "command file --flag value" form while
// retaining the standard library flag package as the command-line parser.
func flagsFirst(args []string, valueFlag string) []string {
	flags, positional := make([]string, 0, len(args)), make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == valueFlag {
			flags = append(flags, args[i])
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, args[i])
	}
	return append(flags, positional...)
}

func readConversation(path string) (conversation.Conversation, error) {
	f, err := os.Open(path)
	if err != nil {
		return conversation.Conversation{}, err
	}
	defer func() { _ = f.Close() }()
	return conversation.Parse(f)
}

func renderConversation(w io.Writer, c conversation.Conversation, color bool) {
	pf(w, "Conversation %s (%d turns)\n", c.ConversationID, len(c.Turns))
	for _, t := range c.Turns {
		role := t.Role
		if color {
			role = colorRole(role)
		}
		pf(w, "[%d] %s: %s", t.Index, role, preview(t.Content))
		if len(t.ToolCalls) > 0 {
			pf(w, "  tools: %s", toolNames(t.ToolCalls))
		}
		if t.Tokens != nil {
			pf(w, "  tokens: in=%d out=%d", t.Tokens.In, t.Tokens.Out)
		}
		pl(w)
	}
}

func renderAt(w io.Writer, c conversation.Conversation, index int) error {
	if _, ok := c.TurnAt(index); !ok {
		return fmt.Errorf("turn %d is out of range", index)
	}
	pf(w, "Transcript through turn %d\n", index)
	for _, t := range c.Turns {
		if t.Index <= index {
			pf(w, "[%d] %s: %s\n", t.Index, t.Role, t.Content)
		}
	}
	pl(w, "State:")
	state, ok := c.StateAt(index)
	if !ok {
		pl(w, "(no state snapshot)")
		return nil
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	pl(w, string(b))
	return nil
}

func renderDiff(w io.Writer, a, b conversation.Conversation) bool {
	ai, bi, different := 0, 0, false
	for ai < len(a.Turns) || bi < len(b.Turns) {
		if ai == len(a.Turns) {
			pf(w, "turn %d only in B\n", b.Turns[bi].Index)
			bi++
			different = true
			continue
		}
		if bi == len(b.Turns) {
			pf(w, "turn %d only in A\n", a.Turns[ai].Index)
			ai++
			different = true
			continue
		}
		x, y := a.Turns[ai], b.Turns[bi]
		if x.Index < y.Index {
			pf(w, "turn %d only in A\n", x.Index)
			ai++
			different = true
			continue
		}
		if y.Index < x.Index {
			pf(w, "turn %d only in B\n", y.Index)
			bi++
			different = true
			continue
		}
		if x.Role != y.Role {
			pf(w, "turn %d role differs: %q != %q\n", x.Index, x.Role, y.Role)
			different = true
		}
		if x.Content != y.Content {
			pf(w, "turn %d content differs\n", x.Index)
			different = true
		}
		if !reflect.DeepEqual(x.ToolCalls, y.ToolCalls) {
			pf(w, "turn %d tool calls differ\n", x.Index)
			different = true
		}
		ai++
		bi++
	}
	if !different {
		pl(w, "No differences.")
	}
	return different
}

func preview(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len([]rune(s)) > 80 {
		return string([]rune(s)[:77]) + "..."
	}
	return s
}

func toolNames(calls []conversation.ToolCall) string {
	n := make([]string, len(calls))
	for i, c := range calls {
		n[i] = c.Name
	}
	return strings.Join(n, ", ")
}

func colorRole(role string) string { return "\x1b[36m" + role + "\x1b[0m" }

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// replayHandler serves the conversation JSON at /api/conversation and the self-contained viewer at /.
// Extracted from serve so it is testable over httptest without binding a port.
func replayHandler(data []byte) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/conversation", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, pageHTML)
	})
	return mux
}

func serve(out io.Writer, c conversation.Conversation, port int) int {
	data, err := json.Marshal(c)
	if err != nil {
		pl(out, "error:", err)
		return 1
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	pf(out, "Conversation Replay listening at http://%s\n", address)
	if err := http.ListenAndServe(address, replayHandler(data)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		pl(out, "error:", err)
		return 1
	}
	return 0
}

const pageHTML = `<!doctype html><html><head><meta charset="utf-8"><title>Conversation Replay</title><style>body{font:16px system-ui;margin:2rem;max-width:900px;color:#18212f}input{width:100%}button{margin:.5rem}pre{white-space:pre-wrap;background:#f4f6f8;padding:1rem;border-radius:6px}.muted{color:#637083}</style></head><body><h1>Conversation Replay</h1><p id="meta" class="muted">Loading…</p><button id="prev">← Previous</button><button id="next">Next →</button><input id="slider" type="range" min="0" value="0"><h2 id="title"></h2><pre id="content"></pre><h3>Tool calls</h3><pre id="tools">None</pre><h3>State snapshot in effect</h3><pre id="state">None</pre><script>fetch('/api/conversation').then(r=>r.json()).then(c=>{let ts=c.turns,s=document.querySelector('#slider');document.querySelector('#meta').textContent=c.conversation_id+' · '+ts.length+' turns';s.max=Math.max(0,ts.length-1);function show(){let n=+s.value,t=ts[n],state=null;for(let i=0;i<=n;i++)if(ts[i].state!=null)state=ts[i].state;document.querySelector('#title').textContent='Turn '+t.index+' · '+t.role;document.querySelector('#content').textContent=t.content;document.querySelector('#tools').textContent=t.tool_calls&&t.tool_calls.length?JSON.stringify(t.tool_calls,null,2):'None';document.querySelector('#state').textContent=state?JSON.stringify(state,null,2):'None'}s.oninput=show;document.querySelector('#prev').onclick=()=>{s.value=Math.max(0,+s.value-1);show()};document.querySelector('#next').onclick=()=>{s.value=Math.min(+s.max,+s.value+1);show()};show()})</script></body></html>`
