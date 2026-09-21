// Package apitypes holds the wire contract shared by server, CLI, and MCP.
package apitypes

// DecideRequest is the native Laya decide call.
// Features are structured signals; Laya never generates free text.
type DecideRequest struct {
	Model      string             `json:"model,omitempty"`
	Features   map[string]float64 `json:"features,omitempty"`
	Candidates []string           `json:"candidates,omitempty"`
	TopK       int                `json:"top_k,omitempty"`
	// JEVCompat keeps legacy clients working without rewriting payloads.
	JEVCompat bool `json:"jev_compat,omitempty"`
}

// Candidate is one structured pattern prediction.
type Candidate struct {
	ID          string             `json:"id"`
	Probability float64            `json:"probability"`
	Score       float64            `json:"score"`
	Features    map[string]float64 `json:"features,omitempty"`
}

// DecideResponse is returned by /v1/decide and /v1/jev/decide.
type DecideResponse struct {
	Model       string      `json:"model"`
	Engine      string      `json:"engine"`
	Top         Candidate   `json:"top"`
	Candidates  []Candidate `json:"candidates"`
	LatencyUS   int64       `json:"latency_us"`
	CompatLayer string      `json:"compat_layer,omitempty"`
}

// ModelInfo describes one member of the horizontal System 1 family.
type ModelInfo struct {
	ID           string   `json:"id"`
	Family       string   `json:"family"`
	Description  string   `json:"description"`
	Patterns     []string `json:"patterns"`
	MaxLatencyMs int      `json:"max_latency_ms"`
}

// ModelsResponse lists available models.
type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// HealthResponse reports service status.
type HealthResponse struct {
	Status  string `json:"status"`
	Engine  string `json:"engine"`
	Version string `json:"version"`
	Models  int    `json:"models"`
}

// ErrorResponse is the JSON error body.
type ErrorResponse struct {
	Error string `json:"error"`
}
