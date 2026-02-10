package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"telechatbot/internal/models"
	"time"
)

type Client struct {
	Token      string
	HttpClient *http.Client
	BaseURL    string
}

func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HttpClient: &http.Client{Timeout: 30 * time.Second},
		BaseURL:    fmt.Sprintf("https://api.telegram.org/bot%s", token),
	}
}

func (c *Client) GetUpdates(offset int) ([]models.Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=60", c.BaseURL, offset)
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result models.TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Ok {
		return nil, fmt.Errorf("telegram api error: result not ok")
	}

	return result.Result, nil
}

func (c *Client) SendChatAction(chatID int64, threadID int, action string) error {
	reqBody := models.SendChatActionRequest{
		ChatID:          chatID,
		MessageThreadID: threadID,
		Action:          action,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/sendChatAction", c.BaseURL)
	_, err = c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	return err
}

func (c *Client) SendMessageDraft(chatID int64, threadID int, replyToMsgID int, draftID, text string) error {
	reqBody := models.SendMessageDraftRequest{
		ChatID:          chatID,
		MessageThreadID: threadID,
		DraftID:         draftID,
		Text:            text,
		ParseMode:       "Markdown",
		ReplyToMessageID: replyToMsgID, 
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/sendMessageDraft", c.BaseURL)
	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("draft status: %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) SendMessage(chatID int64, threadID int, replyToMsgID int, text string, replyMarkup interface{}) error {
	reqBody := models.SendMessageRequest{
		ChatID:          chatID,
		MessageThreadID: threadID,
		Text:            text,
		ParseMode:       "Markdown", 
		ReplyToMessageID: replyToMsgID, 
		ReplyMarkup:     replyMarkup,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/sendMessage", c.BaseURL)
	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to send message, status: %d", resp.StatusCode)
		return fmt.Errorf("failed to send message, status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) EditForumTopic(chatID int64, threadID int, name string) error {
	reqBody := models.EditForumTopicRequest{
		ChatID:          chatID,
		MessageThreadID: threadID,
		Name:            name,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/editForumTopic", c.BaseURL)
	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to edit topic, status: %d", resp.StatusCode)
		return fmt.Errorf("failed to edit topic, status: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) AnswerCallbackQuery(callbackID string) {
	url := fmt.Sprintf("%s/answerCallbackQuery?callback_query_id=%s", c.BaseURL, callbackID)
	c.HttpClient.Get(url)
}

func (c *Client) AnswerInlineQuery(queryID string, results []models.InlineQueryResult) error {
	reqBody := models.AnswerInlineQueryRequest{
		InlineQueryID: queryID,
		Results:       results,
		CacheTime:     0, // Set 0 agar development mudah, naikkan ke 300 nanti
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/answerInlineQuery", c.BaseURL)
	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to answer inline query, status: %d", resp.StatusCode)
		return fmt.Errorf("failed status: %d", resp.StatusCode)
	}
	return nil
}

// Update fungsi EditMessageText agar bisa pakai InlineMessageID
func (c *Client) EditMessageText(chatID int64, messageID int, inlineMessageID string, text string) error {
	// Logic: Jika inlineMessageID ada, chatID dan messageID akan otomatis diabaikan oleh JSON omitempty
	reqBody := models.EditMessageTextRequest{
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{}},
	}

	if inlineMessageID != "" {
		reqBody.InlineMessageID = inlineMessageID
	} else {
		reqBody.ChatID = chatID
		reqBody.MessageID = messageID
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/editMessageText", c.BaseURL)
	resp, err := c.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Abaikan error "message is not modified" atau error minor lainnya
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}

	return nil
}