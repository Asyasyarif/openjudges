package vendor

import (
	"errors"
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

func TestShouldRetryByStatusCode(t *testing.T) {
	if !shouldRetry(nil, 429) {
		t.Fatalf("expected retry for HTTP 429")
	}

	if !shouldRetry(nil, 503) {
		t.Fatalf("expected retry for HTTP 503")
	}

	if shouldRetry(nil, 400) {
		t.Fatalf("did not expect retry for HTTP 400")
	}
}

func TestShouldRetryByErrorText(t *testing.T) {
	err := errors.New("stream reading error: stream error: stream ID 63; INTERNAL_ERROR; received from peer")
	if !shouldRetry(err, 200) {
		t.Fatalf("expected retry for stream internal error")
	}
}
