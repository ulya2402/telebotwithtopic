package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"telechatbot/internal/api"
	"telechatbot/internal/bot"
	"telechatbot/internal/database"
	"telechatbot/internal/i18n"
	"telechatbot/internal/models"
	"time"
)

type Dispatcher struct {
	Bot          *bot.Client
	AI           *api.GroqClient
	DB           *database.DB
	Localizer    *i18n.Localizer
	SystemPrompt string
	BotUsername  string
}

func NewDispatcher(b *bot.Client, ai *api.GroqClient, db *database.DB, loc *i18n.Localizer, sysPrompt, botUsername string) *Dispatcher {
	return &Dispatcher{
		Bot:          b,
		AI:           ai,
		DB:           db,
		Localizer:    loc,
		SystemPrompt: sysPrompt,
		BotUsername:  botUsername,
	}
}

func (d *Dispatcher) HandleUpdate(update models.Update) {
	if update.Message != nil {
		d.handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		d.handleCallback(update.CallbackQuery)
	}
}

func (d *Dispatcher) extractThinkContent(raw string) (string, string) {
	reThink := regexp.MustCompile(`(?s)<think>(.*?)</think>`)
	match := reThink.FindStringSubmatch(raw)
	
	thinkContent := ""
	if len(match) > 1 {
		thinkContent = strings.TrimSpace(match[1])
	}

	cleanResponse := reThink.ReplaceAllString(raw, "")
	return thinkContent, strings.TrimSpace(cleanResponse)
}

func (d *Dispatcher) continuouslySendTyping(chatID int64, threadID int, stopChan chan bool) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	d.Bot.SendChatAction(chatID, threadID, "typing")

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			d.Bot.SendChatAction(chatID, threadID, "typing")
		}
	}
}

func (d *Dispatcher) handleMessage(msg *models.Message) {
	// Filter logic
	shouldRespond, cleanText := ShouldProcessMessage(msg, d.BotUsername)
	if !shouldRespond {
		return 
	}

	text := cleanText
	if text == "" {
		return
	}

	userID := msg.From.ID
	chatID := msg.Chat.ID
	msgID := msg.MessageID // ID pesan user untuk di-reply
	threadID := 0
	
	if msg.IsTopicMessage || msg.MessageThreadID != 0 {
		threadID = msg.MessageThreadID
	}

	userLang := d.DB.GetUserLanguage(userID)
	
	if strings.HasPrefix(text, "/start") {
		welcomeText := d.Localizer.Get(userLang, "welcome")
		d.Bot.SendMessage(chatID, threadID, 0, welcomeText, nil)
		return
	}
	if strings.HasPrefix(text, "/lang") {
		d.sendLanguageSelector(chatID, threadID, 0, userLang)
		return
	}

	history, _ := d.DB.GetHistory(chatID, threadID)
	isNewTopic := len(history) == 0

	typingStop := make(chan bool)
	go d.continuouslySendTyping(chatID, threadID, typingStop)


	var messages []models.GroqMessage
	messages = append(messages, models.GroqMessage{Role: "system", Content: d.SystemPrompt})

	for _, h := range history {
		role := "user"
		if h.Role == "AI" {
			role = "assistant"
		}
		messages = append(messages, models.GroqMessage{Role: role, Content: h.Content})
	}

	messages = append(messages, models.GroqMessage{Role: "user", Content: text})

	
	aiRawResponse, err := d.AI.SendChat(messages)
	
	typingStop <- true
	close(typingStop)

	if err != nil {
		log.Printf("Error fetching AI response: %v", err)
		d.Bot.SendMessage(chatID, threadID, msgID, "Error connecting to AI service (Groq).", nil)
		return
	}

	thinkContent, cleanResponse := d.extractThinkContent(aiRawResponse)
	safeResponse := strings.ReplaceAll(cleanResponse, "**", "*")

	if thinkContent != "" {
		draftID := fmt.Sprintf("%d", time.Now().UnixNano())
		thoughtDisplay := fmt.Sprintf("🧠 %s...", thinkContent)
		
		d.Bot.SendMessageDraft(chatID, threadID, msgID, draftID, thoughtDisplay)
		
		delay := time.Duration(len(thinkContent)/50) * time.Second
		if delay < 1*time.Second {
			delay = 1 * time.Second
		}
		if delay > 3*time.Second {
			delay = 3 * time.Second
		}
		time.Sleep(delay)
	}

	err = d.Bot.SendMessage(chatID, threadID, msgID, safeResponse, nil)
	if err != nil {
		log.Printf("Markdown send failed, trying raw: %v", err)
		d.Bot.SendMessage(chatID, threadID, msgID, cleanResponse, nil)
	}

	d.DB.AddHistory(chatID, threadID, "User", text)
	d.DB.AddHistory(chatID, threadID, "AI", cleanResponse)

	if isNewTopic && threadID != 0 && msg.Chat.Type == "private" {
		go d.generateAndSetTopicTitle(chatID, threadID, cleanResponse)
	}
}

func (d *Dispatcher) generateAndSetTopicTitle(chatID int64, threadID int, contextText string) {
	if len(contextText) > 500 {
		contextText = contextText[:500]
	}
	
	prompt := fmt.Sprintf("Buatkan judul topik maksimal 3 kata, sangat ringkas, tanpa simbol, tanpa tanda baca, berdasarkan teks ini: %s", contextText)
	
	msgs := []models.GroqMessage{
		{Role: "system", Content: d.SystemPrompt},
		{Role: "user", Content: prompt},
	}

	title, err := d.AI.SendChat(msgs)
	if err != nil {
		log.Printf("Failed to generate title: %v", err)
		return
	}

	_, cleanTitle := d.extractThinkContent(title)
	cleanTitle = strings.ReplaceAll(cleanTitle, "*", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, "\"", "")
	cleanTitle = strings.ReplaceAll(cleanTitle, ".", "")
	
	words := strings.Fields(cleanTitle)
	if len(words) > 3 {
		cleanTitle = strings.Join(words[:3], " ")
	}

	log.Printf("Renaming topic %d to: %s", threadID, cleanTitle)
	d.Bot.EditForumTopic(chatID, threadID, cleanTitle)
}

func (d *Dispatcher) handleCallback(cb *models.CallbackQuery) {
	userID := cb.From.ID
	chatID := cb.Message.Chat.ID
	threadID := cb.Message.MessageThreadID 

	d.Bot.AnswerCallbackQuery(cb.ID)

	if strings.HasPrefix(cb.Data, "set_lang_") {
		newLang := strings.TrimPrefix(cb.Data, "set_lang_")
		if err := d.DB.SetUserLanguage(userID, newLang); err != nil {
			log.Printf("Error setting language: %v", err)
			return
		}
		
		confirmText := d.Localizer.Get(newLang, "lang_set")
		d.Bot.SendMessage(chatID, threadID, 0, confirmText, nil)
	}
}

func (d *Dispatcher) sendLanguageSelector(chatID int64, threadID int, replyToID int, currentLang string) {
	text := d.Localizer.Get(currentLang, "choose_lang")
	
	keyboard := models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "🇬🇧 English", CallbackData: "set_lang_en"},
				{Text: "🇮🇩 Indonesia", CallbackData: "set_lang_id"},
			},
		},
	}

	d.Bot.SendMessage(chatID, threadID, replyToID, text, keyboard)
}