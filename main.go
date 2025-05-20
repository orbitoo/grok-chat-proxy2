package main

import (
	"context"
	"flag"
	"fmt"
	"grok-chat-proxy2/client"
	"grok-chat-proxy2/server"
	"grok-chat-proxy2/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("hello world")
	var cookiesFlag bool
	flag.BoolVar(&cookiesFlag, "s", false, "Use `cookies` file to start sessions and login automatically")
	var headlessFlag bool
	flag.BoolVar(&headlessFlag, "h", false, "Run in headless mode, you will not see the browser (in this case, wait flag is ignored)")
	var token string
	flag.StringVar(&token, "i", "", "Identify token (api key)")
	var sessionNumber int
	flag.IntVar(&sessionNumber, "n", 0, "Number of sessions to create")
	var privateFlag bool
	flag.BoolVar(&privateFlag, "p", false, "Use private mode")
	var port int
	flag.IntVar(&port, "port", 9867, "Port to listen on")
	flag.Parse()
	var sm *client.SessionManager
	if cookiesFlag {
		cookies, err := utils.ReadCookies()
		if err != nil {
			log.Fatalf("Failed to read cookies: %v", err)
		}
		sm = client.NewSessionManagerWithCookie(cookies, headlessFlag, privateFlag)
	} else if sessionNumber > 0 {
		sm = client.NewSessionManagerN(sessionNumber, headlessFlag, privateFlag)
	} else {
		sm = client.NewSessionManager(headlessFlag, privateFlag)
	}
	defer sm.Close()
	grokAPI := func(prompt string, think bool, responseChan chan string) (context.CancelFunc, error) {
		return sm.SendMessage(&prompt, nil, think, responseChan)
	}
	grokAPIUsingFile := func(prompt string, filename string, think bool, responseChan chan string) (context.CancelFunc, error) {
		return sm.SendMessage(&prompt, &filename, think, responseChan)
	}
	server.ConfigureGrokAPI(grokAPI, grokAPIUsingFile)
	server.ConfigureExpectedAPIKey(token)
	mux := http.NewServeMux()
	chatCompletionHandler := http.HandlerFunc(server.ChatCompletionHandler)
	listModelsHandler := http.HandlerFunc(server.ListModelsHandler)
	mux.Handle("/v1/chat/completions", server.NeedAuthorization(chatCompletionHandler))
	mux.Handle("/v1/models", server.NeedAuthorization(listModelsHandler))
	log.Printf("Starting server on port %d...\n", port)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		if sig == os.Interrupt {
			fmt.Println("Received interrupt signal, exiting...")
			sm.Close()
			os.Exit(0)
		}
	}()
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
}
