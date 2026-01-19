package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Provider string

const (
	OpenAI = "openai"
	Ollama = "ollama"
)

type Config struct {
	Model    string   `json:"model"`
	Provider Provider `json:"provider"` // "ollama" or "openai"
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const configFileName = ".config/ai-cli.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var outputFile string
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			if i+1 >= len(args) {
				return fmt.Errorf("-o flag requires a filename argument")
			}
			outputFile = args[i+1]
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	if len(args) > 0 {
		switch args[0] {
		case "set-model":
			return setModelCommand()
		case "--help", "-h", "help":
			return printHelp()
		default:
			if err := ensureConfigExists(); err != nil {
				return err
			}
			prompt := strings.Join(args, " ")

			// If there's piped input, append it to the prompt
			if isPiped() {
				input, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read piped input: %w", err)
				}
				prompt = prompt + "\n\n" + strings.TrimSpace(string(input))
			}

			output, err := executePrompt(prompt)
			if err != nil {
				return err
			}
			return writeOutput(output, outputFile)
		}
	}

	if isPiped() {
		path := getConfigPath()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("not initialized: run once in interactive mode to configure")
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read piped input: %w", err)
		}
		output, err := executePrompt(strings.TrimSpace(string(input)))
		if err != nil {
			return err
		}
		return writeOutput(output, outputFile)
	}

	// interactive mode
	if err := ensureConfigExists(); err != nil {
		return err
	}
	fmt.Print("Enter your prompt: ")
	reader := bufio.NewReader(os.Stdin)
	prompt, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	output, err := executePrompt(strings.TrimSpace(prompt))
	if err != nil {
		return err
	}
	return writeOutput(output, outputFile)
}

func ensureConfigExists() error {
	path := getConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No configuration found. Running initial setup...")
		return initCommand()
	}
	return nil
}

func isOllamaInstalled() bool {
	_, err := exec.LookPath("ollama")
	return err == nil
}

func hasOpenAIToken() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

func isPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, configFileName)
}

func loadConfig() (*Config, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func saveConfig(config *Config) error {
	path := getConfigPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func getInstalledModels() ([]string, error) {
	cmd := exec.Command("ollama", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var models []string

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			models = append(models, fields[0])
		}
	}

	return models, nil
}

func getOpenAIModels() []string {
	return []string{
		"gpt-5-nano",
		"gpt-5-mini",
		"gpt-5.2",
	}
}

func getAllAvailableModels() (map[string][]string, error) {
	available := make(map[string][]string)

	if isOllamaInstalled() {
		ollamaModels, err := getInstalledModels()
		if err == nil && len(ollamaModels) > 0 {
			available["ollama"] = ollamaModels
		}
	}

	if hasOpenAIToken() {
		available["openai"] = getOpenAIModels()
	}

	return available, nil
}

func initCommand() error {
	available, err := getAllAvailableModels()
	if err != nil {
		return err
	}

	if len(available) == 0 {
		fmt.Println("No models available.")
		fmt.Println("Please either:")
		fmt.Println("  1. Install ollama and pull a model (e.g., 'ollama pull llama3.2')")
		fmt.Println("  2. Set OPENAI_API_KEY environment variable")
		return nil
	}

	// Build a flat list of models with their providers
	type ModelOption struct {
		Provider Provider
		Model    string
	}
	var options []ModelOption

	if models, ok := available["ollama"]; ok {
		for _, model := range models {
			options = append(options, ModelOption{Provider: Ollama, Model: model})
		}
	}
	if models, ok := available["openai"]; ok {
		for _, model := range models {
			options = append(options, ModelOption{Provider: OpenAI, Model: model})
		}
	}

	fmt.Println("Available models:")
	for i, opt := range options {
		fmt.Printf("%d. [%s] %s\n", i+1, opt.Provider, opt.Model)
	}
	fmt.Printf("Select a model (1-%d) [1]: ", len(options))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	if input == "" {
		choice = 1
	} else {
		fmt.Sscanf(input, "%d", &choice)
		if choice < 1 || choice > len(options) {
			return fmt.Errorf("invalid choice")
		}
	}

	selected := options[choice-1]
	fmt.Printf("Selected: [%s] %s\n", selected.Provider, selected.Model)

	config := &Config{
		Model:    selected.Model,
		Provider: selected.Provider,
	}
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Configuration saved successfully!")
	return nil
}

func setModelCommand() error {
	available, err := getAllAvailableModels()
	if err != nil {
		return err
	}

	if len(available) == 0 {
		return fmt.Errorf("no models available")
	}

	type ModelOption struct {
		Provider Provider
		Model    string
	}
	var options []ModelOption

	if models, ok := available["ollama"]; ok {
		for _, model := range models {
			options = append(options, ModelOption{Provider: Ollama, Model: model})
		}
	}
	if models, ok := available["openai"]; ok {
		for _, model := range models {
			options = append(options, ModelOption{Provider: OpenAI, Model: model})
		}
	}

	fmt.Println("Available models:")
	for i, opt := range options {
		fmt.Printf("%d. [%s] %s\n", i+1, opt.Provider, opt.Model)
	}
	fmt.Printf("Select a model (1-%d): ", len(options))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	fmt.Sscanf(input, "%d", &choice)
	if choice < 1 || choice > len(options) {
		return fmt.Errorf("invalid choice")
	}

	selected := options[choice-1]
	config := &Config{
		Model:    selected.Model,
		Provider: selected.Provider,
	}
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Model changed to: [%s] %s", selected.Provider, selected.Model)
	return nil
}

func printHelp() error {
	config, err := loadConfig()
	currentModel := "not configured"
	if err == nil {
		currentModel = fmt.Sprintf("[%s] %s", config.Provider, config.Model)
	}

	fmt.Printf(`AI CLI - Ollama & OpenAI Command Line Interface

Current model: %s

Usage:
  ai-cli                        Interactive mode (prompts for input)
  ai-cli "your prompt"          Execute with direct prompt
  ai-cli -o file.txt "prompt"   Execute and save output to file
  echo "prompt" | ai-cli        Execute with piped input
  echo "prompt" | ai-cli -o out.txt  Save piped output to file
  ai-cli set-model              Change the model
  ai-cli --help                 Show this help message

Examples:
  ai-cli "What is the capital of France?"
  ai-cli -o answer.txt "Explain quantum computing"
  echo "Explain quantum computing" | ai-cli -o output.txt

Environment Variables:
  OPENAI_API_KEY                OpenAI API key (enables OpenAI models)

Note: Configuration is created automatically on first run.
`, currentModel)
	return nil
}

func isModelInstalled(model string) (bool, error) {
	models, err := getInstalledModels()
	if err != nil {
		return false, err
	}

	return slices.Contains(models, model), nil
}

func writeOutput(output string, outputFile string) error {
	if outputFile == "" {
		fmt.Print(output)
		return nil
	}

	if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}

func executePrompt(prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("empty prompt")
	}

	config, err := loadConfig()
	if err != nil {
		return "", err
	}

	switch config.Provider {
	case "ollama":
		return executeOllama(config.Model, prompt)
	case "openai":
		return executeOpenAI(config.Model, prompt)
	default:
		return "", fmt.Errorf("unknown provider: %s", config.Provider)
	}
}

func executeOllama(model, prompt string) (string, error) {
	installed, err := isModelInstalled(model)
	if err != nil {
		return "", err
	}
	if !installed {
		return "", fmt.Errorf("configured model '%s' is not installed. Please run 'set-model'", model)
	}

	cmd := exec.Command("ollama", "run", model, prompt)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute prompt: %w", err)
	}

	return string(output), nil
}

func executeOpenAI(model, prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	reqBody := OpenAIRequest{
		Model: model,
		Messages: []OpenAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return openAIResp.Choices[0].Message.Content, nil
}
