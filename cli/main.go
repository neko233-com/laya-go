// Command laya is the agent-side CLI for the Laya System 1 decision service.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/neko233-com/laya-go/internal/apitypes"
	"github.com/neko233-com/laya-go/internal/client"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "help", "-h", "--help":
		printUsage()
		return 0
	case "health":
		return cmdHealth(rest)
	case "models":
		return cmdModels(rest)
	case "decide":
		return cmdDecide(rest, false)
	case "jev-decide":
		return cmdDecide(rest, true)
	case "predict":
		return cmdPredict(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		printUsage()
		return 2
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `laya — agent-side CLI for Laya System 1 decisions (JEV replacement)

Usage:
  laya health [--url URL]
  laya models [--url URL] [--json]
  laya decide [--url URL] [--model ID] [--top-k N] [--jev] [--json]
              [--candidate id]... [--feature key=value]...
  laya jev-decide  # alias of decide --jev
  laya predict  [--url URL] [--model ID] [--json] [--feature key=value]...

Env:
  LAYA_URL   default http://127.0.0.1:7710
`)
}

type commonFlags struct {
	url      string
	model    string
	topK     int
	jev      bool
	jsonOut  bool
	features []string
	cands    []string
}

func parseFlags(args []string) (*commonFlags, []string, error) {
	f := &commonFlags{url: os.Getenv("LAYA_URL")}
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		next := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch a {
		case "--url":
			v, ok := next()
			if !ok {
				return nil, nil, fmt.Errorf("--url needs a value")
			}
			f.url = v
		case "--model":
			v, ok := next()
			if !ok {
				return nil, nil, fmt.Errorf("--model needs a value")
			}
			f.model = v
		case "--top-k":
			v, ok := next()
			if !ok {
				return nil, nil, fmt.Errorf("--top-k needs a value")
			}
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
				return nil, nil, fmt.Errorf("--top-k must be an integer")
			}
			f.topK = n
		case "--feature":
			v, ok := next()
			if !ok {
				return nil, nil, fmt.Errorf("--feature needs key=value")
			}
			f.features = append(f.features, v)
		case "--candidate":
			v, ok := next()
			if !ok {
				return nil, nil, fmt.Errorf("--candidate needs an id")
			}
			f.cands = append(f.cands, v)
		case "--jev":
			f.jev = true
		case "--json":
			f.jsonOut = true
		case "-h", "--help":
			return nil, nil, fmt.Errorf("help")
		default:
			if strings.HasPrefix(a, "-") {
				return nil, nil, fmt.Errorf("unknown flag %q", a)
			}
			positional = append(positional, a)
		}
	}
	return f, positional, nil
}

func newClient(flags *commonFlags) *client.Client {
	return client.New(flags.url)
}

func cmdHealth(args []string) int {
	f, _, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	h, err := newClient(f).Health()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if f.jsonOut {
		return printJSON(h)
	}
	fmt.Printf("status=%s engine=%s version=%s models=%d\n", h.Status, h.Engine, h.Version, h.Models)
	return 0
}

func cmdModels(args []string) int {
	f, _, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	m, err := newClient(f).Models()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if f.jsonOut {
		return printJSON(m)
	}
	for _, info := range m.Models {
		fmt.Printf("%s\tlatency_budget_ms=%d\tpatterns=%d\t%s\n", info.ID, info.MaxLatencyMs, len(info.Patterns), info.Description)
	}
	return 0
}

func buildDecideReq(f *commonFlags) (apitypes.DecideRequest, error) {
	feats, err := client.ParseFeatures(f.features)
	if err != nil {
		return apitypes.DecideRequest{}, err
	}
	return apitypes.DecideRequest{
		Model:      f.model,
		Features:   feats,
		Candidates: f.cands,
		TopK:       f.topK,
		JEVCompat:  f.jev,
	}, nil
}

func cmdDecide(args []string, forceJEV bool) int {
	f, _, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if forceJEV {
		f.jev = true
	}
	req, err := buildDecideReq(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	c := newClient(f)
	var resp *apitypes.DecideResponse
	if f.jev {
		resp, err = c.JEVDecide(req)
	} else {
		resp, err = c.Decide(req)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if f.jsonOut {
		return printJSON(resp)
	}
	fmt.Printf("model=%s top=%s p=%.4f latency_us=%d", resp.Model, resp.Top.ID, resp.Top.Probability, resp.LatencyUS)
	if resp.CompatLayer != "" {
		fmt.Printf(" compat=%s", resp.CompatLayer)
	}
	fmt.Println()
	for i, cand := range resp.Candidates {
		fmt.Printf("  %d. %s\tp=%.4f\tscore=%.4f\n", i+1, cand.ID, cand.Probability, cand.Score)
	}
	return 0
}

func cmdPredict(args []string) int {
	f, _, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	req, err := buildDecideReq(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	resp, err := newClient(f).Predict(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if f.jsonOut {
		return printJSON(resp)
	}
	for _, cand := range resp.Candidates {
		fmt.Printf("%s\tp=%.4f\tscore=%.4f\n", cand.ID, cand.Probability, cand.Score)
	}
	return 0
}

func printJSON(v any) int {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
