package vendor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"openjudges/testcase"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var placeholderRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

// CallVendorAPI makes an HTTP request to the vendor API
func (v *VendorConfig) CallVendorAPI(ctx context.Context, tc testcase.TestCase) (string, error) {
	// Resolve API key from environment if needed
	apiKey := resolveEnvVar(v.APIKey)

	// Prepare data for replacement
	data := make(map[string]string)
	for k, val := range tc.RawData {
		data[k] = val
	}
	data["api_key"] = apiKey
	data["prompt"] = tc.Prompt
	data["input"] = tc.Prompt
	data["question"] = tc.Prompt // backward compatibility
	data["expectation"] = tc.Expectation
	data["id"] = tc.ID

	// Replace placeholders in URL (with URL encoding for query params)
	targetURL := v.replacePlaceholders(v.URL, data, "url")

	// Determine HTTP method
	method := strings.ToUpper(v.Method)
	if method == "" {
		method = "POST"
	}

	// Prepare request body for POST/PUT
	var bodyReader io.Reader
	if method == "POST" || method == "PUT" {
		bodyBytes, err := v.prepareBody(data)
		if err != nil {
			return "", fmt.Errorf("failed to prepare request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, method, targetURL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers with placeholder substitution
	for key, val := range v.Headers {
		val = v.replacePlaceholders(val, data, "header")
		req.Header.Set(key, val)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vendor API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Extract response
	if v.ParseAs != "" {
		// Handle streaming/SSE response
		return v.extractStreamResponse(resp.Body)
	}

	// Read full response body (non-streaming)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(respBody)
		if msg == "" {
			msg = "(empty response body)"
		}
		return "", fmt.Errorf("vendor API error (status %d): %s", resp.StatusCode, msg)
	}

	// Extract response using response_path
	return v.extractResponse(respBody)
}

// extractStreamResponse reads SSE/line-delimited JSON stream and aggregates content
func (v *VendorConfig) extractStreamResponse(body io.ReadCloser) (string, error) {
	scanner := bufio.NewScanner(body)
	var sb strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Handle SSE "data: " prefix if present
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
		}

		// Skip heartbeat or done messages if needed (e.g. "[DONE]")
		if line == "[DONE]" {
			break
		}

		// Parse JSON Chunk
		var chunk interface{}
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			// Skip non-JSON lines (maybe comments or other SSE fields)
			continue
		}

		// Extract content
		content := extractByPath(chunk, v.ParseAs)

		// Check guardrails
		skip := false
		trimmedContent := strings.TrimSpace(content)
		for _, g := range v.Guardrail {
			if trimmedContent == g {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		sb.WriteString(content)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	return sb.String(), nil
}

// prepareBody marshals the body with placeholder substitution
func (v *VendorConfig) prepareBody(data map[string]string) ([]byte, error) {
	if len(v.Body) == 0 {
		return nil, nil
	}

	// Marshal body to JSON
	bodyJSON, err := json.Marshal(v.Body)
	if err != nil {
		return nil, err
	}

	// Replace placeholders in the JSON string
	bodyStr := string(bodyJSON)
	bodyStr = v.replacePlaceholders(bodyStr, data, "json")

	return []byte(bodyStr), nil
}

func (v *VendorConfig) replacePlaceholders(input string, data map[string]string, mode string) string {
	return placeholderRegex.ReplaceAllStringFunc(input, func(match string) string {
		key := strings.TrimSpace(match[2 : len(match)-2])
		val, ok := data[strings.ToLower(key)]
		if !ok {
			return match // Keep original if not found
		}

		switch mode {
		case "url":
			return url.QueryEscape(val)
		case "json":
			return escapeJSONString(val)
		default:
			return val
		}
	})
}

// extractResponse extracts the response text using response_path
func (v *VendorConfig) extractResponse(body []byte) (string, error) {
	// If no response path specified, return raw body
	if v.ResponsePath == "" {
		return string(body), nil
	}

	// Parse JSON response
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// If not JSON, return raw body
		return string(body), nil
	}

	// Extract by path
	return extractByPath(result, v.ResponsePath), nil
}

// extractByPath extracts a value from nested JSON using dot notation
// Example: "choices.0.text" extracts result["choices"][0]["text"]
func extractByPath(data interface{}, path string) string {
	if path == "" {
		return fmt.Sprintf("%v", data)
	}

	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		case []interface{}:
			// Support array index access (e.g., "0", "1")
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return fmt.Sprintf("%v", current)
			}
			current = v[idx]
		default:
			return fmt.Sprintf("%v", current)
		}
	}

	if s, ok := current.(string); ok {
		return s
	}
	if current == nil {
		return ""
	}
	return fmt.Sprintf("%v", current)
}

// resolveEnvVar resolves environment variable syntax ${VAR_NAME}
func resolveEnvVar(value string) string {
	if len(value) > 3 && value[:2] == "${" && value[len(value)-1] == '}' {
		envName := value[2 : len(value)-1]
		if envVal := os.Getenv(envName); envVal != "" {
			return envVal
		}
	}
	return value
}

// escapeJSONString escapes a string for use inside JSON
// This handles special characters that need escaping
func escapeJSONString(s string) string {
	// Use json.Marshal to properly escape the string, then remove the quotes
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}
	// Remove the surrounding quotes added by Marshal
	escaped := string(b)
	if len(escaped) >= 2 && escaped[0] == '"' && escaped[len(escaped)-1] == '"' {
		return escaped[1 : len(escaped)-1]
	}
	return escaped
}
