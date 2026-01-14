package judge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"openjudges/config"
	"openjudges/testcase"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Judge evaluates AI responses using LLM as evaluator
type Judge struct {
	name           string
	apiKey         string
	apiURL         string
	model          string
	provider       string
	method         string
	headers        map[string]string
	responsePath   string
	bodyTemplate   string
	httpClient     *http.Client
	openAIClient   *openai.Client
	promptTemplate *PromptTemplate
	promptValues   map[string]string
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EvalResult contains evaluation result with token usage
type EvalResult struct {
	Result *testcase.JudgeResult
	Tokens TokenUsage
}

// NewJudge creates a new Judge instance from config
func NewJudge(cfg config.JudgeConfig) *Judge {
	j := &Judge{
		name:         cfg.Name,
		apiKey:       cfg.LLM.APIKey,
		apiURL:       cfg.LLM.APIURL,
		model:        cfg.LLM.Model,
		provider:     cfg.LLM.Provider,
		method:       cfg.LLM.Method,
		headers:      cfg.LLM.Headers,
		responsePath: cfg.LLM.ResponsePath,
		bodyTemplate: cfg.LLM.BodyTemplate, // Initialized bodyTemplate
		httpClient:   &http.Client{},
		promptValues: cfg.PromptValues,
	}

	// 1. Try specified prompt file
	promptPath := cfg.PromptFile
	// 2. Fallback to default path if not specified
	if promptPath == "" {
		promptPath = "prompts/judges_prompt_dont_change.md"
	}

	pt, err := LoadPromptTemplate(promptPath)
	if err == nil {
		j.promptTemplate = pt
	} else if cfg.PromptFile != "" {
		// Only warn if they explicitly asked for a file that failed
		fmt.Printf("Warning: failed to load prompt file %s: %v\n", cfg.PromptFile, err)
	}

	// 3. Absolute fallback: Hardcoded default
	if j.promptTemplate == nil {
		j.promptTemplate = &PromptTemplate{
			Name:    "internal_default",
			Content: DefaultPrompt(),
		}
	}

	client := openai.NewClient(
		option.WithAPIKey(cfg.LLM.APIKey),
		option.WithBaseURL(strings.TrimSuffix(cfg.LLM.APIURL, "/chat/completions")), // SDK adds the suffix
	)
	j.openAIClient = &client

	return j
}

// Name returns the judge name
func (j *Judge) Name() string {
	return j.name
}

// SetPromptTemplate sets the prompt template for the judge
func (j *Judge) SetPromptTemplate(template *PromptTemplate) {
	j.promptTemplate = template
}

// Client returns the HTTP client (for streaming)
func (j *Judge) Client() *http.Client {
	return j.httpClient
}

// buildHTTPRequest creates an HTTP request with proper headers
func (j *Judge) buildHTTPRequest(ctx context.Context, reqBody []byte) (*http.Request, error) {
	var apiURL string

	switch j.provider {
	case "google":
		apiURL = fmt.Sprintf("%s?key=%s", j.apiURL, j.apiKey)
	default:
		apiURL = j.apiURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	switch j.provider {
	case "anthropic":
		httpReq.Header.Set("x-api-key", j.apiKey)
		httpReq.Header.Set("anthropic-version", "2025-01-01")
		httpReq.Header.Set("anthropic-dangerous-direct-browser-access", "false")
	case "google":
		// API key is in URL
	default:
		httpReq.Header.Set("Authorization", "Bearer "+j.apiKey)
	}

	return httpReq, nil
}

// PrepareEvaluationPrompt prepares the final prompt for the LLM judge
func (j *Judge) PrepareEvaluationPrompt(tc testcase.TestCase) string {
	return j.promptTemplate.Render(tc, j.promptValues)
}

// Evaluate runs the LLM judge on a test case
func (j *Judge) Evaluate(ctx context.Context, tc testcase.TestCase) (*EvalResult, error) {
	prompt := j.PrepareEvaluationPrompt(tc)

	response, tokens, err := j.callLLM(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	result, err := j.ParseJudgeResponse(response, tc.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse judge response: %w", err)
	}

	return &EvalResult{
		Result: result,
		Tokens: tokens,
	}, nil
}

// OpenAI format
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Anthropic format
type anthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	Temperature float64            `json:"temperature"`
	Metadata    *struct {
		UserID string `json:"user_id,omitempty"`
	} `json:"metadata,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Gemini format
type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig *geminiConfig   `json:"generationConfig,omitempty"`
}

type geminiConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	ThinkingLevel   string  `json:"thinkingLevel,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		TotalTokenCount      int `json:"totalTokenCount"`
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata,omitempty"`
}

// buildStructuredOutputFormat creates OpenAI's structured output JSON schema for judge responses
func (j *Judge) buildStructuredOutputFormat() interface{} {
	// Note: This returns an interface{} to be assignable to ResponseFormat field
	// which is a union type that can accept ResponseFormatJSONSchema

	schemaJSON := []byte(`{
		"type": "object",
		"properties": {
			"overall_score": {
				"type": "number",
				"description": "Overall score from 0-10",
				"minimum": 0,
				"maximum": 10
			},
			"criteria_scores": {
				"type": "object",
				"description": "Individual criterion scores (relevance, accuracy, completeness, clarity, helpfulness)",
				"additionalProperties": {
					"type": "object",
					"properties": {
						"score": {
							"type": "number",
							"minimum": 1,
							"maximum": 10
						},
						"reasoning": {
							"type": "string"
						},
						"max_score": {
							"type": "number"
						}
					},
					"required": ["score", "reasoning", "max_score"]
				}
			},
			"strengths": {
				"type": "array",
				"description": "List of strengths in the response",
				"items": {
					"type": "string"
				}
			},
			"weaknesses": {
				"type": "array",
				"description": "List of weaknesses in the response",
				"items": {
					"type": "string"
				}
			},
			"suggestions": {
				"type": "array",
				"description": "List of actionable suggestions for improvement",
				"items": {
					"type": "string"
				}
			},
			"summary": {
				"type": "string",
				"description": "1-2 sentence overall assessment"
			}
		},
		"required": ["overall_score", "criteria_scores", "strengths", "weaknesses", "suggestions", "summary"]
	}`)

	var schema interface{}
	json.Unmarshal(schemaJSON, &schema)

	// For now, return a simple JSON mode response format
	// The structured output feature requires latest OpenAI models (gpt-4o, gpt-4-turbo, etc.)
	// We'll instruct the model via prompt instead for broader compatibility
	return nil
}

func (j *Judge) callLLM(ctx context.Context, prompt string) (string, TokenUsage, error) {
	var reqBody []byte
	var err error

	var httpReq *http.Request

	switch j.provider {
	case "anthropic":
		req := anthropicRequest{
			Model:       j.model,
			MaxTokens:   4096,
			Messages:    []anthropicMessage{{Role: "user", Content: prompt}},
			Temperature: 0,
		}
		reqBody, err = json.Marshal(req)
		if err != nil {
			return "", TokenUsage{}, err
		}
		httpReq, err = http.NewRequestWithContext(ctx, "POST", j.apiURL, bytes.NewBuffer(reqBody))
		if err != nil {
			return "", TokenUsage{}, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-api-key", j.apiKey)
		httpReq.Header.Set("anthropic-version", "2025-01-01")
		httpReq.Header.Set("anthropic-dangerous-direct-browser-access", "false")

	case "google":
		req := geminiRequest{
			Contents: []geminiContent{
				{
					Role:  "user",
					Parts: []geminiPart{{Text: prompt}},
				},
			},
			GenerationConfig: &geminiConfig{
				MaxOutputTokens: 2048,
				Temperature:     0,
			},
		}
		reqBody, err = json.Marshal(req)
		if err != nil {
			return "", TokenUsage{}, err
		}
		u := fmt.Sprintf("%s?key=%s", j.apiURL, j.apiKey)
		httpReq, err = http.NewRequestWithContext(ctx, "POST", u, bytes.NewBuffer(reqBody))
		if err != nil {
			return "", TokenUsage{}, err
		}
		httpReq.Header.Set("Content-Type", "application/json")

	default:
		// Check if it's a "standard" OpenAI-compatible or "generic"
		if j.method != "" || j.responsePath != "" {
			// Generic Provider
			method := j.method
			if method == "" {
				method = "POST"
			}

			u := j.apiURL
			u = strings.ReplaceAll(u, "{{input}}", strings.ReplaceAll(prompt, "\"", "\\\""))
			u = strings.ReplaceAll(u, "{{data}}", strings.ReplaceAll(prompt, "\"", "\\\""))
			u = strings.ReplaceAll(u, "{{model}}", j.model)

			var body io.Reader
			if method != "GET" && method != "DELETE" {
				if j.bodyTemplate != "" {
					// Use custom body template
					b := j.bodyTemplate
					b = strings.ReplaceAll(b, "{{input}}", strings.ReplaceAll(prompt, "\"", "\\\""))
					b = strings.ReplaceAll(b, "{{data}}", strings.ReplaceAll(prompt, "\"", "\\\""))
					b = strings.ReplaceAll(b, "{{model}}", j.model)
					b = strings.ReplaceAll(b, "{{api_key}}", j.apiKey)
					body = bytes.NewBufferString(b)
				} else if !strings.Contains(j.apiURL, "{{input}}") {
					// Default simple JSON body for generic
					reqData := map[string]interface{}{
						"model":       j.model,
						"input":       prompt,
						"temperature": 0,
					}
					jsonBody, _ := json.Marshal(reqData)
					body = bytes.NewBuffer(jsonBody)
				}
			}

			httpReq, err = http.NewRequestWithContext(ctx, method, u, body)
			if err != nil {
				return "", TokenUsage{}, err
			}

			// Add custom headers
			for k, v := range j.headers {
				httpReq.Header.Set(k, strings.ReplaceAll(v, "{{api_key}}", j.apiKey))
			}
			if httpReq.Header.Get("Content-Type") == "" {
				httpReq.Header.Set("Content-Type", "application/json")
			}
			if j.apiKey != "" && httpReq.Header.Get("Authorization") == "" && httpReq.Header.Get("x-api-key") == "" {
				httpReq.Header.Set("Authorization", "Bearer "+j.apiKey)
			}

		} else {
			// Standard OpenAI-compatible
			req := openAIRequest{
				Model:       j.model,
				Messages:    []openAIMessage{{Role: "user", Content: prompt}},
				Temperature: 0,
			}
			reqBody, err = json.Marshal(req)
			if err != nil {
				return "", TokenUsage{}, err
			}
			httpReq, err = http.NewRequestWithContext(ctx, "POST", j.apiURL, bytes.NewBuffer(reqBody))
			if err != nil {
				return "", TokenUsage{}, err
			}
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", "Bearer "+j.apiKey)
		}
	}

	if j.provider == "openai" && (j.method == "" && j.responsePath == "") {
		// Use SDK for standard OpenAI calls with structured output
		// If prompt contains JSON instructions (heuristic), use SystemMessage + UserMessage
		// Otherwise (simple chat), just use UserMessage
		var messages []openai.ChatCompletionMessageParamUnion
		if strings.Contains(prompt, "JSON") || strings.Contains(prompt, "overall_score") {
			// This is a judge evaluation prompt
			messages = []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(prompt),
				openai.UserMessage("Provide the structured evaluation."),
			}

			chatCompletion, err := j.openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Messages:    messages,
				Model:       openai.ChatModel(j.model),
				Temperature: param.NewOpt(0.0),
			})
			if err != nil {
				return "", TokenUsage{}, err
			}

			if len(chatCompletion.Choices) > 0 {
				return chatCompletion.Choices[0].Message.Content, TokenUsage{
					PromptTokens:     int(chatCompletion.Usage.PromptTokens),
					CompletionTokens: int(chatCompletion.Usage.CompletionTokens),
					TotalTokens:      int(chatCompletion.Usage.TotalTokens),
				}, nil
			}
			return "", TokenUsage{}, fmt.Errorf("no response from OpenAI")
		} else {
			// Regular chat - no structured output
			messages = []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}

			chatCompletion, err := j.openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
				Messages:    messages,
				Model:       openai.ChatModel(j.model),
				Temperature: param.NewOpt(0.0),
			})
			if err != nil {
				return "", TokenUsage{}, err
			}

			if len(chatCompletion.Choices) > 0 {
				return chatCompletion.Choices[0].Message.Content, TokenUsage{
					PromptTokens:     int(chatCompletion.Usage.PromptTokens),
					CompletionTokens: int(chatCompletion.Usage.CompletionTokens),
					TotalTokens:      int(chatCompletion.Usage.TotalTokens),
				}, nil
			}
			return "", TokenUsage{}, fmt.Errorf("no response from OpenAI")
		}
	}

	resp, err := j.httpClient.Do(httpReq)
	if err != nil {
		return "", TokenUsage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", TokenUsage{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", TokenUsage{}, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var content string
	var tokens TokenUsage

	switch j.provider {
	case "anthropic":
		var anthropicResp anthropicResponse
		if err := json.Unmarshal(body, &anthropicResp); err != nil {
			return "", TokenUsage{}, err
		}
		if len(anthropicResp.Content) > 0 {
			content = anthropicResp.Content[0].Text
		}
		tokens = TokenUsage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		}
	case "google":
		var geminiResp geminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			return "", TokenUsage{}, err
		}
		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			content = geminiResp.Candidates[0].Content.Parts[0].Text
		}
		tokens = TokenUsage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}
	default:
		// Handle generic or OpenAI
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return "", TokenUsage{}, err
		}

		if j.responsePath != "" {
			// Extract using responsePath
			content = extractByPath(data, j.responsePath)
		} else {
			// Extract assuming OpenAI format
			var openAIResp openAIResponse
			if err := json.Unmarshal(body, &openAIResp); err == nil && len(openAIResp.Choices) > 0 {
				content = openAIResp.Choices[0].Message.Content
				tokens = TokenUsage{
					PromptTokens:     openAIResp.Usage.PromptTokens,
					CompletionTokens: openAIResp.Usage.CompletionTokens,
					TotalTokens:      openAIResp.Usage.TotalTokens,
				}
			}
		}
	}

	if content == "" {
		return "", TokenUsage{}, fmt.Errorf("no response from LLM")
	}

	return content, tokens, nil
}

func (j *Judge) ParseJudgeResponse(response string, testCaseID string) (*testcase.JudgeResult, error) {
	// Try to extract JSON from response (in case there's extra text)
	start := -1
	end := -1
	depth := 0
	for i, c := range response {
		if c == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in response: %s", response)
	}

	jsonStr := response[start:end]
	var result testcase.JudgeResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w\nJSON: %s", err, jsonStr)
	}

	// Set test case ID
	result.TestCaseID = testCaseID

	// Backward compatibility mapping
	// If we got new format but old Score field is empty, map OverallScore to Score
	if result.OverallScore > 0 && result.Score == 0 {
		result.Score = result.OverallScore
	} else if result.Score > 0 && result.OverallScore == 0 {
		// If we got old format with Score but no OverallScore, use Score as OverallScore
		result.OverallScore = result.Score
	}

	// Initialize empty maps if they are nil
	if result.CriteriaScores == nil {
		result.CriteriaScores = make(map[string]testcase.CriteriaScore)
	}
	if result.Strengths == nil {
		result.Strengths = []string{}
	}
	if result.Weaknesses == nil {
		result.Weaknesses = []string{}
	}
	if result.Suggestions == nil {
		result.Suggestions = []string{}
	}

	// Heuristic: If OverallScore is <= 10.0 (but > 0), assume LLM used 0-10 scale
	// and normalize to 0-100. This is common when LLMs ignore the 0-100 instruction.
	if result.OverallScore > 0 && result.OverallScore <= 10.0 {
		result.OverallScore *= 10.0
	}

	// Derive grade if missing
	if result.OverallGrade == "" && result.OverallScore > 0 {
		result.OverallGrade = deriveOverallGrade(result.OverallScore)
	}

	// Calculate pass/fail if not explicitly set (based on overall_score)
	// New scale is 0-100, pass is >= 70
	if result.Passed {
		// If already passed (explicitly), keep it
	} else if result.OverallScore >= 70.0 {
		result.Passed = true
	} else {
		result.Passed = false
	}

	// Backward compatibility: If Score is 0-10 but OverallScore is > 10, map back
	if result.Score == 0 && result.OverallScore > 0 {
		result.Score = result.OverallScore
	}

	return &result, nil
}

func deriveOverallGrade(score float64) string {
	if score >= 90 {
		return "A"
	} else if score >= 80 {
		return "B"
	} else if score >= 70 {
		return "C"
	} else if score >= 60 {
		return "D"
	}
	return "F"
}

// Chat simple direct completion without evaluation
func (j *Judge) Chat(ctx context.Context, prompt string) (string, error) {
	resp, _, err := j.callLLM(ctx, prompt)
	return resp, err
}

func extractByPath(data interface{}, path string) string {
	if path == "" {
		return fmt.Sprintf("%v", data)
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return fmt.Sprintf("%v", current)
		}
	}

	if s, ok := current.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", current)
}
