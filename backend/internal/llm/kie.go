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

const defaultKieBaseURL = "https://api.kie.ai"
const kieResponsesPath = "/codex/v1/responses"

var httpClient = &http.Client{Timeout: 90 * time.Second}

type CardEntry struct {
	PositionIndex       int      `json:"positionIndex"`
	PositionTitle       string   `json:"positionTitle"`
	PositionDescription string   `json:"positionDescription"`
	CardName            string   `json:"cardName"`
	UprightMeaning      string   `json:"uprightMeaning"`
	ReversedMeaning     string   `json:"reversedMeaning"`
	UprightKeywords     []string `json:"uprightKeywords"`
	ReversedKeywords    []string `json:"reversedKeywords"`
	IsReversed          bool     `json:"isReversed"`
}

type InterpretRequest struct {
	SpreadID          string      `json:"spreadId"`
	SpreadName        string      `json:"spreadName"`
	SpreadDescription string      `json:"spreadDescription"`
	Language          string      `json:"language"`
	Cards             []CardEntry `json:"cards"`
}

type InsightPosition struct {
	PositionIndex int    `json:"positionIndex"`
	PositionTitle string `json:"positionTitle"`
	CardName      string `json:"cardName"`
	Orientation   string `json:"orientation"`
	Meaning       string `json:"meaning"`
}

type Insight struct {
	Summary   string            `json:"summary"`
	Positions []InsightPosition `json:"positions"`
	Model     string            `json:"-"`
}

type kieContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type kieMessage struct {
	Role    string           `json:"role"`
	Content []kieContentPart `json:"content"`
}

type kieRequest struct {
	Model     string        `json:"model"`
	Stream    bool          `json:"stream"`
	Input     []kieMessage  `json:"input"`
	Reasoning *kieReasoning `json:"reasoning,omitempty"`
}

type kieReasoning struct {
	Effort string `json:"effort"`
}

type kieAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type kieResponse struct {
	OutputText string `json:"output_text"`
	Status     string `json:"status"`
	Output     []struct {
		Type    string          `json:"type"`
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"output"`
	Error *kieAPIError    `json:"error"`
	Code  json.RawMessage `json:"code"`
	Msg   string          `json:"msg"`
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

func systemPrompt() string {
	schemaJSON, err := json.Marshal(outputSchema)
	if err != nil {
		schemaJSON = []byte(`{"type":"object","required":["summary","positions"]}`)
	}
	return "You are a compassionate tarot expert. Provide grounded, clear guidance without disclaimers. " +
		"Reply with a single JSON object only — no markdown, no extra text. Schema:\n" + string(schemaJSON)
}

func buildPrompt(req InterpretRequest) string {
	lang := languageLabel[req.Language]
	if lang == "" {
		lang = "English"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Spread: %s\n", req.SpreadName))
	if req.SpreadDescription != "" {
		sb.WriteString(fmt.Sprintf("Spread overview: %s\n", req.SpreadDescription))
	}
	sb.WriteString(fmt.Sprintf("Respond in %s.\n\nCards:\n", lang))

	for _, c := range req.Cards {
		orientation := "upright"
		if c.IsReversed {
			orientation = "reversed"
		}
		title := c.PositionTitle
		if title == "" {
			title = fmt.Sprintf("Position %d", c.PositionIndex)
		}
		sb.WriteString(fmt.Sprintf("Position %d: %s\n", c.PositionIndex, title))
		if c.PositionDescription != "" {
			sb.WriteString(fmt.Sprintf("Position detail: %s\n", c.PositionDescription))
		}
		sb.WriteString(fmt.Sprintf("Card: %s\nOrientation: %s\n", c.CardName, orientation))
		if c.UprightMeaning != "" {
			sb.WriteString(fmt.Sprintf("Upright meaning: %s\n", c.UprightMeaning))
		}
		if c.ReversedMeaning != "" {
			sb.WriteString(fmt.Sprintf("Reversed meaning: %s\n", c.ReversedMeaning))
		}
		if len(c.UprightKeywords) > 0 {
			sb.WriteString(fmt.Sprintf("Upright keywords: %s\n", strings.Join(c.UprightKeywords, ", ")))
		}
		if len(c.ReversedKeywords) > 0 {
			sb.WriteString(fmt.Sprintf("Reversed keywords: %s\n", strings.Join(c.ReversedKeywords, ", ")))
		}
		sb.WriteByte('\n')
	}

	sb.WriteString("Generate concise, empowering guidance with practical advice.")
	return sb.String()
}

type Client struct {
	apiKey          string
	baseURL         string
	model           string
	reasoningEffort string
}

func NewClient(apiKey, baseURL, model, reasoningEffort string) *Client {
	if strings.TrimRight(baseURL, "/") == "" {
		baseURL = defaultKieBaseURL
	}
	if model == "" {
		model = "gpt-5-6-luna"
	}
	if reasoningEffort == "" {
		reasoningEffort = "low"
	}
	return &Client{
		apiKey:          apiKey,
		baseURL:         strings.TrimRight(baseURL, "/"),
		model:           model,
		reasoningEffort: reasoningEffort,
	}
}

func (c *Client) Model() string {
	return c.model
}

func (c *Client) Interpret(ctx context.Context, req InterpretRequest) (Insight, error) {
	body := kieRequest{
		Model:  c.model,
		Stream: false,
		Input: []kieMessage{
			{
				Role: "system",
				Content: []kieContentPart{
					{Type: "input_text", Text: systemPrompt()},
				},
			},
			{
				Role: "user",
				Content: []kieContentPart{
					{Type: "input_text", Text: buildPrompt(req)},
				},
			},
		},
		Reasoning: &kieReasoning{Effort: c.reasoningEffort},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return Insight{}, fmt.Errorf("marshal kie request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+kieResponsesPath, bytes.NewReader(payload))
	if err != nil {
		return Insight{}, fmt.Errorf("build kie http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return Insight{}, fmt.Errorf("kie http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Insight{}, fmt.Errorf("read kie response body: %w", err)
	}

	text, err := parseKieOutput(resp.StatusCode, raw)
	if err != nil {
		return Insight{}, err
	}

	insight, err := parseInsight(text, c.model)
	if err != nil {
		return Insight{}, err
	}
	return insight, nil
}

func parseKieOutput(statusCode int, raw []byte) (string, error) {
	var parsed kieResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		if statusCode >= 400 {
			return "", fmt.Errorf("kie api http %d: %s", statusCode, truncate(string(raw), 300))
		}
		return "", fmt.Errorf("unmarshal kie response: %w", err)
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		kind := parsed.Error.Type
		if kind == "" {
			kind = fmt.Sprintf("http_%d", statusCode)
		}
		return "", fmt.Errorf("kie api error %s: %s", kind, parsed.Error.Message)
	}
	if parsed.Msg != "" && statusCode >= 400 {
		return "", fmt.Errorf("kie api error %s: %s", string(parsed.Code), parsed.Msg)
	}
	if statusCode >= 400 {
		return "", fmt.Errorf("kie api http %d: %s", statusCode, truncate(string(raw), 300))
	}

	text := strings.TrimSpace(parsed.OutputText)
	if text == "" {
		text = extractOutputText(parsed)
	}
	if text == "" {
		return "", fmt.Errorf("empty response from kie")
	}
	return text, nil
}

func extractOutputText(parsed kieResponse) string {
	var parts []string
	for _, item := range parsed.Output {
		if item.Type != "" && item.Type != "message" {
			continue
		}
		if len(item.Content) == 0 {
			continue
		}

		var contentParts []kieContentPart
		if err := json.Unmarshal(item.Content, &contentParts); err == nil {
			for _, part := range contentParts {
				if part.Type == "output_text" || part.Type == "text" || part.Type == "" {
					if strings.TrimSpace(part.Text) != "" {
						parts = append(parts, part.Text)
					}
				}
			}
			continue
		}

		var asString string
		if err := json.Unmarshal(item.Content, &asString); err == nil && strings.TrimSpace(asString) != "" {
			parts = append(parts, asString)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func parseInsight(text, model string) (Insight, error) {
	cleaned := extractJSONObject(text)
	var insight Insight
	if err := json.Unmarshal([]byte(cleaned), &insight); err != nil {
		return Insight{}, fmt.Errorf("parse kie json insight: %w", err)
	}
	if strings.TrimSpace(insight.Summary) == "" {
		return Insight{}, fmt.Errorf("kie insight missing summary")
	}
	insight.Model = model
	return insight, nil
}

func stripJSONFences(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	trimmed = strings.TrimPrefix(trimmed, "```json")
	trimmed = strings.TrimPrefix(trimmed, "```JSON")
	trimmed = strings.TrimPrefix(trimmed, "```")
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

func extractJSONObject(text string) string {
	cleaned := stripJSONFences(text)
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		return cleaned[start : end+1]
	}
	return cleaned
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
