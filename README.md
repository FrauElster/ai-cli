# AI CLI

A simple command-line interface for interacting with Ollama and OpenAI models.

## Prerequisites

- **For Ollama**: [Ollama](https://ollama.ai/) must be installed and running with at least one model pulled
- **For OpenAI**: Set the `OPENAI_API_KEY` environment variable
- Go 1.25.6 or later (for building from source)

## Installation

Install directly using Go:

```bash
go install github.com/frauelster/ai-cli@latest
```

Make sure `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

## Usage

### First Run

On first run, the CLI will automatically detect available models from both Ollama and OpenAI (if configured) and prompt you to select one:

```bash
ai-cli
```

### Interactive Mode

Run without arguments to enter interactive mode:

```bash
ai-cli
```

### Direct Prompt

Pass your prompt as an argument:

```bash
ai-cli "What is the capital of France?"
```

### Piped Input

Pipe input from other commands:

```bash
echo "Explain quantum computing" | ai-cli
```

### Combining Prompt with Piped Input

Provide both a prompt and piped data - the piped content will be appended to your prompt:

```bash
cat data.txt | ai-cli "summarize this:"
cat error.log | ai-cli "what are the main errors in this log?"
```

### Output to File

Use the `-o` flag to save output to a file:

```bash
ai-cli -o answer.txt "Explain quantum computing"
echo "What is AI?" | ai-cli -o output.txt
cat document.txt | ai-cli "summarize this:" -o summary.txt
```

### Change Model

Switch between available models:

```bash
ai-cli set-model
```

### Help

Display help information:

```bash
ai-cli --help
```

## Configuration

Configuration is stored in `~/.config/ai-cli.json` and is created automatically on first run. The configuration includes:
- Selected model name
- Provider (ollama or openai)

### Environment Variables

- `OPENAI_API_KEY`: Required for using OpenAI models

## Examples

```bash
# Simple question
ai-cli "What is the meaning of life?"

# Code explanation
ai-cli "Explain how binary search works"

# Save response to file
ai-cli -o explanation.txt "Explain machine learning in simple terms"

# Use with pipes (prompt only from stdin)
echo "Explain quantum computing" | ai-cli

# Combine prompt argument with piped data
cat document.txt | ai-cli "Summarize this text"
cat code.go | ai-cli "Review this code for bugs" -o review.txt

# Change model
ai-cli set-model

# Use OpenAI (if OPENAI_API_KEY is set)
export OPENAI_API_KEY=your-api-key-here
ai-cli set-model  # Select an OpenAI model
ai-cli "Explain neural networks"
```

## Supported Models

### Ollama
Any model installed via Ollama (e.g., llama3.2, mistral, codellama)

### OpenAI
- gpt-5-nano
- gpt-5-mini
- gpt-5.2

*Note: OpenAI models require a valid API key set in the `OPENAI_API_KEY` environment variable.*
