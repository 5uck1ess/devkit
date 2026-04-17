package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	_ = json.RawMessage{}
}
