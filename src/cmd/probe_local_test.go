package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunProbe_Disabled(t *testing.T) {
	cfg := ProbeConfig{
		Enabled:  false,
		Endpoint: "http://localhost:8080/v1",
		Model:    "test-model",
		Timeout:  1 * time.Second,
	}
	got := runProbe(context.Background(), cfg)

	if got.Enabled {
		t.Errorf("Enabled: got true, want false")
	}
	if got.Reachable {
		t.Errorf("Reachable: got true, want false (disabled should not probe)")
	}
	if got.HTTPStatus != 0 {
		t.Errorf("HTTPStatus: got %d, want 0 (no request should fire)", got.HTTPStatus)
	}
}

func TestRunProbe_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"gemma-4-26b-a4b"},{"id":"mistral-small-4"}]}`)
	}))
	defer srv.Close()

	cfg := ProbeConfig{
		Enabled:  true,
		Endpoint: srv.URL + "/v1",
		Model:    "gemma-4-26b-a4b",
		Timeout:  2 * time.Second,
	}
	got := runProbe(context.Background(), cfg)

	if !got.Reachable {
		t.Fatalf("Reachable: got false, want true (err=%q)", got.ErrorMsg)
	}
	if got.HTTPStatus != 200 {
		t.Errorf("HTTPStatus: got %d, want 200", got.HTTPStatus)
	}
	if !got.ModelMatch {
		t.Errorf("ModelMatch: got false, want true (models=%v)", got.ModelsSeen)
	}
	wantModels := []string{"gemma-4-26b-a4b", "mistral-small-4"}
	if len(got.ModelsSeen) != len(wantModels) {
		t.Fatalf("ModelsSeen len: got %d, want %d", len(got.ModelsSeen), len(wantModels))
	}
	for i, m := range wantModels {
		if got.ModelsSeen[i] != m {
			t.Errorf("ModelsSeen[%d]: got %q, want %q", i, got.ModelsSeen[i], m)
		}
	}
}

func TestRunProbe_ModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"something-else"}]}`)
	}))
	defer srv.Close()

	got := runProbe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "not-there", Timeout: 2 * time.Second,
	})

	if !got.Reachable {
		t.Fatalf("Reachable: got false, want true")
	}
	if got.ModelMatch {
		t.Errorf("ModelMatch: got true, want false")
	}
	if got.Hint == "" {
		t.Errorf("Hint: got empty, want actionable text about DEVKIT_LOCAL_MODEL")
	}
}

func TestRunProbe_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer srv.Close()

	got := runProbe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Reachable {
		t.Errorf("Reachable: got true, want false on 401")
	}
	if got.HTTPStatus != 401 {
		t.Errorf("HTTPStatus: got %d, want 401", got.HTTPStatus)
	}
	if !strings.Contains(got.Hint, "DEVKIT_LOCAL_API_KEY") {
		t.Errorf("Hint: got %q, want mention of DEVKIT_LOCAL_API_KEY", got.Hint)
	}
}

func TestRunProbe_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	got := runProbe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/wrong", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Reachable {
		t.Errorf("Reachable: got true, want false on 404")
	}
	if got.HTTPStatus != 404 {
		t.Errorf("HTTPStatus: got %d, want 404", got.HTTPStatus)
	}
	if !strings.Contains(got.Hint, "/v1") {
		t.Errorf("Hint: got %q, want mention of /v1 suffix", got.Hint)
	}
}

func TestRunProbe_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	got := runProbe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 50 * time.Millisecond,
	})

	if got.Reachable {
		t.Errorf("Reachable: got true, want false on timeout")
	}
	if got.ErrorMsg == "" {
		t.Errorf("ErrorMsg: got empty, want timeout error text")
	}
}

func TestRunProbe_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json at all`)
	}))
	defer srv.Close()

	got := runProbe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Reachable {
		t.Errorf("Reachable: got true, want false on invalid JSON")
	}
	if !strings.Contains(got.ErrorMsg, "parsing") {
		t.Errorf("ErrorMsg: got %q, want mention of parse error", got.ErrorMsg)
	}
}

func TestFormatHuman_HappyPath(t *testing.T) {
	r := ProbeResult{
		Endpoint:   "http://host:8080/v1",
		Model:      "gemma",
		Enabled:    true,
		Reachable:  true,
		HTTPStatus: 200,
		LatencyMS:  123,
		ModelsSeen: []string{"gemma", "mistral"},
		ModelMatch: true,
	}
	out := formatHuman(r)

	for _, want := range []string{
		"endpoint:    http://host:8080/v1",
		"model:       gemma",
		"enabled:     yes",
		"reachable:   yes (HTTP 200, 123ms)",
		"models seen: gemma, mistral",
		"model match: OK",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q. Got:\n%s", want, out)
		}
	}
}

func TestFormatHuman_Disabled(t *testing.T) {
	out := formatHuman(ProbeResult{Endpoint: "http://x/v1", Model: "y", Enabled: false})
	if !strings.Contains(out, "disabled") {
		t.Errorf("disabled output missing 'disabled' marker. Got:\n%s", out)
	}
}

func TestFormatHuman_Unreachable(t *testing.T) {
	r := ProbeResult{
		Endpoint: "http://x/v1", Model: "y", Enabled: true,
		HTTPStatus: 401, ErrorMsg: `{"error":"nope"}`,
		Hint: "check DEVKIT_LOCAL_API_KEY — endpoint requires auth",
	}
	out := formatHuman(r)

	for _, want := range []string{"reachable:   NO", "HTTP 401", "hint:", "DEVKIT_LOCAL_API_KEY", "body:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q. Got:\n%s", want, out)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	r := ProbeResult{
		Endpoint: "http://x/v1", Model: "m", Enabled: true,
		Reachable: true, HTTPStatus: 200, LatencyMS: 42,
		ModelsSeen: []string{"m", "other"}, ModelMatch: true,
	}
	out, err := formatJSON(r)
	if err != nil {
		t.Fatalf("formatJSON err: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	if parsed["endpoint"] != "http://x/v1" {
		t.Errorf("endpoint: got %v, want http://x/v1", parsed["endpoint"])
	}
	if parsed["model_match"] != true {
		t.Errorf("model_match: got %v, want true", parsed["model_match"])
	}
	if parsed["http_status"].(float64) != 200 {
		t.Errorf("http_status: got %v, want 200", parsed["http_status"])
	}
}
