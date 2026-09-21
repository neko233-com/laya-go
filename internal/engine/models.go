package engine

// DefaultModelID is used when the client omits model.
const DefaultModelID = "laya-base"

// Family name shared by all built-in models.
const Family = "laya-system1"

// BuiltInModels returns the open horizontal System 1 family.
//
// These models replace the legacy JEV decision service: same agent-side job
// (fast structured pattern prediction), open weights-as-rules, no text gen.
func BuiltInModels() []*Model {
	return []*Model{
		miniModel(),
		baseModel(),
		proModel(),
	}
}

// Agent routing pattern IDs used across the family.
// Agents that previously called JEV should target these IDs.
const (
	PatToolSearch    = "tool.search"
	PatToolRead      = "tool.read"
	PatToolEdit      = "tool.edit"
	PatToolRun       = "tool.run"
	PatToolAsk       = "tool.ask"
	PatAgentPlan     = "agent.plan"
	PatAgentDelegate = "agent.delegate"
	PatAgentFinish   = "agent.finish"
	PatAgentRetry    = "agent.retry"
	PatGuardBlock    = "guard.block"
)

func miniModel() *Model {
	return &Model{
		ID:           "laya-mini",
		Family:       Family,
		Description:  "Lowest-latency System 1 router for high-frequency agent steps.",
		Temperature:  0.1,
		MaxLatencyMs: 1,
		Patterns: []Pattern{
			{ID: PatToolRead, Name: "read source", Bias: 0.2, Priority: 1, Weights: map[string]float64{
				"needs_context": 1.2, "uncertainty": 0.6,
			}},
			{ID: PatToolEdit, Name: "edit code", Bias: 0.1, Priority: 2, Weights: map[string]float64{
				"intent_change": 1.4, "has_plan": 0.4,
			}},
			{ID: PatToolRun, Name: "run command", Bias: 0.0, Priority: 3, Weights: map[string]float64{
				"needs_verify": 1.3, "intent_change": 0.3,
			}},
			{ID: PatToolAsk, Name: "ask user", Bias: -0.2, Priority: 4, Weights: map[string]float64{
				"blocked": 1.5, "ambiguity": 1.0,
			}},
			{ID: PatAgentFinish, Name: "finish", Bias: -0.1, Priority: 5, Weights: map[string]float64{
				"goal_done": 1.6,
			}},
		},
	}
}

func baseModel() *Model {
	return &Model{
		ID:           "laya-base",
		Family:       Family,
		Description:  "Default open System 1 decision model for agent tool routing.",
		Temperature:  0.15,
		MaxLatencyMs: 2,
		Patterns: []Pattern{
			{ID: PatToolSearch, Name: "search codebase", Bias: 0.15, Priority: 1, Weights: map[string]float64{
				"uncertainty": 1.3, "needs_context": 0.8, "scope_unknown": 1.0,
			}},
			{ID: PatToolRead, Name: "read file", Bias: 0.2, Priority: 2, Weights: map[string]float64{
				"needs_context": 1.2, "target_known": 0.7,
			}},
			{ID: PatToolEdit, Name: "edit file", Bias: 0.05, Priority: 3, Weights: map[string]float64{
				"intent_change": 1.5, "target_known": 0.5, "has_plan": 0.4,
			}},
			{ID: PatToolRun, Name: "run command", Bias: 0.0, Priority: 4, Weights: map[string]float64{
				"needs_verify": 1.4, "intent_change": 0.2, "has_plan": 0.3,
			}},
			{ID: PatToolAsk, Name: "ask user", Bias: -0.25, Priority: 5, Weights: map[string]float64{
				"blocked": 1.6, "ambiguity": 1.2, "risk_high": 0.8,
			}},
			{ID: PatAgentPlan, Name: "plan first", Bias: 0.0, Priority: 6, Weights: map[string]float64{
				"complexity": 1.3, "ambiguity": 0.7, "scope_unknown": 0.6,
			}},
			{ID: PatAgentDelegate, Name: "delegate", Bias: -0.15, Priority: 7, Weights: map[string]float64{
				"complexity": 0.9, "parallelizable": 1.2,
			}},
			{ID: PatAgentRetry, Name: "retry / recover", Bias: -0.3, Priority: 8, Weights: map[string]float64{
				"has_error": 1.5, "needs_verify": 0.4,
			}},
			{ID: PatAgentFinish, Name: "finish", Bias: -0.2, Priority: 9, Weights: map[string]float64{
				"goal_done": 1.8, "needs_verify": -0.3,
			}},
			{ID: PatGuardBlock, Name: "guard: block unsafe", Bias: -0.5, Priority: 10, Weights: map[string]float64{
				"risk_high": 1.8, "unsafe": 2.0,
			}},
		},
	}
}

func proModel() *Model {
	m := baseModel()
	m.ID = "laya-pro"
	m.Description = "Wider System 1 pattern set for complex multi-step agent decisions."
	m.MaxLatencyMs = 5
	m.Temperature = 0.2
	m.Patterns = append(m.Patterns,
		Pattern{ID: "agent.research", Name: "deep research", Bias: -0.1, Priority: 11, Weights: map[string]float64{
			"scope_unknown": 1.1, "complexity": 0.8, "external_info": 1.4,
		}},
		Pattern{ID: "agent.reflect", Name: "reflect / review", Bias: -0.2, Priority: 12, Weights: map[string]float64{
			"has_error": 0.9, "risk_high": 0.7, "complexity": 0.5,
		}},
		Pattern{ID: "tool.batch", Name: "batch operations", Bias: -0.25, Priority: 13, Weights: map[string]float64{
			"parallelizable": 1.0, "repetitive": 1.3,
		}},
	)
	return m
}
