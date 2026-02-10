package handlers

import (
	"strings"
	"telechatbot/internal/models"
)

func ShouldProcessMessage(msg *models.Message, botUsername string) (bool, string) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return false, ""
	}

	if msg.Chat.Type == "private" {
		return true, text
	}

	// [Pembaruan 1] Command dengan argumen (contoh: /ask pertanyaan)
	// Ini tetap dibersihkan agar AI hanya menerima pertanyaannya
	if strings.HasPrefix(text, "/ask") {
		return true, cleanTrigger(text, "/ask")
	}
	if strings.HasPrefix(text, "/ai") {
		return true, cleanTrigger(text, "/ai")
	}

	// [Pembaruan 2] Command TANPA argumen (Action Commands)
	// Kita harus mengembalikan string command-nya agar tidak dianggap pesan kosong oleh dispatcher
	if strings.HasPrefix(text, "/newchat") {
		return true, "/newchat"
	}
	if strings.HasPrefix(text, "/lang") {
		return true, "/lang"
	}

	// B. Check for Mention Trigger (@BotName)
	if botUsername != "" && strings.Contains(text, "@"+botUsername) {
		cleaned := strings.ReplaceAll(text, "@"+botUsername, "")
		return true, strings.TrimSpace(cleaned)
	}

	// C. Check for Reply Trigger
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		if msg.ReplyToMessage.From.Username == botUsername {
			return true, text
		}
	}

	return false, ""
}

// cleanTrigger removes the command prefix and extra spaces
func cleanTrigger(text, prefix string) string {
	cleaned := strings.TrimPrefix(text, prefix)
	return strings.TrimSpace(cleaned)
}