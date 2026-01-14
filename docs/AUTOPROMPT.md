# Autonomous Prompt Engineer (WIP)

Automatically test, analyze, and improve prompts using AI agents.

## Workflow

```
1. Download prompt (GET vendor) → local file
2. Run test cases with selected judge
3. If fail → Agent analyzes & suggests improvements
4. Edit prompt → Upload (UPDATE vendor)
5. Repeat until pass or max iterations
```

## Usage

```bash
# Interactive TUI
openllmjudge auto-prompt

# Direct mode
openllmjudge auto-prompt --judge=<name> --config=<name>
```

## Configuration

Add to `config.json`:

```json
{
  "auto_prompts": [
    {
      "name": "Production Optimizer",
      "get_prompt_vendor": "vendor-name-for-get",
      "update_prompt_vendor": "vendor-name-for-update",
      "chat_vendor": "vendor-name-for-analysis",
      "max_iterations": 10,
      "min_pass_score": 80
    }
  ]
}
```

## Config Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique config name |
| `get_prompt_vendor` | Yes | Vendor to download prompt |
| `update_prompt_vendor` | Yes | Vendor to upload prompt |
| `chat_vendor` | Yes | LLM for analysis/suggestions |
| `max_iterations` | No | Max retry attempts (default: 5) |
| `min_pass_score` | No | Min score to pass (default: 70) |

## Agent System Prompt

Customize the agent behavior by editing:

```
prompts/agent_system_prompt.md
```

This file controls how the agent analyzes failures and generates improvements.

## Vendors

All three vendors must be defined in `./vendors/` folder:

- **GET vendor**: Returns prompt content via API
- **UPDATE vendor**: Accepts prompt content and saves it
- **CHAT vendor**: LLM for analyzing failures and suggesting improvements

## Output

- **Prompts**: Saved to `prompts/refined/prompt_iter{N}_{timestamp}.md`
- **Logs**: Displayed in real-time during execution
- **Results**: Final pass/fail status with iteration count

## Example Vendors

**GET vendor** (`vendors/prompt-get.json`):
```json
{
  "name": "prompt-get",
  "url": "https://api.example.com/prompt",
  "method": "GET",
  "response_path": "content"
}
```

**UPDATE vendor** (`vendors/prompt-update.json`):
```json
{
  "name": "prompt-update",
  "url": "https://api.example.com/prompt",
  "method": "POST",
  "body_template": "{\"prompt\":\"{{prompt}}\"}"
}
```

**CHAT vendor**: Use any LLM vendor (OpenAI, Anthropic, etc.)
