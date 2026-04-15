package runners

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// LocalRunner dispatches to an OpenAI-compatible local endpoint (Ollama, llama-server, vLLM).
// Gated behind DEVKIT_LOCAL_ENABLED=1 so it's opt-in — local models have higher
// tool-call error rates than cloud tiers and can corrupt enforce:hard workflow state.
//
// Config via env:
//
//	DEVKIT_LOCAL_ENABLED   — "1" to enable (default: disabled)
//	DEVKIT_LOCAL_ENDPOINT  — base URL (default: http://localhost:11434/v1)
//	DEVKIT_LOCAL_MODEL     — model name (default: qwen3:32b)
//	DEVKIT_LOCAL_API_KEY   — optional bearer token
type LocalRunner struct{}

type localChatRequest struct {
	Model    string         `json:"model"`
	Messages []localMessage `json:"messages"`
	Stream   bool           `json:"stream"`
}

type localMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	if os.Getenv("DEVKIT_LOCAL_ENABLED") != "1" {
		return false
	}
	endpoint := localEndpoint()
	req, err := http.NewRequest(http.MethodGet, endpoint+"/models", nil)
	if err != nil {
		return false
	}
	if key := os.Getenv("DEVKIT_LOCAL_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}

func (r *LocalRunner) Run(ctx context.Context, prompt string, opts RunOpts) (RunResult, error) {
	messages := []localMessage{}
	if opts.AppendSystemPrompt != "" {
		messages = append(messages, localMessage{Role: "system", Content: opts.AppendSystemPrompt})
	}
	if opts.AppendSystemPromptFile != "" {
		data, err := os.ReadFile(opts.AppendSystemPromptFile)
		if err != nil {
			return RunResult{ExitCode: 1}, fmt.Errorf("reading system prompt file %q: %w", opts.AppendSystemPromptFile, err)
		}
		messages = append(messages, localMessage{Role: "system", Content: string(data)})
	}
	messages = append(messages, localMessage{Role: "user", Content: prompt})

	body, err := json.Marshal(localChatRequest{
		Model:    localModel(),
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("marshaling local request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, localEndpoint()+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return RunResult{ExitCode: 1}, fmt.Errorf("building local request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := os.Getenv("DEVKIT_LOCAL_API_KEY"); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{Timeout: 10 * time.Minute}
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
		return RunResult{Output: string(raw), ExitCode: resp.StatusCode},
			fmt.Errorf("local endpoint returned %d: %s", resp.StatusCode, TruncStr(string(raw), 200))
	}

	var parsed localChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return RunResult{Output: string(raw), ExitCode: 1},
			fmt.Errorf("local endpoint returned non-JSON: %s", TruncStr(string(raw), 200))
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

func localEndpoint() string {
	if v := os.Getenv("DEVKIT_LOCAL_ENDPOINT"); v != "" {
		return v
	}
	return "http://localhost:11434/v1"
}

func localModel() string {
	if v := os.Getenv("DEVKIT_LOCAL_MODEL"); v != "" {
		return v
	}
	return "qwen3:32b"
}
