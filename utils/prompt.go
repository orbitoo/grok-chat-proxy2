package utils

import (
	"encoding/base64"
	"encoding/json"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	TopK        int       `json:"top_k,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

func ParseRequest(request string) (*OpenAIRequest, error) {
	var openAIRequest OpenAIRequest
	err := json.Unmarshal([]byte(request), &openAIRequest)
	if err != nil {
		return nil, err
	}
	return &openAIRequest, nil
}

func formatPrompt(msgs []Message, roleMap map[string]string) string {
	var prompt string
	for _, msg := range msgs {
		switch msg.Role {
		case "user":
			prompt += roleMap["user"] + ": " + msg.Content + "\n\n"
		case "assistant":
			prompt += roleMap["assistant"] + ": " + msg.Content + "\n\n"
		case "system":
			prompt += roleMap["system"] + ": " + msg.Content + "\n\n"
		}
	}
	return prompt
}

var roleMap = map[string]string{
	"user":      "human",
	"assistant": "assistant",
	"system":    "user",
}

func Base64Decode(input string) (*[]byte, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(string(input))
	if err != nil {
		return nil, err
	}
	return &decodedBytes, nil
}

func PromptHandler(msgs []Message) string {
	prompt := formatPrompt(msgs, roleMap)
	return prompt
}

func ConfigureRoleMap(userRole, assistantRole, systemRole string) {
	roleMap["user"] = userRole
	roleMap["assistant"] = assistantRole
	roleMap["system"] = systemRole
}
