package testcase

// TestCase represents a single test case for AI response evaluation
type TestCase struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Prompt      string            `json:"prompt"`      // Input prompt to the AI
	AIResponse  string            `json:"ai_response"` // Response from the AI to be evaluated
	Expectation string            `json:"expectation"` // What the response should contain/achieve
	Criteria    []string          `json:"criteria"`    // Specific criteria to evaluate
	RawData     map[string]string `json:"raw_data"`    // All columns from the original dataset row
}

// CriteriaScore represents evaluation score for a single criterion (Deprecated but kept for backward compatibility)
type CriteriaScore struct {
	Score     float64 `json:"score"`
	Reasoning string  `json:"reasoning"`
	MaxScore  float64 `json:"max_score"`
}

// EvaluationDimension represents a specific dimension of evaluation (e.g., Accuracy, Completeness)
type EvaluationDimension struct {
	Score         float64  `json:"score"`
	Weight        float64  `json:"weight"`
	WeightedScore float64  `json:"weighted_score"`
	Evidence      []string `json:"evidence"`
	Strengths     []string `json:"strengths"`
	Weaknesses    []string `json:"weaknesses"`
	Reasoning     string   `json:"reasoning"`
}

// CriticalIssue represents a major problem found in the response
type CriticalIssue struct {
	Severity   string `json:"severity"` // high, medium, low
	Issue      string `json:"issue"`
	Location   string `json:"location"` // Quote or reference
	Impact     string `json:"impact"`
	Suggestion string `json:"suggestion"`
}

// ImprovementRecommendation represents actionable advice
type ImprovementRecommendation struct {
	Priority        string `json:"priority"` // high, medium, low
	Area            string `json:"area"`
	CurrentState    string `json:"current_state"`
	SuggestedChange string `json:"suggested_change"`
	ExpectedImpact  string `json:"expected_impact"`
}

// JudgeComparison captures how the response compares to expectations
type JudgeComparison struct {
	AlignmentScore float64  `json:"alignment_score"`
	Gaps           []string `json:"gaps"`
	Exceeds        []string `json:"exceeds"`
}

// JudgeResult represents the comprehensive evaluation result from LLM judge
type JudgeResult struct {
	// Identifiers
	TestCaseID string `json:"test_case_id"`

	// Input context (for reference)
	Prompt      string `json:"prompt,omitempty"`
	AIResponse  string `json:"ai_response,omitempty"`
	Expectation string `json:"expectation,omitempty"`

	// High-level Result
	OverallScore float64 `json:"overall_score"`
	OverallGrade string  `json:"overall_grade"` // A, B, C, D, F
	Summary      string  `json:"summary"`
	Passed       bool    `json:"passed"` // Computed based on OverallScore

	// Detailed Analysis
	DimensionScores map[string]EvaluationDimension `json:"dimension_scores"`
	CriticalIssues  []CriticalIssue                `json:"critical_issues"`
	Improvements    []ImprovementRecommendation    `json:"improvement_recommendations"`
	Exemplary       []string                       `json:"exemplary_elements"`

	// Confidence & Metadata
	Confidence       string          `json:"confidence_level"`
	ConfidenceReason string          `json:"confidence_reasoning"`
	Comparison       JudgeComparison `json:"comparison_to_expectation"`

	// Legacy/Backward Compatibility fields
	Score          float64                  `json:"score,omitempty"` // Map to OverallScore
	Reasoning      string                   `json:"reasoning,omitempty"`
	CriteriaScores map[string]CriteriaScore `json:"criteria_scores,omitempty"` // Deprecated but kept for now
	Strengths      []string                 `json:"strengths,omitempty"`       // Now in Exemplary or DimensionScores
	Weaknesses     []string                 `json:"weaknesses,omitempty"`      // Now in CriticalIssues or DimensionScores
	Suggestions    []string                 `json:"suggestions,omitempty"`     // Now in Improvements
}

// TestSuite contains multiple test cases
type TestSuite struct {
	Name      string     `json:"name"`
	TestCases []TestCase `json:"test_cases"`
}
