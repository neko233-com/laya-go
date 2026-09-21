package engine

import (
	"testing"
)

func TestRegistryDefaultAndList(t *testing.T) {
	r := NewRegistry()
	m, err := r.Get("")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != DefaultModelID {
		t.Fatalf("default model = %s, want %s", m.ID, DefaultModelID)
	}
	list := r.List()
	if len(list) < 3 {
		t.Fatalf("want >=3 models, got %d", len(list))
	}
	if r.Count() != len(list) {
		t.Fatalf("count mismatch")
	}
}

func TestDecideNonAutoregressiveTopK(t *testing.T) {
	r := NewRegistry()
	m, err := r.Get("laya-base")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Decide(m, map[string]float64{
		"intent_change": 0.9,
		"target_known":  0.8,
		"has_plan":      0.7,
	}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Candidates) != 3 {
		t.Fatalf("topK candidates = %d, want 3", len(res.Candidates))
	}
	top := res.Top()
	if top.ID != PatToolEdit {
		t.Fatalf("top = %s, want %s", top.ID, PatToolEdit)
	}
	sum := 0.0
	for _, c := range res.Candidates {
		if c.Probability <= 0 {
			t.Fatalf("non-positive probability for %s", c.ID)
		}
		sum += c.Probability
	}
	if sum <= 0.99 || sum > 1.01 {
		// top-3 sum is not required to be 1; just ensure sensible range.
		if sum <= 0 || sum > 1.0001 {
			t.Fatalf("probability sum out of range: %v", sum)
		}
	}
}

func TestDecideFilterAndUnknownModel(t *testing.T) {
	r := NewRegistry()
	m, err := r.Get("laya-mini")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Decide(m, map[string]float64{"goal_done": 1}, []string{PatAgentFinish, PatToolAsk}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if res.Top().ID != PatAgentFinish {
		t.Fatalf("top = %s, want %s", res.Top().ID, PatAgentFinish)
	}
	for _, c := range res.Candidates {
		if c.ID != PatAgentFinish && c.ID != PatToolAsk {
			t.Fatalf("filter leaked %s", c.ID)
		}
	}
	if _, err := r.Get("laya-nope"); err == nil {
		t.Fatal("expected unknown model error")
	}
}

func TestExplain(t *testing.T) {
	r := NewRegistry()
	m, _ := r.Get("laya-base")
	parts, total, err := Explain(m, PatToolEdit, map[string]float64{
		"intent_change": 1,
		"target_known":  1,
		"has_plan":      1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if total <= parts["bias"] {
		t.Fatalf("total %v should exceed bias %v", total, parts["bias"])
	}
	if parts["intent_change"] != 1.5 {
		t.Fatalf("intent_change contribution = %v", parts["intent_change"])
	}
}

func TestGuardPatternWinsOnUnsafe(t *testing.T) {
	r := NewRegistry()
	m, _ := r.Get("laya-pro")
	res, err := Decide(m, map[string]float64{
		"unsafe":    1,
		"risk_high": 1,
	}, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Top().ID != PatGuardBlock {
		t.Fatalf("top = %s, want guard.block", res.Top().ID)
	}
}
