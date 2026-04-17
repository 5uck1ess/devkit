package cmd

import (
	"context"
	"time"
)

type ProbeConfig struct {
	Enabled  bool
	Endpoint string
	Model    string
	APIKey   string
	Timeout  time.Duration
}

type ProbeResult struct {
	Endpoint   string   `json:"endpoint"`
	Model      string   `json:"model"`
	Enabled    bool     `json:"enabled"`
	Reachable  bool     `json:"reachable"`
	HTTPStatus int      `json:"http_status"`
	LatencyMS  int64    `json:"latency_ms"`
	ModelsSeen []string `json:"models_seen"`
	ModelMatch bool     `json:"model_match"`
	ErrorMsg   string   `json:"error,omitempty"`
	Hint       string   `json:"hint,omitempty"`
}

func runProbe(ctx context.Context, cfg ProbeConfig) ProbeResult {
	r := ProbeResult{
		Endpoint: cfg.Endpoint,
		Model:    cfg.Model,
		Enabled:  cfg.Enabled,
	}
	if !cfg.Enabled {
		return r
	}
	return r
}
