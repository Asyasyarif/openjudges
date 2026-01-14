package judge

import (
	"testing"
)

func TestParseJudgeResponse_GradeDerivation(t *testing.T) {
	j := &Judge{}

	tests := []struct {
		name          string
		response      string
		expectedScore float64
		expectedGrade string
	}{
		{
			name:          "derive A from 95",
			response:      `{"overall_score": 95.0, "summary": "Great"}`,
			expectedScore: 95.0,
			expectedGrade: "A",
		},
		{
			name:          "derive B from 85",
			response:      `{"overall_score": 85.0, "summary": "Good"}`,
			expectedScore: 85.0,
			expectedGrade: "B",
		},
		{
			name:          "derive C from 75",
			response:      `{"overall_score": 75.0, "summary": "Okay"}`,
			expectedScore: 75.0,
			expectedGrade: "C",
		},
		{
			name:          "derive D from 65",
			response:      `{"overall_score": 65.0, "summary": "Poor"}`,
			expectedScore: 65.0,
			expectedGrade: "D",
		},
		{
			name:          "derive F from 55",
			response:      `{"overall_score": 55.0, "summary": "Fail"}`,
			expectedScore: 55.0,
			expectedGrade: "F",
		},
		{
			name:          "normalize 9.0 to 90 and derive A",
			response:      `{"overall_score": 9.0, "summary": "Great"}`,
			expectedScore: 90.0,
			expectedGrade: "A",
		},
		{
			name:          "keep existing grade",
			response:      `{"overall_score": 95.0, "overall_grade": "B", "summary": "Great"}`,
			expectedScore: 95.0,
			expectedGrade: "B", // Should not overwrite if present
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := j.ParseJudgeResponse(tt.response, "test-1")
			if err != nil {
				t.Fatalf("ParseJudgeResponse failed: %v", err)
			}
			if result.OverallScore != tt.expectedScore {
				t.Errorf("expected score %.1f, got %.1f", tt.expectedScore, result.OverallScore)
			}
			if result.OverallGrade != tt.expectedGrade {
				t.Errorf("expected grade %s, got %s", tt.expectedGrade, result.OverallGrade)
			}
		})
	}
}
