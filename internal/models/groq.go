package models

// Request structure for Groq/OpenAI compatible APIs
type GroqChatRequest struct {
	Model    string        `json:"model"`
	Messages []GroqMessage `json:"messages"`
}



type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
}

// Response structure
type GroqChatResponse struct {
	ID      string       `json:"id"`
	Choices []GroqChoice `json:"choices"`
}

type GroqChoice struct {
	Index        int         `json:"index"`
	Message      GroqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}