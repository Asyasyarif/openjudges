package vendor

// VendorConfig represents a single vendor configuration loaded from JSON
type VendorConfig struct {
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Method       string                 `json:"method"`
	Headers      map[string]string      `json:"headers"`
	Body         map[string]interface{} `json:"body"`
	ResponsePath string                 `json:"response_path"`
	APIKey       string                 `json:"api_key"`
	ParseAs      string                 `json:"parse_as"`  // Optional: regex or JSON path for streaming chunks
	Guardrail    []string               `json:"guardrail"` // Optional: strings to skip/filter out from stream
	TimeoutMs    int                    `json:"timeout_ms,omitempty"`
	MaxRetries   int                    `json:"max_retries,omitempty"`
	BackoffMs    int                    `json:"backoff_ms,omitempty"`

	// Filename stores the source JSON filename (not serialized to JSON)
	Filename string `json:"-"`
}
