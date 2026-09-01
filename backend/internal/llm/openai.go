package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openAIURL = "https://api.openai.com/v1/chat/completions"

var httpClient = &http.Client{Timeout: 60 * time.Second}

type CardEntry struct {
	PositionIndex int    `json:"positionIndex"`
	CardName      string `json:"cardName"`
	IsReversed    bool   `json:"isReversed"`
}

type InterpretRequest struct {
	SpreadID    string      `json:"spreadId"`
	SpreadName  string      `json:"spreadName"`
	Language    string      `json:"language"`
	Cards       []CardEntry `json:"cards"`
}

type InterpretResponse struct {
	Text string `json:"text"`
}

type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	MaxTokens      int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

var languageLabel = map[string]string{
	"en": "English",
	"ru": "Russian",
	"th": "Thai",
	"zh": "Simplified Chinese",
}

var outputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{
			"type":        "string",
			"description": "Overall insight summary for the entire spread, 3-5 sentences.",
		},
		"positions": map[string]any{
			"type":        "array",
			"description": "Detailed guidance for each spread position.",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"positionIndex": map[string]any{"type": "integer"},
					"positionTitle": map[string]any{"type": "string"},
					"cardName":      map[string]any{"type": "string"},
					"orientation":   map[string]any{"type": "string", "enum": []string{"upright", "reversed"}},
					"meaning": map[string]any{
						"type":        "string",
						"description": "1-3 sentences with actionable advice.",
					},
				},
				"required":             []string{"positionIndex", "positionTitle", "cardName", "orientation", "meaning"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"summary", "positions"},
	"additionalProperties": false,
}

func buildPrompt(req InterpretRequest) string {
	lang := languageLabel[req.Language]
	if lang == "" {
		lang = "English"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Spread: %s\n", req.SpreadName))
	sb.WriteString(fmt.Sprintf("Respond in %s.\n\nCards:\n", lang))

	for _, c := range req.Cards {
		orientation := "upright"
		if c.IsReversed {
			orientation = "reversed"
		}
		sb.WriteString(fmt.Sprintf("Position %d: %s (%s)\n", c.PositionIndex, c.CardName, orientation))
	}

	sb.WriteString("\nGenerate concise, empowering guidance with practical advice.")
	return sb.String()
}

type Client struct {
	apiKey string
	model  string
}

func NewClient(apiKey, model string) *Client {
	return &Client{apiKey: apiKey, model: model}
}

func (c *Client) Interpret(ctx context.Context, req InterpretRequest) (string, error) {
	prompt := buildPrompt(req)

	body := openAIRequest{
		Model: c.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "You are a compassionate tarot expert. Provide grounded, clear guidance without disclaimers. Output only JSON following the provided schema.",
			},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &responseFormat{
			Type: "json_schema",
			JSONSchema: &jsonSchema{
				Name:   "tarot_interpretation",
				Schema: outputSchema,
				Strict: true,
			},
		},
		MaxTokens: 2048,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build openai http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read openai response body: %w", err)
	}

	var oaResp openAIResponse
	if err = json.Unmarshal(raw, &oaResp); err != nil {
		return "", fmt.Errorf("unmarshal openai response: %w", err)
	}

	if oaResp.Error != nil {
		return "", fmt.Errorf("openai api error %s: %s", oaResp.Error.Code, oaResp.Error.Message)
	}

	if len(oaResp.Choices) == 0 || oaResp.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("empty response from openai")
	}

	return oaResp.Choices[0].Message.Content, nil
}
