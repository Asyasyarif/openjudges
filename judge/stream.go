package judge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"

	"openjudges/internal/tools"
)

// TokenType represents the type of streaming token
type TokenType int

const (
	TokenContent  TokenType = iota // Regular content token
	TokenThinking                  // Thinking/reasoning token (Claude extended thinking)
	TokenStop                      // Stream end signal
)

// StreamConfig configures streaming behavior
type StreamConfig struct {
	OnToken    func(token string, tokenType TokenType, usage TokenUsage) // Callback for each token
	BufferSize int                                                       // Buffer size for reading stream
}

// DefaultStreamConfig returns default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		OnToken:    func(string, TokenType, TokenUsage) {}, // No-op by default
		BufferSize: 4096,
	}
}

// defaultTools defines the built-in tools available to judges
var defaultTools = []openai.ChatCompletionToolParam{
	{
		Function: shared.FunctionDefinitionParam{
			Name:        "read",
			Description: openai.String("Read file content with line numbers. Use offset and limit for pagination."),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"path":   map[string]string{"type": "string", "description": "File path to read"},
					"offset": map[string]string{"type": "integer", "description": "Line offset (0-based)"},
					"limit":  map[string]string{"type": "integer", "description": "Max lines to return"},
				},
				"required": []string{"path"},
			},
		},
	},
	{
		Function: shared.FunctionDefinitionParam{
			Name:        "write",
			Description: openai.String("Write content to a file. Creates directories if needed."),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"path":    map[string]string{"type": "string", "description": "File path to write"},
					"content": map[string]string{"type": "string", "description": "Content to write"},
				},
				"required": []string{"path", "content"},
			},
		},
	},
	{
		Function: shared.FunctionDefinitionParam{
			Name:        "edit",
			Description: openai.String("Replace a unique string in a file. Fails if string is not found or not unique."),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"path":     map[string]string{"type": "string", "description": "File path to edit"},
					"old_text": map[string]string{"type": "string", "description": "Text to find and replace"},
					"new_text": map[string]string{"type": "string", "description": "Replacement text"},
				},
				"required": []string{"path", "old_text", "new_text"},
			},
		},
	},
	{
		Function: shared.FunctionDefinitionParam{
			Name:        "glob",
			Description: openai.String("Find files matching a glob pattern, sorted by modification time (newest first)."),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]string{"type": "string", "description": "Glob pattern (e.g., *.go)"},
				},
				"required": []string{"pattern"},
			},
		},
	},
	{
		Function: shared.FunctionDefinitionParam{
			Name:        "grep",
			Description: openai.String("Search files for a regex pattern. Returns matching lines with file paths and line numbers."),
			Parameters: shared.FunctionParameters{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]string{"type": "string", "description": "Regex pattern to search for"},
					"path":    map[string]string{"type": "string", "description": "Directory or file path to search"},
				},
				"required": []string{"pattern", "path"},
			},
		},
	},
}

// EvaluateStreaming evaluates a test case with streaming response
func (j *Judge) EvaluateStreaming(ctx context.Context, prompt string, config StreamConfig) (string, TokenUsage, error) {
	if config.BufferSize == 0 {
		config.BufferSize = 4096
	}

	if j.provider == "openai" && (j.method == "" && j.responsePath == "") {
		// Distinguish between chat and evaluation (roasting)
		isJudge := strings.Contains(prompt, "JSON") || strings.Contains(prompt, "overall_score")

		var messages []openai.ChatCompletionMessageParamUnion
		if isJudge {
			messages = append(messages, openai.SystemMessage(JudgeMasterPrompt))
			messages = append(messages, openai.UserMessage(prompt))
		} else {
			messages = append(messages, openai.SystemMessage(ChatMasterPrompt))
			messages = append(messages, openai.UserMessage(prompt))
		}

		// Tool Loop (Sequential tool calls supported)
		for {
			// Build parameters
			params := openai.ChatCompletionNewParams{
				Model:       j.model,
				Temperature: openai.Float(0.0),
				StreamOptions: openai.ChatCompletionStreamOptionsParam{
					IncludeUsage: param.NewOpt(true),
				},
			}
			params.Messages = messages

			// Add tools only for judge evaluation
			if isJudge {
				params.Tools = defaultTools
			}

			stream := j.openAIClient.Chat.Completions.NewStreaming(ctx, params)

			var fullText strings.Builder
			var currentUsage TokenUsage
			var toolCalls []openai.ChatCompletionMessageToolCallParam

			for stream.Next() {
				chunk := stream.Current()
				if chunk.Usage.TotalTokens > 0 {
					currentUsage = TokenUsage{
						PromptTokens:     int(chunk.Usage.PromptTokens),
						CompletionTokens: int(chunk.Usage.CompletionTokens),
						TotalTokens:      int(chunk.Usage.TotalTokens),
					}
				}
				if len(chunk.Choices) > 0 {
					choice := chunk.Choices[0]
					// Handle Content
					delta := choice.Delta.Content
					if delta != "" {
						fullText.WriteString(delta)
						config.OnToken(delta, TokenContent, currentUsage)
					}

					// Handle Tool Calls (Streaming)
					for _, tc := range choice.Delta.ToolCalls {
						// Ensure toolCalls slice has enough capacity
						for int(tc.Index) >= len(toolCalls) {
							toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{})
						}
						target := &toolCalls[tc.Index]
						if tc.ID != "" {
							target.ID = tc.ID
						}
						if tc.Function.Name != "" {
							target.Function.Name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							// Arguments come in chunks
							target.Function.Arguments += tc.Function.Arguments
						}
					}
				} else if chunk.Usage.TotalTokens > 0 {
					config.OnToken("", TokenContent, currentUsage)
				}
			}

			if err := stream.Err(); err != nil {
				return "", TokenUsage{}, err
			}

			// If no tool calls, we are done
			if len(toolCalls) == 0 {
				config.OnToken("", TokenStop, currentUsage)
				return fullText.String(), currentUsage, nil
			}

			// Execute tools and continue the loop
			// First, add the assistant's message with tool calls to history
			assistantMsg := openai.ChatCompletionMessageParamUnion{}
			assistantMsg.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
				ToolCalls: toolCalls,
			}
			messages = append(messages, assistantMsg)

			for _, tc := range toolCalls {
				result, err := j.executeTool(tc.Function.Name, tc.Function.Arguments)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}
				// Add tool result to history
				messages = append(messages, openai.ToolMessage(tc.ID, result))
			}
			// Loop continues with updated messages
		}
	}

	// For other providers, use the generic streaming logic (no tools support yet)
	// Build request body with stream enabled
	reqBody, err := j.buildStreamingRequest(prompt)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("failed to build streaming request: %w", err)
	}

	// Make HTTP request
	req, err := j.buildHTTPRequest(ctx, reqBody)
	if err != nil {
		return "", TokenUsage{}, err
	}

	resp, err := j.httpClient.Do(req)
	if err != nil {
		return "", TokenUsage{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", TokenUsage{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Read streaming response
	fullText, tokens, err := j.readStreamingResponse(resp.Body, config)
	if err != nil {
		return "", TokenUsage{}, err
	}

	return fullText, tokens, nil
}

// buildStreamingRequest builds request body with streaming enabled
func (j *Judge) buildStreamingRequest(prompt string) ([]byte, error) {
	switch j.provider {
	case "anthropic":
		req := map[string]interface{}{
			"model": j.model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens":  4096,
			"stream":      true,
			"temperature": 0,
		}
		return json.Marshal(req)

	case "openai":
		req := map[string]interface{}{
			"model": j.model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream":      true,
			"temperature": 0,
		}
		return json.Marshal(req)

	case "google":
		// Google uses SSE streaming
		req := map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"parts": []map[string]string{
						{"text": prompt},
					},
				},
			},
		}
		return json.Marshal(req)

	case "groq":
		// Groq uses OpenAI-compatible API
		req := map[string]interface{}{
			"model": j.model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream":      true,
			"temperature": 0,
		}
		return json.Marshal(req)

	case "openrouter":
		// OpenRouter uses OpenAI-compatible API
		req := map[string]interface{}{
			"model": j.model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream":      true,
			"temperature": 0,
		}
		return json.Marshal(req)

	default:
		return nil, fmt.Errorf("streaming not supported for provider: %s", j.provider)
	}
}

// readStreamingResponse reads and parses SSE stream
func (j *Judge) readStreamingResponse(body io.Reader, config StreamConfig) (string, TokenUsage, error) {
	var fullText strings.Builder
	var tokens TokenUsage

	reader := bufio.NewReader(body)

	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", tokens, fmt.Errorf("error reading stream: %w", err)
		}

		// Skip empty lines
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Parse SSE format: "data: {...}"
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))

			// Check for end of stream
			if bytes.Equal(data, []byte("[DONE]")) {
				config.OnToken("", TokenStop, TokenUsage{TotalTokens: int(float64(estimateTokens(fullText.String())) * 1.5)})
				break
			}

			// Parse delta based on provider
			delta, tokenType := j.parseStreamDelta(data)
			if delta != "" {
				fullText.WriteString(delta)
				config.OnToken(delta, tokenType, TokenUsage{})
			}
		}
	}

	// Estimate token usage (streaming doesn't always provide exact counts)
	// We'll update this when the stream completes
	text := fullText.String()
	tokens.PromptTokens = estimateTokens(text) / 2 // Rough estimate
	tokens.CompletionTokens = estimateTokens(text)
	tokens.TotalTokens = tokens.PromptTokens + tokens.CompletionTokens

	return text, tokens, nil
}

// parseStreamDelta parses streaming delta based on provider
func (j *Judge) parseStreamDelta(data []byte) (string, TokenType) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", TokenContent
	}

	switch j.provider {
	case "anthropic":
		return j.parseAnthropicDelta(result)
	case "openai", "groq", "openrouter":
		return j.parseOpenAIDelta(result)
	case "google":
		return j.parseGoogleDelta(result)
	default:
		return "", TokenContent
	}
}

// parseAnthropicDelta parses Anthropic streaming format
func (j *Judge) parseAnthropicDelta(data map[string]interface{}) (string, TokenType) {
	eventType, ok := data["type"].(string)
	if !ok {
		return "", TokenContent
	}

	switch eventType {
	case "content_block_delta":
		if delta, ok := data["delta"].(map[string]interface{}); ok {
			if text, ok := delta["text"].(string); ok {
				// Check if it's thinking content
				if contentType, ok := delta["type"].(string); ok && contentType == "thinking" {
					return text, TokenThinking
				}
				return text, TokenContent
			}
		}
	case "message_delta":
		// Extract token usage if available
		if usage, ok := data["usage"].(map[string]interface{}); ok {
			// Store usage data
			_ = usage
		}
	}

	return "", TokenContent
}

// parseOpenAIDelta parses OpenAI streaming format
func (j *Judge) parseOpenAIDelta(data map[string]interface{}) (string, TokenType) {
	choices, ok := data["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", TokenContent
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", TokenContent
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return "", TokenContent
	}

	if content, ok := delta["content"].(string); ok {
		return content, TokenContent
	}

	return "", TokenContent
}

// parseGoogleDelta parses Google streaming format
func (j *Judge) parseGoogleDelta(data map[string]interface{}) (string, TokenType) {
	candidates, ok := data["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", TokenContent
	}

	candidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return "", TokenContent
	}

	content, ok := candidate["content"].(map[string]interface{})
	if !ok {
		return "", TokenContent
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", TokenContent
	}

	part, ok := parts[0].(map[string]interface{})
	if !ok {
		return "", TokenContent
	}

	if text, ok := part["text"].(string); ok {
		return text, TokenContent
	}

	return "", TokenContent
}

// executeTool dispatches tool calls to actual implementations
func (j *Judge) executeTool(name, argsJSON string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments JSON: %v", err)
	}

	switch name {
	case "read":
		path, _ := args["path"].(string)
		offset, _ := args["offset"].(float64)
		limit, _ := args["limit"].(float64)
		res, err := tools.Read(path, int(offset), int(limit))
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(res)
		return string(data), nil

	case "write":
		path, _ := args["path"].(string)
		content, _ := args["content"].(string)
		err := tools.Write(path, content)
		if err != nil {
			return "", err
		}
		return "File written successfully", nil

	case "edit":
		path, _ := args["path"].(string)
		oldText, _ := args["old_text"].(string)
		newText, _ := args["new_text"].(string)
		err := tools.Edit(path, oldText, newText)
		if err != nil {
			return "", err
		}
		return "File edited successfully", nil

	case "glob":
		pattern, _ := args["pattern"].(string)
		res, err := tools.Glob(pattern)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(res)
		return string(data), nil

	case "grep":
		pattern, _ := args["pattern"].(string)
		path, _ := args["path"].(string)
		res, err := tools.Grep(pattern, path)
		if err != nil {
			return "", err
		}
		data, _ := json.Marshal(res)
		return string(data), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// estimateTokens provides a rough token count estimate
// Real token counts should come from the API response
func estimateTokens(text string) int {
	// Rough estimate: ~4 characters per token
	return len(text) / 4
}
