// Command laya-mcp exposes Laya System 1 decisions as MCP tools for agents.
//
// This is the agent-side integration that replaces legacy JEV MCP servers:
// agents call laya_decide / laya_models / laya_health instead of jev_* tools.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/neko233-com/laya-go/internal/apitypes"
	"github.com/neko233-com/laya-go/internal/client"
)

const serverName = "laya-mcp"
const serverVersion = "0.1.0"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func main() {
	c := client.New(os.Getenv("LAYA_URL"))
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	dec := json.NewDecoder(in)
	for {
		var req rpcRequest
		if err := dec.Decode(&req); err != nil {
			return
		}
		resp := handle(c, &req)
		if req.ID == nil {
			continue
		}
		b, err := json.Marshal(resp)
		if err != nil {
			continue
		}
		out.Write(b)
		out.Write([]byte("\n"))
		out.Flush()
	}
}

func handle(c *client.Client, req *rpcRequest) rpcResponse {
	base := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		base.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": serverVersion,
			},
		}
		return base
	case "notifications/initialized", "initialized":
		base.Result = map[string]any{}
		return base
	case "ping":
		base.Result = map[string]any{}
		return base
	case "tools/list":
		base.Result = map[string]any{"tools": tools()}
		return base
	case "tools/call":
		return handleToolCall(c, req, base)
	default:
		base.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
		return base
	}
}

func tools() []toolDef {
	return []toolDef{
		{
			Name:        "laya_health",
			Description: "Check Laya System 1 decision server health. Replaces legacy JEV health tools.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "laya_models",
			Description: "List open Laya System 1 decision models (laya-mini/base/pro).",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:        "laya_decide",
			Description: "Fast structured pattern decision. Input features; output ranked pattern IDs with probabilities. Does not generate text. Use instead of jev decide tools.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"model": map[string]any{
						"type":        "string",
						"description": "Model id: laya-mini | laya-base | laya-pro",
					},
					"features": map[string]any{
						"type":        "object",
						"description": "Structured feature map, e.g. intent_change=0.9",
						"additionalProperties": map[string]any{
							"type": "number",
						},
					},
					"candidates": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Optional pattern id allow-list",
					},
					"top_k": map[string]any{
						"type":        "integer",
						"description": "Number of candidates to return",
					},
					"jev_compat": map[string]any{
						"type":        "boolean",
						"description": "Use JEV-compatible decide path",
					},
				},
			},
		},
	}
}

func handleToolCall(c *client.Client, req *rpcRequest, base rpcResponse) rpcResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			base.Error = &rpcError{Code: -32602, Message: "invalid params"}
			return base
		}
	}
	switch params.Name {
	case "laya_health":
		h, err := c.Health()
		if err != nil {
			return toolError(base, err.Error())
		}
		return toolJSON(base, h)
	case "laya_models":
		m, err := c.Models()
		if err != nil {
			return toolError(base, err.Error())
		}
		return toolJSON(base, m)
	case "laya_decide":
		var dreq apitypes.DecideRequest
		if len(params.Arguments) > 0 {
			if err := json.Unmarshal(params.Arguments, &dreq); err != nil {
				return toolError(base, "invalid arguments: "+err.Error())
			}
		}
		var resp *apitypes.DecideResponse
		var err error
		if dreq.JEVCompat {
			resp, err = c.JEVDecide(dreq)
		} else {
			resp, err = c.Decide(dreq)
		}
		if err != nil {
			return toolError(base, err.Error())
		}
		return toolJSON(base, resp)
	default:
		base.Error = &rpcError{Code: -32602, Message: "unknown tool: " + params.Name}
		return base
	}
}

func toolJSON(base rpcResponse, v any) rpcResponse {
	b, err := json.Marshal(v)
	if err != nil {
		return toolError(base, err.Error())
	}
	base.Result = map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(b)},
		},
		"isError": false,
	}
	return base
}

func toolError(base rpcResponse, msg string) rpcResponse {
	base.Result = map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": strings.TrimSpace(msg)},
		},
		"isError": true,
	}
	return base
}

var _ = fmt.Sprintf
