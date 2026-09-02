package llm

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestExtractOutputTextFromMessage(t *testing.T) {
	raw := []byte(`{
		"output": [
			{"type": "reasoning", "id": "rs_1", "summary": []},
			{
				"type": "message",
				"role": "assistant",
				"content": [
					{"type": "output_text", "text": "{\"summary\":\"ok\",\"positions\":[]}"}
				]
			}
		],
		"status": "completed"
	}`)

	text, err := parseKieOutput(http.StatusOK, raw)
	if err != nil {
		t.Fatalf("parseKieOutput: %v", err)
	}
	if text != `{"summary":"ok","positions":[]}` {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseKieOutputUsesOutputText(t *testing.T) {
	raw := []byte(`{"output_text":"{\"summary\":\"from field\",\"positions\":[]}","output":[]}`)
	text, err := parseKieOutput(http.StatusOK, raw)
	if err != nil {
		t.Fatalf("parseKieOutput: %v", err)
	}
	if text != `{"summary":"from field","positions":[]}` {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseKieOutputAuthError(t *testing.T) {
	raw := []byte(`{"code":401,"msg":"You do not have access permissions"}`)
	_, err := parseKieOutput(http.StatusUnauthorized, raw)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseInsightStripsFences(t *testing.T) {
	insight, err := parseInsight("```json\n{\"summary\":\"Hello\",\"positions\":[]}\n```", "gpt-5-6-luna")
	if err != nil {
		t.Fatalf("parseInsight: %v", err)
	}
	if insight.Summary != "Hello" {
		t.Fatalf("summary: %q", insight.Summary)
	}
	if insight.Model != "gpt-5-6-luna" {
		t.Fatalf("model: %q", insight.Model)
	}
}

func TestKieRequestMatchesOpenAPI(t *testing.T) {
	body := kieRequest{
		Model:  "gpt-5-6-luna",
		Stream: false,
		Input: []kieMessage{{
			Role:    "user",
			Content: []kieContentPart{{Type: "input_text", Text: "hi"}},
		}},
		Reasoning: &kieReasoning{Effort: "low"},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err = json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"instructions", "text", "tools", "tool_choice"} {
		if _, ok := asMap[forbidden]; ok {
			t.Fatalf("request must not include %s (not in Kie Luna OpenAPI)", forbidden)
		}
	}
	if asMap["model"] != "gpt-5-6-luna" {
		t.Fatalf("model: %v", asMap["model"])
	}
	if asMap["stream"] != false {
		t.Fatalf("stream must be false, got %v", asMap["stream"])
	}
}
