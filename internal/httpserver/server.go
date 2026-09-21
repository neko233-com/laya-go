// Package httpserver exposes the Laya System 1 decision API.
package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/neko233-com/laya-go/internal/apitypes"
	"github.com/neko233-com/laya-go/internal/engine"
	"github.com/neko233-com/laya-go/internal/metrics"
	"github.com/neko233-com/laya-go/internal/version"
)

// Version is the service version string.
var Version = version.Version

// DecisionRecorder receives decide telemetry. Optional.
type DecisionRecorder interface {
	Record(ev metrics.Event)
}

// Server is the Laya HTTP server.
type Server struct {
	registry *engine.Registry
	mux      *http.ServeMux
	recorder DecisionRecorder
}

// New builds a server with routes.
func New(reg *engine.Registry) *Server {
	if reg == nil {
		reg = engine.NewRegistry()
	}
	s := &Server{registry: reg, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/models", s.handleModels)
	s.mux.HandleFunc("POST /v1/decide", s.handleDecide)
	s.mux.HandleFunc("POST /v1/predict", s.handlePredict)
	s.mux.HandleFunc("POST /v1/jev/decide", s.handleJEVDecide)
	s.mux.HandleFunc("POST /v1/explain", s.handleExplain)
	return s
}

// SetRecorder attaches decision telemetry.
func (s *Server) SetRecorder(r DecisionRecorder) { s.recorder = r }

// Handler returns the root handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apitypes.HealthResponse{
		Status:  "ok",
		Engine:  engine.EngineName,
		Version: Version,
		Models:  s.registry.Count(),
	})
}

func (s *Server) handleModels(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, apitypes.ModelsResponse{Models: s.registry.List()})
}

func (s *Server) handleDecide(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, false)
}

func (s *Server) handleJEVDecide(w http.ResponseWriter, r *http.Request) {
	s.decide(w, r, true)
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request, jev bool) {
	var req apitypes.DecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if jev {
		req.JEVCompat = true
	}
	features := normalizeFeatures(req.Features)
	m, err := s.registry.Get(req.Model)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := engine.Decide(m, features, req.Candidates, req.TopK)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out := apitypes.DecideResponse{
		Model:      res.Model,
		Engine:     engine.EngineName,
		Top:        res.Top(),
		Candidates: res.Candidates,
		LatencyUS:  res.LatencyUS,
	}
	if jev {
		out.CompatLayer = "jev"
		out.Model = res.Model
	}
	if s.recorder != nil {
		top := res.Top()
		path := "/v1/decide"
		if jev {
			path = "/v1/jev/decide"
		}
		s.recorder.Record(metrics.Event{
			At:             time.Now(),
			Model:          res.Model,
			Top:            top.ID,
			Probability:    top.Probability,
			LatencyUS:      res.LatencyUS,
			CandidateCount: len(res.Candidates),
			Compat:         out.CompatLayer,
			Path:           path,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePredict(w http.ResponseWriter, r *http.Request) {
	var req apitypes.DecideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	features := normalizeFeatures(req.Features)
	m, err := s.registry.Get(req.Model)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := engine.Decide(m, features, req.Candidates, len(m.Patterns))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, apitypes.DecideResponse{
		Model:      res.Model,
		Engine:     engine.EngineName,
		Top:        res.Top(),
		Candidates: res.Candidates,
		LatencyUS:  res.LatencyUS,
	})
}

type explainRequest struct {
	Model     string             `json:"model,omitempty"`
	PatternID string             `json:"pattern_id"`
	Features  map[string]float64 `json:"features,omitempty"`
}

type explainResponse struct {
	Model         string             `json:"model"`
	PatternID     string             `json:"pattern_id"`
	Contributions map[string]float64 `json:"contributions"`
	TotalScore    float64            `json:"total_score"`
	LatencyUS     int64              `json:"latency_us"`
}

func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	var req explainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.PatternID == "" {
		writeErr(w, http.StatusBadRequest, "pattern_id is required")
		return
	}
	start := time.Now()
	m, err := s.registry.Get(req.Model)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	parts, total, err := engine.Explain(m, req.PatternID, normalizeFeatures(req.Features))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, explainResponse{
		Model:         m.ID,
		PatternID:     req.PatternID,
		Contributions: parts,
		TotalScore:    total,
		LatencyUS:     time.Since(start).Microseconds(),
	})
}

func normalizeFeatures(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(in))
	for k, v := range in {
		key := engine.NormalizeFeatureKey(k)
		if key == "" {
			continue
		}
		out[key] = v
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, apitypes.ErrorResponse{Error: msg})
}

// AddrLabel normalizes listen address for logs.
func AddrLabel(addr string) string {
	if strings.TrimSpace(addr) == "" {
		return "0.0.0.0:7710"
	}
	return addr
}
