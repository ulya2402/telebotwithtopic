package main

import (
	"log"
	"strings" // [Pembaruan] Import strings untuk memecah API Key
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

	// [Pembaruan] Logika Rotasi API Key
	// Kita memecah string dari .env (contoh: "key1,key2,key3") menjadi array/slice
	apiKeys := strings.Split(cfg.GroqApiKey, ",")
	for i := range apiKeys {
		apiKeys[i] = strings.TrimSpace(apiKeys[i]) // Hapus spasi jika ada
	}

	// Masukkan array apiKeys ke client, bukan cuma satu string
	aiClient := api.NewGroqClient(apiKeys, cfg.GroqModel)

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