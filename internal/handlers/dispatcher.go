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
	} else if update.InlineQuery != nil {
		// [BARU] User sedang mengetik @bot ...
		d.handleInlineQuery(update.InlineQuery)
	} else if update.ChosenInlineResult != nil {
		// [BARU] User SUDAH mengirim pesan inline
		go d.handleChosenInlineResult(update.ChosenInlineResult)
	}
}

func (d *Dispatcher) handleInlineQuery(iq *models.InlineQuery) {
	if iq.Query == "" {
		return
	}

	// [PERBAIKAN] Tambahkan tombol dummy agar Telegram men-generate inline_message_id
	// Tanpa tombol ini, kita TIDAK BISA mengedit pesan tersebut nanti.
	loadingKeyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "⏳ Waiting for AI...", CallbackData: "noop"},
			},
		},
	}

	article := models.InlineQueryResult{
		Type:  "article",
		ID:    iq.Query,
		Title: "Tanya AI: " + iq.Query,
		Description: "Klik untuk mengirim dan memproses jawaban",
		InputMessageContent: models.InputMessageContent{
			MessageText: fmt.Sprintf("⏳ *Sedang berpikir...*\n\nQuery: _%s_", iq.Query),
			ParseMode:   "Markdown",
		},
		ReplyMarkup: loadingKeyboard, // <--- Masukkan keyboard di sini
	}

	d.Bot.AnswerInlineQuery(iq.ID, []models.InlineQueryResult{article})
}

// 2. Saat user KLIK hasil tersebut -> Pesan terkirim -> Bot dapat notif ini
// Di sinilah kita panggil AI Groq dan EDIT pesan tadi.
func (d *Dispatcher) handleChosenInlineResult(cir *models.ChosenInlineResult) {
	log.Printf("Processing inline query: %s", cir.Query)

	prompt := cir.Query
	if prompt == "" {
		prompt = cir.ResultID
	}

	messages := []models.GroqMessage{
		{Role: "system", Content: d.SystemPrompt + "\n(Jawablah dengan ringkas, padat, dan to the point karena ini mode inline)"},
		{Role: "user", Content: prompt},
	}

	// 1. Panggil AI
	aiContent, _, err := d.AI.SendChat(messages) // Parameter ke-2 (reasoning) kita abaikan dengan "_"
	
	finalResponse := aiContent
	if err != nil {
		finalResponse = "⚠️ Gagal menghubungi AI."
	}

	// 2. BERSIHKAN THINKING
	// Kita gunakan helper yang sudah ada di dispatcher.go untuk membuang tag <think>...</think>
	// dan kita abaikan return value pertama (isi think-nya).
	_, cleanResponse := d.extractThinkContent(finalResponse)

	// 3. Format pesan akhir (Hanya Pertanyaan + Jawaban Bersih)
	formattedText := fmt.Sprintf(cleanResponse)

	// 4. Edit pesan
	err = d.Bot.EditMessageText(0, 0, cir.InlineMessageID, formattedText)
	if err != nil {
		log.Printf("Failed to edit inline message: %v", err)
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
	msgID := msg.MessageID
	threadID := 0

	if msg.IsTopicMessage || msg.MessageThreadID != 0 {
		threadID = msg.MessageThreadID
	}

	userLang := d.DB.GetUserLanguage(userID)

	if strings.HasPrefix(text, "/newchat") {
		err := d.DB.ClearHistory(chatID, threadID)
		if err != nil {
			log.Printf("Failed to clear history: %v", err)
			d.Bot.SendMessage(chatID, threadID, msgID, "Failed to reset chat context.", nil)
			return
		}
		resetText := "🧹 Chat context has been reset. I have forgotten our previous conversation in this topic."
		if userLang == "id" {
			resetText = "🧹 Konteks obrolan telah direset. Aku sudah melupakan percakapan kita sebelumnya di topik ini."
		}
		d.Bot.SendMessage(chatID, threadID, msgID, resetText, nil)
		return
	}

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

	searchInstruction := " \n\nIMPORTANT: If you search the web, ALWAYS provide citations/sources as Markdown hyperlinks like this: [Title](URL). Do not use bare URLs or [1] format."
	finalSystemPrompt := d.SystemPrompt + searchInstruction

	var messages []models.GroqMessage
	messages = append(messages, models.GroqMessage{Role: "system", Content: finalSystemPrompt})

	for _, h := range history {
		role := "user"
		if h.Role == "AI" {
			role = "assistant"
		}
		// [Pembaruan 1] Filter pesan kosong.
		// Jika ada history kosong/spasi doang di database, JANGAN kirim ke AI.
		// Ini mencegah AI bingung dan mengulang pesan lama.
		if strings.TrimSpace(h.Content) != "" {
			messages = append(messages, models.GroqMessage{Role: role, Content: h.Content})
		}
	}

	messages = append(messages, models.GroqMessage{Role: "user", Content: text})

	aiContent, aiReasoning, err := d.AI.SendChat(messages)

	typingStop <- true
	close(typingStop)

	if err != nil {
		log.Printf("Error fetching AI response: %v", err)
		d.Bot.SendMessage(chatID, threadID, msgID, "Error connecting to AI service (Groq).", nil)
		return
	}

	extractedThink, cleanBody := d.extractThinkContent(aiContent)

	finalThink := aiReasoning
	if finalThink == "" {
		finalThink = extractedThink
	}

	finalResponse := cleanBody
	safeResponse := strings.ReplaceAll(finalResponse, "**", "*")

	if finalThink != "" && msg.Chat.Type != "private" {
		draftID := fmt.Sprintf("%d", time.Now().UnixNano())
		thoughtDisplay := fmt.Sprintf("🧠 %s...", finalThink)

		d.Bot.SendMessageDraft(chatID, threadID, msgID, draftID, thoughtDisplay)

		delay := time.Duration(len(finalThink)/50) * time.Second
		if delay < 1*time.Second {
			delay = 1 * time.Second
		}
		if delay > 3*time.Second {
			delay = 3 * time.Second
		}
		time.Sleep(delay)
	}

	var replyMarkup interface{}
	if msg.Chat.Type != "private" {
		replyMarkup = models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "Close ❌", CallbackData: "close_msg"},
				},
			},
		}
	}

	err = d.Bot.SendMessage(chatID, threadID, msgID, safeResponse, replyMarkup)
	if err != nil {
		log.Printf("Markdown send failed, trying raw: %v", err)
		d.Bot.SendMessage(chatID, threadID, msgID, finalResponse, replyMarkup)
	}

	if strings.TrimSpace(text) != "" {
		errUser := d.DB.AddHistory(chatID, threadID, "User", text)
		if errUser != nil {
			log.Printf("[ERROR] Failed to save User message to DB: %v", errUser)
		}
	}

	if strings.TrimSpace(finalResponse) != "" {
		errAI := d.DB.AddHistory(chatID, threadID, "AI", finalResponse)
		if errAI != nil {
			log.Printf("[ERROR] Failed to save AI response to DB: %v", errAI)
		} else {
			log.Printf("[DEBUG] Saved AI response to DB.")
		}
	}

	if isNewTopic && threadID != 0 && msg.Chat.Type == "private" {
		go d.generateAndSetTopicTitle(chatID, threadID, finalResponse)
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

	title, _, err := d.AI.SendChat(msgs)
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

	msgID := cb.Message.MessageID

	d.Bot.AnswerCallbackQuery(cb.ID)

	if cb.Data == "close_msg" {
		username := cb.From.Username
		if username == "" {
			username = cb.From.FirstName
		}
		
		closedText := fmt.Sprintf("_Response closed by @%s_", username)
		
		// PERBAIKAN: Tambahkan string kosong "" sebagai parameter ke-3 (inlineMessageID)
		err := d.Bot.EditMessageText(chatID, msgID, "", closedText) 
		
		if err != nil {
			log.Printf("Error closing message: %v", err)
		}
		return
	}

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