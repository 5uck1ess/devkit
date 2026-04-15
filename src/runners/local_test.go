package runners

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalRunnerName(t *testing.T) {
	r := &LocalRunner{}
	if r.Name() != "local" {
		t.Errorf("got %q, want %q", r.Name(), "local")
	}
}

func TestLocalRunner_Available_Gating(t *testing.T) {
	tests := []struct {
		name       string
		enabled    string
		serverCode int
		noServer   bool
		want       bool
	}{
		{"env unset", "", 200, false, false},
		{"env=0", "0", 200, false, false},
		{"env=1 no server", "1", 0, true, false},
		{"env=1 server 200", "1", 200, false, true},
		{"env=1 server 401", "1", 401, false, false},
		{"env=1 server 404", "1", 404, false, false},
		{"env=1 server 500", "1", 500, false, false},
		{"env=1 server 503", "1", 503, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEVKIT_LOCAL_ENABLED", tt.enabled)
			endpoint := "http://127.0.0.1:1"
			if !tt.noServer && tt.enabled == "1" {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.serverCode)
				}))
				defer srv.Close()
				endpoint = srv.URL
			}
			t.Setenv("DEVKIT_LOCAL_ENDPOINT", endpoint)

			r := &LocalRunner{}
			if got := r.Available(); got != tt.want {
				t.Errorf("Available() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalRunner_Run_HTTPStatuses(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantErr     bool
		wantExit    int
		wantOutSub  string
		wantErrSub  string
		wantTokIn   int
		wantTokOut  int
		wantSession string
	}{
		{
			name:        "200 valid",
			statusCode:  200,
			body:        `{"id":"sess-1","choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
			wantExit:    0,
			wantOutSub:  "hello",
			wantTokIn:   10,
			wantTokOut:  5,
			wantSession: "sess-1",
		},
		{
			name:       "200 empty choices",
			statusCode: 200,
			body:       `{"id":"sess-2","choices":[],"usage":{"prompt_tokens":3,"completion_tokens":0}}`,
			wantExit:   0,
			wantOutSub: "",
			wantTokIn:  3,
		},
		{
			name:       "200 non-JSON body",
			statusCode: 200,
			body:       `<html>proxy error</html>`,
			wantErr:    true,
			wantExit:   1,
			wantOutSub: "<html>",
			wantErrSub: "non-JSON",
		},
		{"400", 400, `{"error":"bad request"}`, true, 1, "bad request", "HTTP 400", 0, 0, ""},
		{"401", 401, `{"error":"unauthorized"}`, true, 1, "unauthorized", "HTTP 401", 0, 0, ""},
		{"500", 500, `{"error":"server"}`, true, 1, "server", "HTTP 500", 0, 0, ""},
		{"503", 503, ``, true, 1, "", "HTTP 503", 0, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/chat/completions" {
					t.Errorf("path = %q, want /chat/completions", r.URL.Path)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("missing Content-Type header")
				}
				w.WriteHeader(tt.statusCode)
				io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			t.Setenv("DEVKIT_LOCAL_ENDPOINT", srv.URL)
			t.Setenv("DEVKIT_LOCAL_API_KEY", "")

			r := &LocalRunner{}
			res, err := r.Run(context.Background(), "hi", RunOpts{})

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if err != nil && tt.wantErrSub != "" && !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("err = %q, want substring %q", err.Error(), tt.wantErrSub)
			}
			if res.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", res.ExitCode, tt.wantExit)
			}
			if tt.wantOutSub != "" && !strings.Contains(res.Output, tt.wantOutSub) {
				t.Errorf("Output = %q, want substring %q", res.Output, tt.wantOutSub)
			}
			if tt.wantTokIn != 0 && res.TokensIn != tt.wantTokIn {
				t.Errorf("TokensIn = %d, want %d", res.TokensIn, tt.wantTokIn)
			}
			if tt.wantTokOut != 0 && res.TokensOut != tt.wantTokOut {
				t.Errorf("TokensOut = %d, want %d", res.TokensOut, tt.wantTokOut)
			}
			if tt.wantSession != "" && res.SessionID != tt.wantSession {
				t.Errorf("SessionID = %q, want %q", res.SessionID, tt.wantSession)
			}
		})
	}
}

func TestLocalRunner_Run_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			w.WriteHeader(200)
		case <-r.Context().Done():
			return
		}
	}))
	defer srv.Close()

	t.Setenv("DEVKIT_LOCAL_ENDPOINT", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := &LocalRunner{}
	res, err := r.Run(ctx, "hi", RunOpts{})
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if res.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", res.ExitCode)
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("err = %q, want context-related message", err.Error())
	}
}

func TestLocalRunner_Run_AuthHeader(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   string
	}{
		{"no key", "", ""},
		{"with key", "sekrit", "Bearer sekrit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAuth = r.Header.Get("Authorization")
				w.WriteHeader(200)
				io.WriteString(w, `{"id":"x","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
			}))
			defer srv.Close()

			t.Setenv("DEVKIT_LOCAL_ENDPOINT", srv.URL)
			t.Setenv("DEVKIT_LOCAL_API_KEY", tt.apiKey)

			r := &LocalRunner{}
			if _, err := r.Run(context.Background(), "hi", RunOpts{}); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotAuth != tt.want {
				t.Errorf("Authorization = %q, want %q", gotAuth, tt.want)
			}
		})
	}
}

func TestLocalRunner_Run_NetworkUnreachable(t *testing.T) {
	t.Setenv("DEVKIT_LOCAL_ENDPOINT", "http://127.0.0.1:1")

	r := &LocalRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := r.Run(ctx, "hi", RunOpts{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if res.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", res.ExitCode)
	}
	if !strings.Contains(err.Error(), "local endpoint request failed") {
		t.Errorf("err = %q, want 'local endpoint request failed'", err.Error())
	}
}

func TestLocalRunner_Run_SystemPromptFile(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		r := &LocalRunner{}
		_, err := r.Run(context.Background(), "hi", RunOpts{AppendSystemPromptFile: "/nonexistent/path/xyz"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "reading system prompt file") {
			t.Errorf("err = %q", err.Error())
		}
	})

	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		promptFile := filepath.Join(dir, "sys.txt")
		if err := os.WriteFile(promptFile, []byte("you are a test fixture"), 0644); err != nil {
			t.Fatal(err)
		}

		var gotBody []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(200)
			io.WriteString(w, `{"id":"x","choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
		}))
		defer srv.Close()

		t.Setenv("DEVKIT_LOCAL_ENDPOINT", srv.URL)

		r := &LocalRunner{}
		if _, err := r.Run(context.Background(), "hi", RunOpts{AppendSystemPromptFile: promptFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var req localChatRequest
		if err := json.Unmarshal(gotBody, &req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(req.Messages) != 2 {
			t.Fatalf("messages = %d, want 2 (system + user)", len(req.Messages))
		}
		if req.Messages[0].Role != roleSystem {
			t.Errorf("messages[0].role = %q, want system", req.Messages[0].Role)
		}
		if req.Messages[0].Content != "you are a test fixture" {
			t.Errorf("messages[0].content = %q", req.Messages[0].Content)
		}
		if req.Messages[1].Role != roleUser {
			t.Errorf("messages[1].role = %q, want user", req.Messages[1].Role)
		}
	})
}

func TestLocalRunner_EndpointAndModelDefaults(t *testing.T) {
	tests := []struct {
		name         string
		endpointEnv  string
		modelEnv     string
		wantEndpoint string
		wantModel    string
	}{
		{"both unset", "", "", "http://localhost:11434/v1", "qwen3:32b"},
		{"endpoint override", "http://example.com/v1", "", "http://example.com/v1", "qwen3:32b"},
		{"model override", "", "llama3.3:70b", "http://localhost:11434/v1", "llama3.3:70b"},
		{"both overridden", "http://x/v1", "mistral:7b", "http://x/v1", "mistral:7b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEVKIT_LOCAL_ENDPOINT", tt.endpointEnv)
			t.Setenv("DEVKIT_LOCAL_MODEL", tt.modelEnv)
			if got := localEndpoint(); got != tt.wantEndpoint {
				t.Errorf("localEndpoint() = %q, want %q", got, tt.wantEndpoint)
			}
			if got := localModel(); got != tt.wantModel {
				t.Errorf("localModel() = %q, want %q", got, tt.wantModel)
			}
		})
	}
}

func TestLocalRunner_Timeout(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"unset", "", 10 * time.Minute},
		{"valid", "60", 60 * time.Second},
		{"bogus", "abc", 10 * time.Minute},
		{"zero", "0", 10 * time.Minute},
		{"negative", "-5", 10 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEVKIT_LOCAL_TIMEOUT", tt.env)
			if got := localTimeout(); got != tt.want {
				t.Errorf("localTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
