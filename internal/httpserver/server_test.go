package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neko233-com/laya-go/internal/apitypes"
	"github.com/neko233-com/laya-go/internal/engine"
)

func TestHealthModelsDecideJEV(t *testing.T) {
	srv := New(engine.NewRegistry())
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// health
	hr, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer hr.Body.Close()
	var health apitypes.HealthResponse
	if err := json.NewDecoder(hr.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if health.Status != "ok" || health.Models < 3 {
		t.Fatalf("unexpected health: %+v", health)
	}

	// models
	mr, err := http.Get(ts.URL + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Body.Close()
	var models apitypes.ModelsResponse
	if err := json.NewDecoder(mr.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	if len(models.Models) < 3 {
		t.Fatalf("models = %d", len(models.Models))
	}

	body, _ := json.Marshal(apitypes.DecideRequest{
		Features: map[string]float64{"intent_change": 0.9, "target_known": 0.8},
		TopK:     2,
	})
	dr, err := http.Post(ts.URL+"/v1/decide", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer dr.Body.Close()
	var decide apitypes.DecideResponse
	if err := json.NewDecoder(dr.Body).Decode(&decide); err != nil {
		t.Fatal(err)
	}
	if decide.Top.ID != "tool.edit" {
		t.Fatalf("top = %s", decide.Top.ID)
	}

	// JEV compat endpoint keeps working for legacy agent clients.
	jr, err := http.Post(ts.URL+"/v1/jev/decide", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer jr.Body.Close()
	var jev apitypes.DecideResponse
	if err := json.NewDecoder(jr.Body).Decode(&jev); err != nil {
		t.Fatal(err)
	}
	if jev.CompatLayer != "jev" || jev.Top.ID == "" {
		t.Fatalf("jev response = %+v", jev)
	}
}
