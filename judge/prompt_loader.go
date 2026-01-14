package judge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"openllmjudge/testcase"
)

// PromptTemplate represents a loaded prompt template
type PromptTemplate struct {
	Name      string
	Path      string
	Content   string
	Variables []string
}

// LoadPromptTemplate loads a prompt template from a file
func LoadPromptTemplate(path string) (*PromptTemplate, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	content := string(contentBytes)

	// Extract variables {{variable}}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	var variables []string

	for _, match := range matches {
		if len(match) > 1 {
			v := strings.TrimSpace(match[1])
			if !seen[v] {
				variables = append(variables, v)
				seen[v] = true
			}
		}
	}

	return &PromptTemplate{
		Name:      filepath.Base(path),
		Path:      path,
		Content:   content,
		Variables: variables,
	}, nil
}

// DefaultPrompt returns the hardcoded default prompt
func DefaultPrompt() string {
	return `# Expert AI Response Evaluator

You are an expert evaluator specializing in assessing AI-generated responses. Your evaluations must be:
- **Objective**: Based on observable evidence, not assumptions
- **Detailed**: Provide specific examples from the response
- **Constructive**: Offer actionable feedback for improvement
- **Consistent**: Apply criteria uniformly across all evaluations

## Evaluation Context

**Original Question**: 
{{prompt}}

**AI Response to Evaluate**: 
{{ai_response}}

**Expected Quality Benchmarks**: 
{{expectation}}

**Specific Criteria**: 
{{criteria}}

---

## Evaluation Dimensions

Evaluate the response across these dimensions:

### 1. Accuracy & Correctness
- Are all factual claims correct?
- Are there any misleading or incorrect statements?

### 2. Completeness
- Does it address all aspects of the question?
- Are important points missing?

### 3. Clarity & Structure
- Is the response well-organized and easy to follow?
- Are explanations clear and unambiguous?

### 4. Relevance
- Does the response stay on topic?
- Is all included information pertinent?

### 5. Actionability
- Can the user apply this information?
- Are practical steps or examples provided?

---

## Output Requirements

Provide your evaluation in the following JSON structure:

{
  "overall_score": <0-100 float, VERY IMPORTANT: scale is 0 to 100>,
  "overall_grade": "<A/B/C/D/F>",
  "summary": "<2-3 sentence overall assessment>",
  
  "dimension_scores": {
    "accuracy": {
      "score": <0-100 float>,
      "weight": 0.30,
      "weighted_score": <calculated>,
      "evidence": ["<quote>"],
      "strengths": ["<text>"],
      "weaknesses": ["<text>"],
      "reasoning": "<text>"
    },
    "completeness": {
      "score": <0-100 float>,
      "weight": 0.25,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "clarity": {
      "score": <0-100 float>,
      "weight": 0.20,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "relevance": {
      "score": <0-100 float>,
      "weight": 0.15,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    },
    "actionability": {
      "score": <0-100 float>,
      "weight": 0.10,
      "weighted_score": <calculated>,
      "evidence": [],
      "strengths": [],
      "weaknesses": [],
      "reasoning": "<text>"
    }
  },
  
  "critical_issues": [
    {
      "severity": "<high/medium/low>",
      "issue": "<text>",
      "location": "<text>",
      "impact": "<text>",
      "suggestion": "<text>"
    }
  ],
  
  "improvement_recommendations": [
    {
      "priority": "<high/medium/low>",
      "area": "<dimension>",
      "current_state": "<text>",
      "suggested_change": "<text>",
      "expected_impact": "<text>"
    }
  ],
  
  "exemplary_elements": ["<text>"],
  
  "confidence_level": "<high/medium/low>",
  "confidence_reasoning": "<text>",
  
  "comparison_to_expectation": {
    "alignment_score": <0-100>,
    "gaps": ["<text>"],
    "exceeds": ["<text>"]
  }
}

IMPORTANT: The "overall_score" MUST be between 0 and 100. If the quality is 9/10, output 90.0.`
}

// Render substitutes variables in the template with values from the test case and custom values
func (pt *PromptTemplate) Render(tc testcase.TestCase, customValues map[string]string) string {
	result := pt.Content

	// Standard variables
	replacements := map[string]string{
		"prompt":      tc.Prompt,
		"ai_response": tc.AIResponse,
		"expectation": tc.Expectation,
	}

	// Criteria list
	criteriaList := ""
	if len(tc.Criteria) == 0 {
		criteriaList = "Evaluate against standard AI quality dimensions (Accuracy, Clarity, Relevance)"
	} else {
		for i, c := range tc.Criteria {
			criteriaList += fmt.Sprintf("%d. %s\n", i+1, c)
		}
	}
	replacements["criteria"] = criteriaList

	// Add custom values
	for k, v := range customValues {
		replacements[k] = v
	}

	// Apply replacements
	for k, v := range replacements {
		key := fmt.Sprintf("{{%s}}", k)
		result = strings.ReplaceAll(result, key, v)
	}

	return result
}
