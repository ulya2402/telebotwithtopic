package models

type TelegramResponse struct {
	Ok     bool            `json:"ok"`
	Result []Update        `json:"result"`
}

type Update struct {
	UpdateID      int            `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID       int             `json:"message_id"`
	MessageThreadID int             `json:"message_thread_id"`
	From            *User           `json:"from"`
	Chat            *Chat           `json:"chat"`
	Text            string          `json:"text"`
	IsTopicMessage  bool            `json:"is_topic_message"`
	ReplyToMessage  *Message        `json:"reply_to_message"` // Added for reply detection
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

type User struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type Chat struct {
	ID               int64  `json:"id"`
	Type             string `json:"type"` 
	HasTopicsEnabled bool   `json:"has_topics_enabled"`
}

type SendMessageRequest struct {
	ChatID          int64       `json:"chat_id"`
	MessageThreadID int         `json:"message_thread_id,omitempty"`
	Text            string      `json:"text"`
	ParseMode       string      `json:"parse_mode,omitempty"`
	ReplyToMessageID int         `json:"reply_to_message_id,omitempty"` // Added this
	ReplyMarkup     interface{} `json:"reply_markup,omitempty"`
}

type SendChatActionRequest struct {
	ChatID          int64  `json:"chat_id"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
	Action          string `json:"action"`
}

type SendMessageDraftRequest struct {
	ChatID          int64  `json:"chat_id"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
	DraftID         string `json:"draft_id"`
	Text            string `json:"text"`
	ParseMode       string `json:"parse_mode,omitempty"`
	ReplyToMessageID int    `json:"reply_to_message_id,omitempty"` // Added this
}

type EditForumTopicRequest struct {
	ChatID          int64  `json:"chat_id"`
	MessageThreadID int    `json:"message_thread_id"`
	Name            string `json:"name"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

