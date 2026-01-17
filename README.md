# AI CLI

A simple command-line interface for interacting with Ollama models.

## Prerequisites

- [Ollama](https://ollama.ai/) must be installed and running
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

On first run, the CLI will automatically prompt you to select an installed Ollama model:

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

### Output to File

Use the `-o` flag to save output to a file:

```bash
ai-cli -o answer.txt "Explain quantum computing"
echo "What is AI?" | ai-cli -o output.txt
```

### Change Model

Switch between installed Ollama models:

```bash
ai-cli set-model
```

### Help

Display help information:

```bash
ai-cli --help
```

## Configuration

Configuration is stored in `~/.config/ai-cli.json` and is created automatically on first run.

## Examples

```bash
# Simple question
ai-cli "What is the meaning of life?"

# Code explanation
ai-cli "Explain how binary search works"

# Save response to file
ai-cli -o explanation.txt "Explain machine learning in simple terms"

# Use with pipes
cat document.txt | ai-cli "Summarize this text" -o summary.txt

# Change model
ai-cli set-model
```
