package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type Config struct {
	Model string `json:"model"`
}

const configFileName = ".config/ai-cli.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if !isOllamaInstalled() {
		return fmt.Errorf("ollama is not installed or not in PATH. Please install ollama first")
	}

	var outputFile string
	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		if args[i] == "-o" {
			if i+1 >= len(args) {
				return fmt.Errorf("-o flag requires a filename argument")
			}
			outputFile = args[i+1]
			// Remove -o and its argument from args
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
			output, err := executePrompt(strings.Join(args, " "))
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

	// Skip header line and parse model names
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

func initCommand() error {
	models, err := getInstalledModels()
	if err != nil {
		return err
	}

	var selectedModel string

	if len(models) == 0 {
		fmt.Println("No models installed.")
		fmt.Println("Please install a model first, for example:")
		fmt.Println("  ollama pull llama3.2")
		return nil
	} else if len(models) == 1 {
		selectedModel = models[0]
		fmt.Printf("Using the only installed model: %s\n", selectedModel)
	} else {
		fmt.Println("Multiple models installed:")
		for i, model := range models {
			fmt.Printf("%d. %s\n", i+1, model)
		}
		fmt.Printf("Select a model (1-%d) [1]: ", len(models))

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			selectedModel = models[0]
		} else {
			var choice int
			fmt.Sscanf(input, "%d", &choice)
			if choice < 1 || choice > len(models) {
				return fmt.Errorf("invalid choice")
			}
			selectedModel = models[choice-1]
		}
		fmt.Printf("Selected model: %s\n", selectedModel)
	}

	config := &Config{Model: selectedModel}
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("Configuration saved successfully!")
	return nil
}

func setModelCommand() error {
	models, err := getInstalledModels()
	if err != nil {
		return err
	}

	if len(models) == 0 {
		return fmt.Errorf("no models installed")
	}

	fmt.Println("Available models:")
	for i, model := range models {
		fmt.Printf("%d. %s\n", i+1, model)
	}
	fmt.Printf("Select a model (1-%d): ", len(models))

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	var choice int
	fmt.Sscanf(input, "%d", &choice)
	if choice < 1 || choice > len(models) {
		return fmt.Errorf("invalid choice")
	}

	selectedModel := models[choice-1]
	config := &Config{Model: selectedModel}
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Model changed to: %s\n", selectedModel)
	return nil
}

func printHelp() error {
	config, err := loadConfig()
	currentModel := "not configured"
	if err == nil {
		currentModel = config.Model
	}

	fmt.Printf(`AI CLI - Ollama Command Line Interface

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

	// check if model is still installed
	installed, err := isModelInstalled(config.Model)
	if err != nil {
		return "", err
	}
	if !installed {
		return "", fmt.Errorf("configured model '%s' is not installed. Please run 'init' or 'set-model'", config.Model)
	}

	cmd := exec.Command("ollama", "run", config.Model, prompt)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute prompt: %w", err)
	}

	return string(output), nil
}
