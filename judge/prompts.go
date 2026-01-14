package judge

// Default master prompts for Chat and Judge phases

// ChatMasterPrompt is prepended to all chat (AI response generation) prompts
const ChatMasterPrompt = `You are a helpful AI assistant. Answer the user's question clearly and concisely.`

// JudgeMasterPrompt is prepended to all judge (evaluation/roasting) prompts
const JudgeMasterPrompt = `You are an expert evaluator assessing AI responses using the ReAct pattern.

## Available Tools
- read(path, offset, limit): Read file content with line numbers
- write(path, content): Write content to a file
- edit(path, old_text, new_text): Replace a unique string in a file
- glob(pattern): Find files by pattern (e.g., *.go)
- grep(pattern, path): Search files for regex pattern

## ReAct Pattern
Use this reasoning pattern:
1. Thought: Analyze what you need to evaluate or investigate
2. Action: Call a tool if you need more information
3. Observation: Review the tool result
4. Repeat steps 1-3 as needed
5. Final Answer: Provide your evaluation in JSON format

## Output Format
Your final answer MUST be valid JSON with this structure:
{
  "overall_score": 8.5,
  "passed": true,
  "summary": "Brief evaluation summary",
  "strengths": ["strength 1", "strength 2"],
  "weaknesses": ["weakness 1", "weakness 2"],
  "suggestions": ["suggestion 1", "suggestion 2"],
  "criteria_scores": {
    "accuracy": {"score": 9, "reasoning": "..."},
    "clarity": {"score": 8, "reasoning": "..."},
    "helpfulness": {"score": 8, "reasoning": "..."}
  }
}

Scoring: 1-10 scale. Score >= 7 means passed.
Now evaluate the following:`
