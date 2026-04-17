package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, cfg.Endpoint+"/models", nil)
	if err != nil {
		r.ErrorMsg = fmt.Sprintf("building request: %v", err)
		r.Hint = "check DEVKIT_LOCAL_ENDPOINT format — must be a valid URL ending in /v1"
		return r
	}
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	r.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		r.ErrorMsg = err.Error()
		r.Hint = "endpoint unreachable — check it is running and DEVKIT_LOCAL_ENDPOINT is correct"
		return r
	}
	defer resp.Body.Close()
	r.HTTPStatus = resp.StatusCode

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		r.ErrorMsg = strings.TrimSpace(string(body))
		switch resp.StatusCode {
		case 401, 403:
			r.Hint = "check DEVKIT_LOCAL_API_KEY — endpoint requires auth"
		case 404:
			r.Hint = "check DEVKIT_LOCAL_ENDPOINT — must end in /v1 (some stacks need the suffix)"
		default:
			r.Hint = "endpoint returned an error — check server logs"
		}
		return r
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		r.ErrorMsg = fmt.Sprintf("reading response: %v", err)
		return r
	}

	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		r.ErrorMsg = fmt.Sprintf("parsing /models response: %v", err)
		r.Hint = "endpoint did not return OpenAI /models JSON — confirm it speaks the OpenAI spec"
		return r
	}

	r.Reachable = true
	for _, m := range parsed.Data {
		r.ModelsSeen = append(r.ModelsSeen, m.ID)
		if m.ID == cfg.Model {
			r.ModelMatch = true
		}
	}
	if !r.ModelMatch {
		r.Hint = fmt.Sprintf("configured model %q not in /models — check DEVKIT_LOCAL_MODEL matches a name the server exposes", cfg.Model)
	}
	return r
}

func formatHuman(r ProbeResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "endpoint:    %s\n", r.Endpoint)
	fmt.Fprintf(&b, "model:       %s\n", r.Model)
	if !r.Enabled {
		fmt.Fprintln(&b, "enabled:     no — disabled (set DEVKIT_LOCAL_ENABLED=1 to enable)")
		return b.String()
	}
	fmt.Fprintln(&b, "enabled:     yes")
	if r.Reachable {
		fmt.Fprintf(&b, "reachable:   yes (HTTP %d, %dms)\n", r.HTTPStatus, r.LatencyMS)
		if len(r.ModelsSeen) > 0 {
			fmt.Fprintf(&b, "models seen: %s\n", strings.Join(r.ModelsSeen, ", "))
		} else {
			fmt.Fprintln(&b, "models seen: (none returned)")
		}
		if r.ModelMatch {
			fmt.Fprintln(&b, "model match: OK (configured model present in /models)")
		} else {
			fmt.Fprintln(&b, "model match: MISSING")
			if r.Hint != "" {
				fmt.Fprintf(&b, "hint:        %s\n", r.Hint)
			}
		}
		return b.String()
	}
	if r.HTTPStatus > 0 {
		fmt.Fprintf(&b, "reachable:   NO (HTTP %d in %dms)\n", r.HTTPStatus, r.LatencyMS)
	} else {
		fmt.Fprintf(&b, "reachable:   NO (%dms)\n", r.LatencyMS)
	}
	if r.Hint != "" {
		fmt.Fprintf(&b, "hint:        %s\n", r.Hint)
	}
	if r.ErrorMsg != "" {
		fmt.Fprintf(&b, "body:        %s\n", r.ErrorMsg)
	}
	return b.String()
}

func formatJSON(r ProbeResult) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
