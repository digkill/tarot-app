package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOpenAIBaseURL = "https://api.openai.com"
const openAIChatPath = "/v1/chat/completions"

type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
}

func NewOpenAIClient(apiKey, baseURL, model string) *OpenAIClient {
	if strings.TrimRight(baseURL, "/") == "" {
		baseURL = defaultOpenAIBaseURL
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
	}
}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	ResponseFormat *openAIResponseFmt  `json:"response_format,omitempty"`
	Temperature    float64             `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFmt struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (c *OpenAIClient) Interpret(ctx context.Context, req InterpretRequest) (Insight, error) {
	body := openAIChatRequest{
		Model: c.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt()},
			{Role: "user", Content: buildPrompt(req)},
		},
		ResponseFormat: &openAIResponseFmt{Type: "json_object"},
		Temperature:    0.7,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Insight{}, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+openAIChatPath, bytes.NewReader(payload))
	if err != nil {
		return Insight{}, fmt.Errorf("build openai http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return Insight{}, fmt.Errorf("openai http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Insight{}, fmt.Errorf("read openai response body: %w", err)
	}

	text, err := parseOpenAIOutput(resp.StatusCode, raw)
	if err != nil {
		return Insight{}, err
	}

	insight, err := parseInsight(text, "")
	if err != nil {
		return Insight{}, err
	}
	return insight, nil
}

func parseOpenAIOutput(statusCode int, raw []byte) (string, error) {
	var parsed openAIChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		if statusCode >= 400 {
			return "", fmt.Errorf("openai api http %d", statusCode)
		}
		return "", fmt.Errorf("unmarshal openai response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("openai api error")
	}
	if statusCode >= 400 {
		return "", fmt.Errorf("openai api http %d", statusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty response from openai")
	}
	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("empty response from openai")
	}
	return text, nil
}
