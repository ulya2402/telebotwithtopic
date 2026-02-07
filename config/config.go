package config

import (
	"bufio"
	"log"
	"os"
	"strings"
)

type Config struct {
	TelegramToken string
	DatabaseFile  string
	SystemPrompt  string
	GroqApiKey    string
	GroqModel     string
	BotUsername   string
}

func LoadConfig() *Config {
	file, err := os.Open(".env")
	if err != nil {
		log.Println("Warning: .env file not found, relying on system environment variables")
	} else {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				os.Setenv(key, value)
			}
		}
	}

	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabaseFile:  os.Getenv("DATABASE_FILE"),
		SystemPrompt:  os.Getenv("SYSTEM_PROMPT"),
		GroqApiKey:    os.Getenv("GROQ_API_KEY"),
		GroqModel:     os.Getenv("GROQ_MODEL"),
		BotUsername:   os.Getenv("BOT_USERNAME"),
	}

	if cfg.TelegramToken == "" {
		log.Fatal("Error: TELEGRAM_BOT_TOKEN is required")
	}
	if cfg.DatabaseFile == "" {
		cfg.DatabaseFile = "telechatbot.db"
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = "You are a helpful AI assistant."
	}
	if cfg.GroqApiKey == "" {
		log.Fatal("Error: GROQ_API_KEY is required in .env")
	}
	if cfg.GroqModel == "" {
		cfg.GroqModel = "qwen/qwen3-32b" // Fallback default
	}
	if cfg.BotUsername == "" {
		log.Println("Warning: BOT_USERNAME is not set in .env. Group mentions might not work perfectly.")
	}

	return cfg
}