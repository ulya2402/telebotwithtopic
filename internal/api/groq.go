package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"telechatbot/internal/models"
	"time"
)

const groqURL = "https://api.groq.com/openai/v1/chat/completions"

type GroqClient struct {
	ApiKey string
	Model  string
}

func NewGroqClient(apiKey, model string) *GroqClient {
	return &GroqClient{
		ApiKey: apiKey,
		Model:  model,
	}
}

func (g *GroqClient) SendChat(messages []models.GroqMessage) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}

	reqBody := models.GroqChatRequest{
		Model:    g.Model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", groqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.ApiKey))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("groq api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var groqResp models.GroqChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", err
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("groq returned no choices")
	}

	return groqResp.Choices[0].Message.Content, nil
}