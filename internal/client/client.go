// Package client is the shared agent-side Laya HTTP client.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/neko233-com/laya-go/internal/apitypes"
)

// DefaultBaseURL is the local/LAN default for laya-deploy.
const DefaultBaseURL = "http://127.0.0.1:7710"

// Client talks to a Laya server.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client with sane defaults.
func New(baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Health calls GET /health.
func (c *Client) Health() (*apitypes.HealthResponse, error) {
	var out apitypes.HealthResponse
	if err := c.get("/health", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Models calls GET /v1/models.
func (c *Client) Models() (*apitypes.ModelsResponse, error) {
	var out apitypes.ModelsResponse
	if err := c.get("/v1/models", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Decide calls POST /v1/decide.
func (c *Client) Decide(req apitypes.DecideRequest) (*apitypes.DecideResponse, error) {
	var out apitypes.DecideResponse
	if err := c.post("/v1/decide", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// JEVDecide calls the legacy-compatible POST /v1/jev/decide.
func (c *Client) JEVDecide(req apitypes.DecideRequest) (*apitypes.DecideResponse, error) {
	req.JEVCompat = true
	var out apitypes.DecideResponse
	if err := c.post("/v1/jev/decide", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Predict calls POST /v1/predict for full pattern scores.
func (c *Client) Predict(req apitypes.DecideRequest) (*apitypes.DecideResponse, error) {
	var out apitypes.DecideResponse
	if err := c.post("/v1/predict", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) get(path string, out any) error {
	resp, err := c.HTTP.Get(c.BaseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func (c *Client) post(path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Post(c.BaseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func decode(resp *http.Response, out any) error {
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var er apitypes.ErrorResponse
		if json.Unmarshal(raw, &er) == nil && er.Error != "" {
			return fmt.Errorf("laya http %d: %s", resp.StatusCode, er.Error)
		}
		return fmt.Errorf("laya http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, out)
}

// ParseFeatures parses repeated key=value flags into a feature map.
func ParseFeatures(pairs []string) (map[string]float64, error) {
	out := make(map[string]float64, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			return nil, fmt.Errorf("feature %q must be key=value", p)
		}
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			return nil, fmt.Errorf("feature %q has empty key", p)
		}
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%g", &f); err != nil {
			return nil, fmt.Errorf("feature %q: invalid number", p)
		}
		out[k] = f
	}
	return out, nil
}
