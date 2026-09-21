package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Registry holds loaded models. Safe for hot reload.
type Registry struct {
	mu     sync.RWMutex
	models map[string]*Model
	source string
}

// NewRegistry returns a registry with the built-in Laya family.
func NewRegistry() *Registry {
	r := &Registry{models: make(map[string]*Model)}
	for _, m := range BuiltInModels() {
		cp := *m
		r.models[m.ID] = &cp
	}
	return r
}

// Get returns a model by id, or default when id is empty.
func (r *Registry) Get(id string) (*Model, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if id == "" {
		id = DefaultModelID
	}
	m, ok := r.models[id]
	if !ok {
		return nil, fmt.Errorf("unknown model %q", id)
	}
	// return a shallow copy pointer snapshot for consistent scoring
	cp := *m
	return &cp, nil
}

// List returns model metadata sorted by id.
func (r *Registry) List() []apitypesModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]apitypesModelInfo, 0, len(r.models))
	for _, m := range r.models {
		ids := make([]string, 0, len(m.Patterns))
		for _, p := range m.Patterns {
			ids = append(ids, p.ID)
		}
		sortStrings(ids)
		out = append(out, apitypesModelInfo{
			ID:           m.ID,
			Family:       m.Family,
			Description:  m.Description,
			Patterns:     ids,
			MaxLatencyMs: m.MaxLatencyMs,
		})
	}
	sortModelInfo(out)
	return out
}

// Count returns the number of registered models.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.models)
}

// Source returns the last overlay file path, if any.
func (r *Registry) Source() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.source
}

// modelOverlayFile is the on-disk hot-reload schema.
type modelOverlayFile struct {
	Models []modelOverlay `json:"models"`
}

type modelOverlay struct {
	ID           string           `json:"id"`
	Family       string           `json:"family,omitempty"`
	Description  string           `json:"description,omitempty"`
	Temperature  float64          `json:"temperature,omitempty"`
	MaxLatencyMs int              `json:"max_latency_ms,omitempty"`
	Patterns     []patternOverlay `json:"patterns,omitempty"`
	// ReplacePatterns when true drops built-in patterns for this model id.
	ReplacePatterns bool `json:"replace_patterns,omitempty"`
}

type patternOverlay struct {
	ID       string             `json:"id"`
	Name     string             `json:"name,omitempty"`
	Bias     float64            `json:"bias,omitempty"`
	Priority int                `json:"priority,omitempty"`
	Weights  map[string]float64 `json:"weights,omitempty"`
}

// ReloadFromJSON resets to built-ins then applies overlay file.
// Missing file is not an error: it clears overlay and restores built-ins.
func (r *Registry) ReloadFromJSON(path string) error {
	base := make(map[string]*Model)
	for _, m := range BuiltInModels() {
		cp := *m
		base[m.ID] = &cp
	}

	applied := ""
	if path != "" {
		raw, err := os.ReadFile(path)
		if err == nil {
			var ov modelOverlayFile
			if err := json.Unmarshal(raw, &ov); err != nil {
				return fmt.Errorf("parse models overlay: %w", err)
			}
			for _, mo := range ov.Models {
				if mo.ID == "" {
					return fmt.Errorf("models overlay entry missing id")
				}
				cur, ok := base[mo.ID]
				if !ok {
					cur = &Model{
						ID:          mo.ID,
						Family:      Family,
						Description: mo.Description,
					}
					if mo.Family != "" {
						cur.Family = mo.Family
					}
					base[mo.ID] = cur
				}
				if mo.Description != "" {
					cur.Description = mo.Description
				}
				if mo.Family != "" {
					cur.Family = mo.Family
				}
				if mo.Temperature > 0 {
					cur.Temperature = mo.Temperature
				}
				if mo.MaxLatencyMs > 0 {
					cur.MaxLatencyMs = mo.MaxLatencyMs
				}
				if mo.ReplacePatterns {
					cur.Patterns = nil
				}
				for _, po := range mo.Patterns {
					if po.ID == "" {
						continue
					}
					replaced := false
					for i := range cur.Patterns {
						if cur.Patterns[i].ID == po.ID {
							if po.Name != "" {
								cur.Patterns[i].Name = po.Name
							}
							cur.Patterns[i].Bias = po.Bias
							if po.Priority != 0 {
								cur.Patterns[i].Priority = po.Priority
							}
							if len(po.Weights) > 0 {
								if cur.Patterns[i].Weights == nil {
									cur.Patterns[i].Weights = map[string]float64{}
								}
								for k, v := range po.Weights {
									cur.Patterns[i].Weights[k] = v
								}
							}
							replaced = true
							break
						}
					}
					if !replaced {
						weights := make(map[string]float64, len(po.Weights))
						for k, v := range po.Weights {
							weights[k] = v
						}
						cur.Patterns = append(cur.Patterns, Pattern{
							ID:       po.ID,
							Name:     po.Name,
							Bias:     po.Bias,
							Priority: po.Priority,
							Weights:  weights,
						})
					}
				}
			}
			applied = path
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	r.mu.Lock()
	r.models = base
	r.source = applied
	r.mu.Unlock()
	return nil
}

// SnapshotIDs returns model ids for ops UI.
func (r *Registry) SnapshotIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.models))
	for id := range r.models {
		out = append(out, id)
	}
	sortStrings(out)
	return out
}
