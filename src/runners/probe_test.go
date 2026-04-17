package runners

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProbe_Disabled(t *testing.T) {
	cfg := ProbeConfig{
		Enabled:  false,
		Endpoint: "http://localhost:8080/v1",
		Model:    "test-model",
		Timeout:  1 * time.Second,
	}
	got := Probe(context.Background(), cfg)

	if got.Status != ProbeDisabled {
		t.Errorf("Status: got %q, want %q", got.Status, ProbeDisabled)
	}
	if got.HTTPStatus != 0 {
		t.Errorf("HTTPStatus: got %d, want 0 (no request should fire)", got.HTTPStatus)
	}
	if got.ModelsSeen == nil {
		t.Errorf("ModelsSeen: got nil, want [] (disabled should still init slice)")
	}
}

func TestProbe_Healthy(t *testing.T) {
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
	got := Probe(context.Background(), cfg)

	if got.Status != ProbeHealthy {
		t.Fatalf("Status: got %q, want %q (err=%q)", got.Status, ProbeHealthy, got.ErrorMsg)
	}
	if got.HTTPStatus != 200 {
		t.Errorf("HTTPStatus: got %d, want 200", got.HTTPStatus)
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

func TestProbe_ModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[{"id":"something-else"}]}`)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "not-there", Timeout: 2 * time.Second,
	})

	if got.Status != ProbeModelMissing {
		t.Fatalf("Status: got %q, want %q", got.Status, ProbeModelMissing)
	}
	if got.Hint == "" {
		t.Errorf("Hint: got empty, want actionable text about DEVKIT_LOCAL_MODEL")
	}
}

func TestProbe_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"unauthorized"}`)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Status != ProbeEndpointError {
		t.Errorf("Status: got %q, want %q on 401", got.Status, ProbeEndpointError)
	}
	if got.HTTPStatus != 401 {
		t.Errorf("HTTPStatus: got %d, want 401", got.HTTPStatus)
	}
	if !strings.Contains(got.Hint, "DEVKIT_LOCAL_API_KEY") {
		t.Errorf("Hint: got %q, want mention of DEVKIT_LOCAL_API_KEY", got.Hint)
	}
}

func TestProbe_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/wrong", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Status != ProbeEndpointError {
		t.Errorf("Status: got %q, want %q on 404", got.Status, ProbeEndpointError)
	}
	if got.HTTPStatus != 404 {
		t.Errorf("HTTPStatus: got %d, want 404", got.HTTPStatus)
	}
	if !strings.Contains(got.Hint, "/v1") {
		t.Errorf("Hint: got %q, want mention of /v1 suffix", got.Hint)
	}
}

func TestProbe_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"boom"}`)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Status != ProbeEndpointError {
		t.Errorf("Status: got %q, want %q on 500", got.Status, ProbeEndpointError)
	}
	if got.HTTPStatus != 500 {
		t.Errorf("HTTPStatus: got %d, want 500", got.HTTPStatus)
	}
	if !strings.Contains(got.Hint, "server logs") {
		t.Errorf("Hint: got %q, want mention of server logs (default 5xx branch)", got.Hint)
	}
}

func TestProbe_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		fmt.Fprint(w, `{"data":[]}`)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 50 * time.Millisecond,
	})

	if got.Status != ProbeUnreachable {
		t.Errorf("Status: got %q, want %q on timeout", got.Status, ProbeUnreachable)
	}
	if !strings.Contains(got.Hint, "did not respond within") {
		t.Errorf("Hint: got %q, want deadline-exceeded wording", got.Hint)
	}
}

func TestProbe_CallerCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	got := Probe(ctx, ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 5 * time.Second,
	})

	if got.Status != ProbeUnreachable {
		t.Errorf("Status: got %q, want %q on cancel", got.Status, ProbeUnreachable)
	}
	if !strings.Contains(got.Hint, "canceled by caller") {
		t.Errorf("Hint: got %q, want caller-canceled wording", got.Hint)
	}
}

func TestProbe_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json at all`)
	}))
	defer srv.Close()

	got := Probe(context.Background(), ProbeConfig{
		Enabled: true, Endpoint: srv.URL + "/v1", Model: "m", Timeout: 2 * time.Second,
	})

	if got.Status != ProbeInvalidResponse {
		t.Errorf("Status: got %q, want %q on invalid JSON", got.Status, ProbeInvalidResponse)
	}
	if !strings.Contains(got.ErrorMsg, "parsing") {
		t.Errorf("ErrorMsg: got %q, want mention of parse error", got.ErrorMsg)
	}
}
