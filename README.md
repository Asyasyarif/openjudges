# OpenJudges

OpenJudges is a powerful, interactive CLI tool for evaluating Large Language Models (LLMs) using other LLMs as judges. It helps you automate the process of testing and benchmarking model performance on your specific datasets.

## Features

- **Interactive TUI**: A beautiful, terminal-based user interface for managing judges, running evaluations, and analyzing results.
- **Multiple Providers**: Support for OpenAI, Anthropic, Google Gemini, Groq, and OpenRouter.
- **Customizable Judges**: Configure judges with specific models, prompts, and datasets.
- **Detailed Results**: View comprehensive evaluation reports, including reasoning and scores, directly in your terminal.
- **Secure API Key Management**: Centralized management of API keys for different providers.
- **Reporting**: The result exported to excel automatically

## Installation

### Option 1: Quick Install (Recommended)

```bash
# Install latest version (user mode, auto PATH setup, no sudo required)
curl -fsSL https://raw.githubusercontent.com/Asyasyarif/openjudges/main/scripts/install.sh | bash

# Install specific version
curl -fsSL https://raw.githubusercontent.com/Asyasyarif/openjudges/main/scripts/install.sh | bash -s -- --version v1.2.0

# System-wide install (requires sudo)
sudo ./install.sh --system
```

### Option 2: Go Install

```bash
go install github.com/asyasyarif/openjudges@latest
```

### Supported Platforms

- **Linux**: amd64, arm64, armv7, armv6
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64, 386
- **FreeBSD**: amd64, arm64
- **NetBSD/OpenBSD**: amd64

### Auto-Update

```bash
# Check for and install updates
openjudges update

# Check only without installing
openjudges update --check-only

# Include pre-release versions
openjudges update --prerelease
```

## Quick Start

1.  **Initialize**:
    Run `openjudges` to start the interactive main menu.

2.  **Configure API Keys**:
    Use `openjudges apikeys` to set your LLM provider API keys.
    ```bash
    openjudges apikeys --set openai --key sk-...
    ```

3.  **Create a Judge**:
    Use the interactive wizard to create a new judge:
    ```bash
    openjudges create
    ```
    Or use flags:
    ```bash
    openjudges create --name "MyJudge" --provider openai --model gpt-4o --dataset datasets/test.csv --result results/out.json
    ```

4.  **Run Evaluations**:
    Run your configured judges:
    ```bash
    openjudges run --all
    ```
    Or select specific judges interactively via the CLI menu.

5.  **View Results**:
    Browse and analyze evaluation results:
    ```bash
    openjudges results
    ```

## Development

-   **Build**: `go build`
-   **Run**: `./openjudges`

## License

MIT
