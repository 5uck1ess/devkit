package runners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ProbeStatus is the high-level outcome of a /v1/models probe. It is the
// source of truth for pass/fail decisions; HTTPStatus stays on ProbeResult as
// diagnostic data.
type ProbeStatus string

const (
	ProbeDisabled        ProbeStatus = "disabled"
	ProbeUnreachable     ProbeStatus = "unreachable"
	ProbeEndpointError   ProbeStatus = "endpoint_error"
	ProbeInvalidResponse ProbeStatus = "invalid_response"
	ProbeModelMissing    ProbeStatus = "model_missing"
	ProbeHealthy         ProbeStatus = "healthy"
)

type ProbeConfig struct {
	Enabled  bool
	Endpoint string
	Model    string
	APIKey   string
	Timeout  time.Duration
}

type ProbeResult struct {
	Endpoint   string      `json:"endpoint"`
	Model      string      `json:"model"`
	Enabled    bool        `json:"enabled"`
	Status     ProbeStatus `json:"status"`
	HTTPStatus int         `json:"http_status"`
	LatencyMS  int64       `json:"latency_ms"`
	ModelsSeen []string    `json:"models_seen"`
	ErrorMsg   string      `json:"error,omitempty"`
	Hint       string      `json:"hint,omitempty"`
}

// Probe performs an OpenAI-compatible /v1/models health check and reports the
// outcome as a ProbeStatus. It is used by both the `devkit-engine probe-local`
// CLI and LocalRunner.Available so the two cannot drift on reachability logic.
func Probe(ctx context.Context, cfg ProbeConfig) ProbeResult {
	r := ProbeResult{
		Endpoint:   cfg.Endpoint,
		Model:      cfg.Model,
		Enabled:    cfg.Enabled,
		Status:     ProbeDisabled,
		ModelsSeen: []string{},
	}
	if !cfg.Enabled {
		return r
	}

	reqCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint+"/models", nil)
	if err != nil {
		r.Status = ProbeUnreachable
		r.ErrorMsg = fmt.Sprintf("building request: %v", err)
		r.Hint = "check DEVKIT_LOCAL_ENDPOINT format — must be a valid URL ending in /v1"
		return r
	}
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: cfg.Timeout}
	start := time.Now()
	resp, err := client.Do(req)
	r.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		r.Status = ProbeUnreachable
		r.ErrorMsg = err.Error()
		r.Hint = unreachableHint(reqCtx, cfg.Timeout, err)
		return r
	}
	defer resp.Body.Close()
	r.HTTPStatus = resp.StatusCode

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		r.Status = ProbeEndpointError
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
		r.Status = ProbeInvalidResponse
		r.ErrorMsg = fmt.Sprintf("reading response: %v", err)
		return r
	}

	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		r.Status = ProbeInvalidResponse
		r.ErrorMsg = fmt.Sprintf("parsing /models response: %v", err)
		r.Hint = "endpoint did not return OpenAI /models JSON — confirm it speaks the OpenAI spec"
		return r
	}

	match := false
	for _, m := range parsed.Data {
		r.ModelsSeen = append(r.ModelsSeen, m.ID)
		if m.ID == cfg.Model {
			match = true
		}
	}
	if match {
		r.Status = ProbeHealthy
		return r
	}
	r.Status = ProbeModelMissing
	r.Hint = fmt.Sprintf("configured model %q not in /models — check DEVKIT_LOCAL_MODEL matches a name the server exposes", cfg.Model)
	return r
}

// unreachableHint returns an actionable hint string for a network-layer
// failure. The context is checked first so we can distinguish caller-cancel
// from probe-timeout; otherwise we fall back to a generic reachability hint.
// unreachableHint returns an actionable hint string for a network-layer
// failure. The `err || ctx.Err()` OR pattern is intentional — either source
// may carry the deadline/cancel signal depending on where the cancel races
// the transport; keep both checks.
func unreachableHint(ctx context.Context, timeout time.Duration, err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Sprintf("endpoint did not respond within %s — check server health or confirm DEVKIT_LOCAL_ENDPOINT points to a running server", timeout)
	}
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return "probe canceled by caller before the endpoint responded"
	}
	return "endpoint unreachable — check it is running and DEVKIT_LOCAL_ENDPOINT is correct"
}
