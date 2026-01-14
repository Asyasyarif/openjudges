package vendor

import (
	"io"
	"strings"
	"testing"
)

func TestExtractStreamResponse(t *testing.T) {
	v := &VendorConfig{
		ParseAs: "content",
	}

	mockResponse := `data: {"content": "Hello "}

data: {"content": "world"}

data: [DONE]`

	body := io.NopCloser(strings.NewReader(mockResponse))

	result, err := v.extractStreamResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello world"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractStreamResponseWithNestedPath(t *testing.T) {
	v := &VendorConfig{
		ParseAs: "data.content",
	}

	mockResponse := `data: {"data": {"content": "Part 1"}}
data: {"data": {"content": "Part 2"}}`

	body := io.NopCloser(strings.NewReader(mockResponse))

	result, err := v.extractStreamResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Part 1Part 2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestExtractStreamResponseWithGuardrail(t *testing.T) {
	v := &VendorConfig{
		ParseAs:   "content",
		Guardrail: []string{"Stream completed successfully"},
	}

	mockResponse := `data: {"content": "Here is response. "}
data: {"content": "Stream completed successfully"}
data: {"content": "End."}`
	// Note: "End." should be kept. "Stream completed successfully" should be skipped.

	body := io.NopCloser(strings.NewReader(mockResponse))

	result, err := v.extractStreamResponse(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Here is response. End."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
