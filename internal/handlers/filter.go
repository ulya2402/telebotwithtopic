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

	if strings.HasPrefix(text, "/ask") {
		return true, cleanTrigger(text, "/ask")
	}
	if strings.HasPrefix(text, "/ai") {
		return true, cleanTrigger(text, "/ai")
	}

	// B. Check for Mention Trigger (@BotName)
	// We handle cases like "@BotName hello" or "hello @BotName"
	if botUsername != "" && strings.Contains(text, "@"+botUsername) {
		// Clean the username from text so AI doesn't read it
		cleaned := strings.ReplaceAll(text, "@"+botUsername, "")
		return true, strings.TrimSpace(cleaned)
	}

	// C. Check for Reply Trigger
	// If the user replies to a message sent by THIS bot
	if msg.ReplyToMessage != nil && msg.ReplyToMessage.From != nil {
		if msg.ReplyToMessage.From.Username == botUsername {
			return true, text
		}
	}

	// Default: Do not respond in groups if not triggered
	return false, ""
}

// cleanTrigger removes the command prefix and extra spaces
func cleanTrigger(text, prefix string) string {
	cleaned := strings.TrimPrefix(text, prefix)
	return strings.TrimSpace(cleaned)
}