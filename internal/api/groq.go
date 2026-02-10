package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"telechatbot/internal/models"
	"time"
)

const groqURL = "https://api.groq.com/openai/v1/chat/completions"

type GroqClient struct {
	ApiKeys      []string
	CurrentKeyId int
	Model        string
	mu           sync.Mutex
}

func NewGroqClient(apiKeys []string, model string) *GroqClient {
	return &GroqClient{
		ApiKeys:      apiKeys,
		CurrentKeyId: 0,
		Model:        model,
	}
}

func (g *GroqClient) getCurrentKey() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.ApiKeys) == 0 {
		return ""
	}
	return g.ApiKeys[g.CurrentKeyId]
}

func (g *GroqClient) rotateKey() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.ApiKeys) <= 1 {
		return
	}
	g.CurrentKeyId = (g.CurrentKeyId + 1) % len(g.ApiKeys)
	log.Printf("[INFO] Switched to API Key index: %d", g.CurrentKeyId)
}

func (g *GroqClient) SendChat(messages []models.GroqMessage) (string, string, error) {
	maxRetries := len(g.ApiKeys)
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		content, reasoning, err := g.attemptRequest(messages)
		if err == nil {
			return content, reasoning, nil
		}

		lastErr = err
		log.Printf("[WARN] API Key failed (Attempt %d/%d): %v", i+1, maxRetries, err)

		// Rotate key and try again immediately
		g.rotateKey()
	}

	return "", "", fmt.Errorf("all api keys exhausted, last error: %v", lastErr)
}

func (g *GroqClient) attemptRequest(messages []models.GroqMessage) (string, string, error) {
	client := &http.Client{Timeout: 120 * time.Second}

	reqBody := models.GroqChatRequest{
		Model:    g.Model,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequest("POST", groqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", err
	}

	currentKey := g.getCurrentKey()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", currentKey))

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		// Check for Rate Limit (429) or Unauthorized (401) to trigger rotation
		if resp.StatusCode == 429 || resp.StatusCode == 401 {
			return "", "", fmt.Errorf("api error %d (triggering rotation): %s", resp.StatusCode, string(bodyBytes))
		}
		return "", "", fmt.Errorf("groq api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var groqResp models.GroqChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", "", err
	}

	if len(groqResp.Choices) == 0 {
		return "", "", fmt.Errorf("groq returned no choices")
	}

	return groqResp.Choices[0].Message.Content, groqResp.Choices[0].Message.Reasoning, nil
}