package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// LocalRunner dispatches to an OpenAI-compatible local endpoint (Ollama, llama-server, vLLM).
// Gated behind DEVKIT_LOCAL_ENABLED=1 so it's opt-in. Tier assignment is the caller's
// responsibility (see DetectRunners in runner.go) — this runner does not enforce tiering
// itself; it's intended for fast-tier dispatch because local models have higher tool-call
// error rates than cloud tiers.
//
// Config via env:
//
//	DEVKIT_LOCAL_ENABLED   — "1" to enable (default: disabled)
//	DEVKIT_LOCAL_ENDPOINT  — base URL (default: http://localhost:11434/v1)
//	DEVKIT_LOCAL_MODEL     — model name (default: qwen3:32b)
//	DEVKIT_LOCAL_API_KEY   — optional bearer token
//	DEVKIT_LOCAL_TIMEOUT   — per-request timeout in seconds (default: 600)
//	DEVKIT_LOCAL_DEBUG     — "1" to log probe failures to stderr
type LocalRunner struct{}

type localRole string

const (
	roleSystem    localRole = "system"
	roleUser      localRole = "user"
	roleAssistant localRole = "assistant"
)

type localChatRequest struct {
	Model    string         `json:"model"`
	Messages []localMessage `json:"messages"`
}

type localMessage struct {
	Role    localRole `json:"role"`
	Content string    `json:"content"`
}

type localChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      localMessage `json:"message"`
		FinishReason string       `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (r *LocalRunner) Name() string { return "local" }

func (r *LocalRunner) Available() bool {
	if !LocalEnabled() {
		return false
	}
	endpoint := LocalEndpoint()
	req, err := http.NewRequest(http.MethodGet, endpoint+"/models", nil)
	if err != nil {
		localDebugf("local runner probe: building request for %s: %v", endpoint, err)
		return false
	}
	if key := LocalAPIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		localDebugf("local runner probe: %s unreachable: %v", endpoint, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		localDebugf("local runner probe: %s returned HTTP %d (check DEVKIT_LOCAL_API_KEY / endpoint path)", endpoint, resp.StatusCode)
		return false
	}
	return true
}

func (r *LocalRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	messages := []localMessage{}
	if opts.AppendSystemPrompt != "" {
		messages = append(messages, localMessage{Role: roleSystem, Content: opts.AppendSystemPrompt})
	}
	if opts.AppendSystemPromptFile != "" {
		data, err := os.ReadFile(opts.AppendSystemPromptFile)
		if err != nil {
			return RunResult{ExitCode: 1}, fmt.Errorf("reading system prompt file %q: %w", opts.AppendSystemPromptFile, err)
		}
		messages = append(messages, localMessage{Role: roleSystem, Content: string(data)})
	}
	messages = append(messages, localMessage{Role: roleUser, Content: prompt})

	if len(messages) == 0 {
		return RunResult{ExitCode: 1}, fmt.Errorf("local runner: no messages to send")
	}

	body, err := json.Marshal(localChatRequest{
		Model:    LocalModel(),
		Messages: messages,
	})
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("marshaling local request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, LocalEndpoint()+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("building local request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := LocalAPIKey(); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{Timeout: LocalTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("local endpoint request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("reading local response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return RunResult{Output: string(raw), ExitCode: 1},
			fmt.Errorf("local endpoint returned HTTP %d: %s", resp.StatusCode, TruncStr(string(raw), 200))
	}

	var parsed localChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return RunResult{Output: string(raw), ExitCode: 1},
			fmt.Errorf("local endpoint returned non-JSON (%w): %s", err, TruncStr(string(raw), 200))
	}

	var output string
	if len(parsed.Choices) > 0 {
		output = parsed.Choices[0].Message.Content
	}

	return RunResult{
		Output:    output,
		SessionID: parsed.ID,
		TokensIn:  parsed.Usage.PromptTokens,
		TokensOut: parsed.Usage.CompletionTokens,
		CostUSD:   0,
		ExitCode:  0,
	}, nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LocalEnabled() bool    { return os.Getenv("DEVKIT_LOCAL_ENABLED") == "1" }
func LocalAPIKey() string   { return os.Getenv("DEVKIT_LOCAL_API_KEY") }
func LocalEndpoint() string { return envDefault("DEVKIT_LOCAL_ENDPOINT", "http://localhost:11434/v1") }
func LocalModel() string    { return envDefault("DEVKIT_LOCAL_MODEL", "qwen3:32b") }

func LocalTimeout() time.Duration {
	const fallback = 10 * time.Minute
	raw := os.Getenv("DEVKIT_LOCAL_TIMEOUT")
	if raw == "" {
		return fallback
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		localDebugf("local runner: ignoring DEVKIT_LOCAL_TIMEOUT=%q (want positive integer seconds)", raw)
		return fallback
	}
	return time.Duration(secs) * time.Second
}

func localDebugf(format string, args ...any) {
	if os.Getenv("DEVKIT_LOCAL_DEBUG") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
