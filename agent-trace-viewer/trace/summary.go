package trace

import "sort"

// SlowSpan is a span selected for the slowest-span report.
type SlowSpan struct {
	Name     string
	Duration int64
}

// Summary aggregates the useful debugging totals for a Trace.
type Summary struct {
	WallDuration int64
	SpanCount    int
	ByKind       map[string]int
	TokensIn     int
	TokensOut    int
	CostUSD      float64
	ErrorCount   int
	Slowest      []SlowSpan
}

// Summarize returns totals and the three longest spans, ordered slowest first.
func Summarize(t Trace) Summary {
	s := Summary{WallDuration: int64(t.Duration()), SpanCount: len(t.Spans), ByKind: make(map[string]int)}
	for _, span := range t.Spans {
		s.ByKind[span.Kind]++
		if span.Tokens != nil {
			s.TokensIn += span.Tokens.In
			s.TokensOut += span.Tokens.Out
		}
		s.CostUSD += span.CostUSD
		if span.Status == "error" {
			s.ErrorCount++
		}
		s.Slowest = append(s.Slowest, SlowSpan{Name: span.Name, Duration: int64(span.Duration())})
	}
	sort.SliceStable(s.Slowest, func(i, j int) bool {
		if s.Slowest[i].Duration == s.Slowest[j].Duration {
			return s.Slowest[i].Name < s.Slowest[j].Name
		}
		return s.Slowest[i].Duration > s.Slowest[j].Duration
	})
	if len(s.Slowest) > 3 {
		s.Slowest = s.Slowest[:3]
	}
	return s
}
