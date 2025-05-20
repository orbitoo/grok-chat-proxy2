package client

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type SessionManager struct {
	sessions      map[int]*Session
	nextAvailable chan *Session
	private       bool
}

func NewSessionManager(headless bool, private bool) *SessionManager {
	sessions := make(map[int]*Session)
	files, err := os.ReadDir("./userdata")
	if err != nil {
		log.Printf("Failed to read userdata directory: %v", err)
		log.Fatal("This means you should use `-n <number>` to start <number> sessions")
	}
	ch := make(chan *Session)
	wg := sync.WaitGroup{}
	n := 0
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			id, err := strconv.Atoi(name)
			if err != nil {
				log.Printf("Failed to convert directory name to session index: %v", err)
				log.Fatal("Please manually delete the invalid directory under ./userdata")
			}
			wg.Add(1)
			n++
			go func(id int) {
				defer wg.Done()
				session, err := StartSession(id, headless)
				if err != nil {
					log.Printf("Failed to start session %d: %v", id, err)
					return
				}
				ch <- session
			}(id)
		}
	}
	nextAvailable := make(chan *Session, n)
	go func() {
		wg.Wait()
		close(ch)
	}()
	for session := range ch {
		sessions[session.id] = session
		nextAvailable <- session
	}
	return &SessionManager{sessions: sessions, nextAvailable: nextAvailable, private: private}
}

func NewSessionManagerN(n int, headless bool, private bool) *SessionManager {
	sessions := make(map[int]*Session)
	ch := make(chan *Session)
	wg := sync.WaitGroup{}
	wg.Add(n)
	nextAvailable := make(chan *Session, n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			session, err := StartSession(i, headless)
			if err != nil {
				log.Printf("Failed to start session %d: %v", i, err)
				return
			}
			ch <- session
		}(i)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for session := range ch {
		sessions[session.id] = session
		nextAvailable <- session
	}
	return &SessionManager{sessions: sessions, nextAvailable: nextAvailable, private: private}
}

func NewSessionManagerWithCookie(cookieList []string, headless bool, private bool) *SessionManager {
	sessions := make(map[int]*Session)
	ch := make(chan *Session)
	wg := sync.WaitGroup{}
	n := len(cookieList)
	wg.Add(n)
	nextAvailable := make(chan *Session, n)
	for i, cookieString := range cookieList {
		go func(i int, cookieString string) {
			defer wg.Done()
			session, err := StartSessionWithCookie(i, cookieString, headless)
			if err != nil {
				log.Printf("Failed to start session %d: %v", i, err)
				return
			}
			ch <- session
		}(i, cookieString)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for session := range ch {
		sessions[session.id] = session
		nextAvailable <- session
	}
	return &SessionManager{sessions: sessions, nextAvailable: nextAvailable, private: private}
}

func (sm *SessionManager) SendMessage(prompt *string, filename *string, think bool, responseChan chan string) (context.CancelFunc, error) {
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	var session *Session
	select {
	case <-timer.C:
		errMsg := "timeout waiting for available session"
		log.Println(errMsg)
		close(responseChan)
		return nil, errors.New(errMsg)
	case session = <-sm.nextAvailable:
		if session == nil {
			errMsg := "no available session"
			log.Println(errMsg)
			close(responseChan)
			return nil, errors.New(errMsg)
		}
	}
	listenCtx, cancelListen := context.WithCancel(*session.ctx)
	go func() {
		err := session.SendMessage(prompt, filename, sm.private, think, responseChan, listenCtx, cancelListen)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
			sm.nextAvailable <- session
			return
		} else {
			sm.nextAvailable <- session
		}
	}()
	return cancelListen, nil
}

func (sm *SessionManager) Close() {
	for _, session := range sm.sessions {
		session.Close()
	}
}
