package utils

import (
	"encoding/json"
	"time"
)

type grokChunk struct {
	Result grokResult `json:"result"`
}

type grokResult struct {
	Response json.RawMessage `json:"response"`
}

type grokResponse struct {
	Token      string `json:"token"`
	IsThinking bool   `json:"isThinking"`
	IsSoftStop bool   `json:"isSoftStop"`
	ResponseId string `json:"responseId"`
}

func ParseGrokResponse(data string) (*grokResponse, error) {
	var chunk grokChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, err
	}
	if chunk.Result.Response == nil {
		return nil, nil
	}
	var finalResponse grokResponse
	if err := json.Unmarshal(chunk.Result.Response, &finalResponse); err != nil {
		return nil, err
	}
	return &finalResponse, nil
}

type openAIStreamChoiceDelta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

type openAIStreamChoice struct {
	Index        int                     `json:"index"`
	Delta        openAIStreamChoiceDelta `json:"delta"`
	FinishReason *string                 `json:"finish_reason,omitempty"`
}

type OpenAIStreamingResponseChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openAIStreamChoice `json:"choices"`
}

type OpenAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type OpenAIModelList struct {
	Object string        `json:"object"`
	Data   []OpenAIModel `json:"data"`
}

func BuildChunkStart(delta string, requestId string, model string) *OpenAIStreamingResponseChunk {
	choices := []openAIStreamChoice{
		{
			Index: 0,
			Delta: openAIStreamChoiceDelta{
				Content: delta,
				Role:    "assistant",
			},
			FinishReason: nil,
		},
	}
	return &OpenAIStreamingResponseChunk{
		ID:      requestId,
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   model,
		Choices: choices,
	}
}

func BuildChunk(delta string, requestId string, model string) *OpenAIStreamingResponseChunk {
	choices := []openAIStreamChoice{
		{
			Index: 0,
			Delta: openAIStreamChoiceDelta{
				Content: delta,
			},
			FinishReason: nil,
		},
	}
	return &OpenAIStreamingResponseChunk{
		ID:      requestId,
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   model,
		Choices: choices,
	}
}

func BuildChunkFinish(requestID string, model string) *OpenAIStreamingResponseChunk {
	finishReason := "stop"
	choices := []openAIStreamChoice{
		{
			Index: 0,
			Delta: openAIStreamChoiceDelta{
				Content: "",
			},
			FinishReason: &finishReason,
		},
	}
	return &OpenAIStreamingResponseChunk{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: 0,
		Model:   model,
		Choices: choices,
	}
}

func ModelList(models []string) *OpenAIModelList {
	modelList := make([]OpenAIModel, len(models))
	for i, model := range models {
		modelList[i] = OpenAIModel{
			ID:      model,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "elbert",
		}
	}
	return &OpenAIModelList{
		Object: "list",
		Data:   modelList,
	}
}
