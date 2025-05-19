package server

import (
	"context"
	"encoding/json"
	"fmt"
	"grok-chat-proxy2/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var callGrok func(prompt string, think bool, responseChan chan string) (context.CancelFunc, error)
var callGrokUsingFile func(prompt string, filename string, think bool, responseChan chan string) (context.CancelFunc, error)
var expectedAPIKey string
var MAX_PROMPT_LENGTH = 40000

func ChatCompletionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var request utils.OpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		errMsg := fmt.Sprintf("Failed to parse request body: %v", err)
		http.Error(w, errMsg, http.StatusBadRequest)
		log.Println(errMsg)
		return
	}
	defer r.Body.Close()
	if !request.Stream {
		errMsg := "Only support stream mode"
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		errMsg := "Streaming unsupported!"
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return
	}
	requestID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	modelName := request.Model
	// possibleModels := []string{"grok-3", "grok-3-think"}
	if modelName != "grok-3" && modelName != "grok-3-think" {
		errMsg := fmt.Sprintf("Unsupported model: %s", modelName)
		http.Error(w, errMsg, http.StatusBadRequest)
		log.Println(errMsg)
		return
	}

	prompt := utils.PromptHandler(request.Messages)
	cwd, err := os.Getwd()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get current working directory: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return
	}
	filepath := cwd + "/lastPrompt.txt"
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to open file: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
	}
	defer file.Close()
	_, err = file.WriteString(prompt)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to write to file: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
	}
	done := make(chan bool)
	responseChan := make(chan string, 20)
	var allocErr error
	var cancelFunc context.CancelFunc
	if len(prompt) > MAX_PROMPT_LENGTH {
		cancelFunc, allocErr = callGrokUsingFile("", filepath, modelName == "grok-3-think", responseChan)
	} else {
		cancelFunc, allocErr = callGrok(prompt, modelName == "grok-3-think", responseChan)
	}
	if allocErr != nil {
		errMsg := fmt.Sprintf("Failed to allocate session: %v", allocErr)
		http.Error(w, errMsg, http.StatusTooManyRequests)
		log.Println(errMsg)
		return
	}
	defer cancelFunc()

	go processStreamChunk(responseChan, requestID, modelName, w, flusher, done)
	<-done
}

func processStreamChunk(responseChan chan string, requestID string, model string, w http.ResponseWriter, flusher http.Flusher, done chan bool) {
	first := false
	for delta := range responseChan {
		if first {
			first = false
			chunk := utils.BuildChunkStart(delta, requestID, model)
			if err := sendChunk(w, flusher, chunk); err != nil {
				done <- false
				return
			}
		} else {
			chunk := utils.BuildChunk(delta, requestID, model)
			if err := sendChunk(w, flusher, chunk); err != nil {
				done <- false
				return
			}
		}
	}
	finishChunk := utils.BuildChunkFinish(requestID, model)
	if err := sendChunk(w, flusher, finishChunk); err != nil {
		done <- false
		return
	}
	if err := endStream(w, flusher); err != nil {
		log.Println(err)
	}
	log.Println("Finished sending response")
	done <- true
}

func sendChunk(w http.ResponseWriter, flusher http.Flusher, chunk *utils.OpenAIStreamingResponseChunk) error {
	chunkData, err := json.Marshal(chunk)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal chunk: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", chunkData)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to write chunk to response: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return err
	}
	flusher.Flush()
	return nil
}

func endStream(w http.ResponseWriter, flusher http.Flusher) error {
	_, err := fmt.Fprintf(w, "data: [DONE]\n\n")
	if err != nil {
		errMsg := fmt.Sprintf("Failed to write end stream to response: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return err
	}
	flusher.Flush()
	return nil
}

func ConfigureGrokAPI(apiFunc func(prompt string, think bool, responseChan chan string) (context.CancelFunc, error),
	apiFuncUsingFile func(prompt string, filename string, think bool, responseChan chan string) (context.CancelFunc, error)) {
	callGrok = apiFunc
	callGrokUsingFile = apiFuncUsingFile
}

func ConfigureExpectedAPIKey(apiKey string) {
	expectedAPIKey = apiKey
}

func ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	models := []string{"grok-3", "grok-3-think"}
	modelList := utils.ModelList(models)
	responseData, err := json.Marshal(modelList)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to marshal model list: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Println(errMsg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseData)
}

func NeedAuthorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedAPIKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("Auth: Missing Authorization header")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Printf("Auth: Invalid Authorization header format: %s", authHeader)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := parts[1]
		if token != expectedAPIKey {
			log.Printf("Auth: Invalid API key: %s", token)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
