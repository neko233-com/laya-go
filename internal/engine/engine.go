// Package engine implements non-autoregressive System 1 structured prediction.
//
// Laya scores every registered pattern from structured features in one pass,
// then normalizes probabilities. There is no token loop and no text generation.
package engine

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/neko233-com/laya-go/internal/apitypes"
)

const EngineName = "laya-system1"

// Pattern is a structured decision candidate.
type Pattern struct {
	ID       string
	Name     string
	Weights  map[string]float64
	Bias     float64
	Priority int
}

// Model is one horizontal System 1 family member.
type Model struct {
	ID           string
	Family       string
	Description  string
	Patterns     []Pattern
	Temperature  float64
	MaxLatencyMs int
}

// Result is a scored prediction set.
type Result struct {
	Model      string
	Candidates []apitypes.Candidate
	LatencyUS  int64
}

// Top returns the highest-probability candidate.
func (res *Result) Top() apitypes.Candidate {
	if len(res.Candidates) == 0 {
		return apitypes.Candidate{ID: "", Probability: 0}
	}
	return res.Candidates[0]
}

// Decide runs non-autoregressive scoring over the model patterns.
func Decide(m *Model, features map[string]float64, candidateFilter []string, topK int) (*Result, error) {
	if m == nil {
		return nil, fmt.Errorf("model is nil")
	}
	start := time.Now()
	if features == nil {
		features = map[string]float64{}
	}
	allow := make(map[string]bool, len(candidateFilter))
	for _, id := range candidateFilter {
		if id != "" {
			allow[id] = true
		}
	}

	scored := make([]apitypes.Candidate, 0, len(m.Patterns))
	for _, p := range m.Patterns {
		if len(allow) > 0 && !allow[p.ID] {
			continue
		}
		score := p.Bias
		for k, w := range p.Weights {
			if v, ok := features[k]; ok {
				score += w * v
			}
		}
		// Small priority bias keeps ties stable without dominating scores.
		score += float64(p.Priority) * 1e-6
		scored = append(scored, apitypes.Candidate{
			ID:       p.ID,
			Score:    score,
			Features: cloneFeatures(features),
		})
	}
	if len(scored) == 0 {
		return nil, fmt.Errorf("no candidates after filter")
	}

	temp := m.Temperature
	if temp <= 0 {
		temp = 0.15
	}
	probs := softmax(scoresOf(scored), temp)
	for i := range scored {
		scored[i].Probability = probs[i]
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Probability == scored[j].Probability {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Probability > scored[j].Probability
	})

	if topK <= 0 {
		topK = 5
	}
	if topK > len(scored) {
		topK = len(scored)
	}

	return &Result{
		Model:      m.ID,
		Candidates: scored[:topK],
		LatencyUS:  time.Since(start).Microseconds(),
	}, nil
}

// Explain returns pattern-level contribution breakdown for debugging.
func Explain(m *Model, patternID string, features map[string]float64) (map[string]float64, float64, error) {
	if m == nil {
		return nil, 0, fmt.Errorf("model is nil")
	}
	var p *Pattern
	for i := range m.Patterns {
		if m.Patterns[i].ID == patternID {
			p = &m.Patterns[i]
			break
		}
	}
	if p == nil {
		return nil, 0, fmt.Errorf("unknown pattern %q", patternID)
	}
	parts := make(map[string]float64, len(p.Weights))
	total := p.Bias
	parts["bias"] = p.Bias
	for k, w := range p.Weights {
		v := features[k]
		if v == 0 {
			continue
		}
		c := w * v
		parts[k] = c
		total += c
	}
	return parts, total, nil
}

func scoresOf(c []apitypes.Candidate) []float64 {
	out := make([]float64, len(c))
	for i := range c {
		out[i] = c[i].Score
	}
	return out
}

func softmax(scores []float64, temperature float64) []float64 {
	maxS := scores[0]
	for _, s := range scores[1:] {
		if s > maxS {
			maxS = s
		}
	}
	out := make([]float64, len(scores))
	sum := 0.0
	for i, s := range scores {
		e := math.Exp((s - maxS) / temperature)
		out[i] = e
		sum += e
	}
	if sum <= 0 {
		uniform := 1.0 / float64(len(scores))
		for i := range out {
			out[i] = uniform
		}
		return out
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

func cloneFeatures(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// NormalizeFeatureKey lowercases and trims feature keys.
func NormalizeFeatureKey(k string) string {
	return strings.ToLower(strings.TrimSpace(k))
}
