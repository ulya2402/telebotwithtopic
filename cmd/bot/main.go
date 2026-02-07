package main

import (
	"log"
	"telechatbot/config"
	"telechatbot/internal/api"
	"telechatbot/internal/bot"
	"telechatbot/internal/database"
	"telechatbot/internal/handlers"
	"telechatbot/internal/i18n"
)

func main() {
	log.Println("Starting TeleChatBot with Group Support...")

	cfg := config.LoadConfig()

	db := database.InitDB(cfg.DatabaseFile)
	defer db.Conn.Close()

	loc := i18n.NewLocalizer()

	botClient := bot.NewClient(cfg.TelegramToken)

	aiClient := api.NewGroqClient(cfg.GroqApiKey, cfg.GroqModel)

	// Update: Pass cfg.BotUsername to the dispatcher
	d := handlers.NewDispatcher(botClient, aiClient, db, loc, cfg.SystemPrompt, cfg.BotUsername)

	log.Println("Bot is running. Waiting for updates...")

	offset := 0
	for {
		updates, err := botClient.GetUpdates(offset)
		if err != nil {
			log.Printf("Error getting updates: %v", err)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			
			go d.HandleUpdate(update)
		}
	}
}