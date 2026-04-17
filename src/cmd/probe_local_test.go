package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/5uck1ess/devkit/runners"
)

func TestFormatHuman_HappyPath(t *testing.T) {
	r := runners.ProbeResult{
		Endpoint:   "http://host:8080/v1",
		Model:      "gemma",
		Enabled:    true,
		Status:     runners.ProbeHealthy,
		HTTPStatus: 200,
		LatencyMS:  123,
		ModelsSeen: []string{"gemma", "mistral"},
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
	out := formatHuman(runners.ProbeResult{Endpoint: "http://x/v1", Model: "y", Enabled: false})
	if !strings.Contains(out, "disabled") {
		t.Errorf("disabled output missing 'disabled' marker. Got:\n%s", out)
	}
}

func TestFormatHuman_Unreachable(t *testing.T) {
	r := runners.ProbeResult{
		Endpoint: "http://x/v1", Model: "y", Enabled: true,
		Status: runners.ProbeEndpointError, HTTPStatus: 401,
		ErrorMsg: `{"error":"nope"}`,
		Hint:     "check DEVKIT_LOCAL_API_KEY — endpoint requires auth",
	}
	out := formatHuman(r)

	for _, want := range []string{"reachable:   NO", "HTTP 401", "hint:", "DEVKIT_LOCAL_API_KEY", "body:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q. Got:\n%s", want, out)
		}
	}
}

func TestFormatHuman_ModelMissing(t *testing.T) {
	r := runners.ProbeResult{
		Endpoint: "http://x/v1", Model: "y", Enabled: true,
		Status: runners.ProbeModelMissing, HTTPStatus: 200, LatencyMS: 42,
		ModelsSeen: []string{"z"},
		Hint:       "configured model \"y\" not in /models",
	}
	out := formatHuman(r)

	for _, want := range []string{"reachable:   yes", "HTTP 200", "models seen: z", "model match: MISSING", "hint:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q. Got:\n%s", want, out)
		}
	}
}

func TestFormatJSON(t *testing.T) {
	r := runners.ProbeResult{
		Endpoint: "http://x/v1", Model: "m", Enabled: true,
		Status: runners.ProbeHealthy, HTTPStatus: 200, LatencyMS: 42,
		ModelsSeen: []string{"m", "other"},
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
	if parsed["status"] != "healthy" {
		t.Errorf("status: got %v, want healthy", parsed["status"])
	}
	if parsed["http_status"].(float64) != 200 {
		t.Errorf("http_status: got %v, want 200", parsed["http_status"])
	}
}

func TestFormatJSON_ModelsSeenEmptyNotNull(t *testing.T) {
	r := runners.ProbeResult{
		Endpoint: "http://x/v1", Model: "m", Enabled: true,
		Status: runners.ProbeUnreachable, ModelsSeen: []string{},
	}
	out, err := formatJSON(r)
	if err != nil {
		t.Fatalf("formatJSON err: %v", err)
	}
	if !strings.Contains(string(out), `"models_seen": []`) {
		t.Errorf("want models_seen: [] (not null). Got:\n%s", out)
	}
}
