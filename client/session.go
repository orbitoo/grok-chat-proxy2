package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"grok-chat-proxy2/utils"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Session struct {
	ctx     *context.Context
	id      int
	cookies string
	release func()
	output  chan string
}

var TIMEOUT = 15 * time.Second

func StartSessionWithCookie(id int, cookieString string, headless bool) (*Session, error) {
	session, err := initSession(id, headless)
	session.cookies = cookieString
	if err != nil {
		log.Printf("Failed to initialize session %d: %v", id, err)
		return nil, err
	}
	err = session.setupCookies()
	if err != nil {
		log.Printf("Failed to setup cookies for session %d: %v", session.id, err)
		session.Close()
		return nil, err
	}
	err = session.jsInjection()
	if err != nil {
		log.Printf("Failed to inject JS for session %d: %v", session.id, err)
		session.Close()
		return nil, err
	}
	err = session.navigateToHomepage()
	if err != nil {
		log.Printf("Failed to navigate to homepage for session %d: %v", session.id, err)
		session.Close()
		return nil, err
	}

	return session, nil
}

func StartSession(id int, headless bool) (*Session, error) {
	session, err := initSession(id, headless)
	if err != nil {
		log.Printf("Failed to initialize session %d: %v", id, err)
		return nil, err
	}
	err = session.jsInjection()
	if err != nil {
		log.Printf("Failed to inject JS for session %d: %v", session.id, err)
		session.Close()
		return nil, err
	}
	err = session.navigateToHomepage()
	if err != nil {
		log.Printf("Failed to navigate to homepage for session %d: %v", session.id, err)
		session.Close()
		return nil, err
	}

	return session, nil
}

func initSession(id int, headless bool) (*Session, error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get current working directory for session %d: %v", id, err)
		return nil, err
	}
	userDataDir := cwd + "/userdata/" + fmt.Sprintf("%d", id)
	err = utils.MakeDirIfNotExist(userDataDir)
	if err != nil {
		log.Printf("Failed to create user data directory for session %d: %v", id, err)
		return nil, err
	}
	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserDataDir(userDataDir),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"),
	}
	if headless {
		allocOpts = append(allocOpts, chromedp.Headless)
	}
	allocCtx, releaseAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)

	ctx, releaseCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))

	release := func() {
		releaseAlloc()
		releaseCtx()
	}

	output := make(chan string, 20)

	session := Session{ctx: &ctx, id: id, cookies: "", release: release, output: output}
	return &session, nil
}

func (s *Session) setupCookies() error {
	cookieString := s.cookies
	cookies := utils.ParseCookies(cookieString)
	domain := "grok.com"
	var cookiesToSet []*network.CookieParam
	for _, hc := range cookies {
		param := &network.CookieParam{
			Name:   hc.Name,
			Value:  hc.Value,
			Domain: domain,
			Path:   "/",
		}
		cookiesToSet = append(cookiesToSet, param)
	}
	err := chromedp.Run(*s.ctx, network.SetCookies(cookiesToSet))
	if err != nil {
		log.Printf("Failed to set cookies: %v", err)
		return err
	}
	return nil
}

func (s *Session) jsInjection() error {
	jsToInject := `
Object.defineProperty(navigator, 'webdriver', {
	get: () => false,
})`
	chromedp.Run(*s.ctx, page.SetWebLifecycleState("active"))
	task := chromedp.ActionFunc(func(ctx context.Context) error {
		page.AddScriptToEvaluateOnNewDocument(jsToInject).Do(ctx)
		return nil
	})
	err := chromedp.Run(*s.ctx, task)
	if err != nil {
		log.Printf("Failed to inject JS: %v", err)
		return err
	}
	return nil
}

func (s *Session) navigateToHomepage() error {
	targetURL := "https://grok.com"
	tasks := chromedp.Tasks{
		chromedp.Navigate(targetURL),
	}
	err := chromedp.Run(*s.ctx, tasks)
	if err != nil {
		log.Printf("Failed to navigate: %v", err)
		return err
	}
	return nil
}

func (s *Session) sendPrompt(prompt *string, filename *string, private bool, think bool, cancelListen context.CancelFunc) error {
	grokInputSelector := `textarea[dir="auto"]`
	grokSendButtonSelector := `button[type="submit"]`
	grokPrivateButtonSelector := `a[type="button"]`
	grokThinkButtonSelector := `button[aria-label="Think"]`
	grokInputFileSelector := `input[type="file"]`
	jsSetValueTemplate := `(function (){
let el = document.querySelector('%s');
let descriptor = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(el), 'value');
let prompt = %s;
descriptor.set.call(el, prompt);
let event = new Event('input', { bubbles: true });
el.dispatchEvent(event);
})();
`
	jsClickTemplate := `(function (){
let el = document.querySelector('%s');
el.click();
})();`
	jsonPrompt, err := json.Marshal(*prompt)
	if err != nil {
		log.Printf("Failed to marshal prompt: %v", err)
		cancelListen()
		return err
	}
	setMessage := fmt.Sprintf(jsSetValueTemplate, grokInputSelector, string(jsonPrompt))
	clickSendButton := fmt.Sprintf(jsClickTemplate, grokSendButtonSelector)
	tasks := chromedp.Tasks{
		chromedp.WaitReady(grokInputSelector, chromedp.ByQuery),
		chromedp.WaitReady(grokSendButtonSelector, chromedp.ByQuery),
		chromedp.EvaluateAsDevTools(setMessage, nil),
	}
	if filename != nil {
		files := []string{*filename}
		tasks = append(tasks, chromedp.SetUploadFiles(grokInputFileSelector, files, chromedp.ByQuery))
	}
	if private {
		clickPrivateButton := fmt.Sprintf(jsClickTemplate, grokPrivateButtonSelector)
		tasks = append(tasks, chromedp.WaitVisible(grokPrivateButtonSelector, chromedp.ByQuery))
		tasks = append(tasks, chromedp.EvaluateAsDevTools(clickPrivateButton, nil))
	}
	if think {
		clickThinkButton := fmt.Sprintf(jsClickTemplate, grokThinkButtonSelector)
		tasks = append(tasks, chromedp.WaitVisible(grokThinkButtonSelector, chromedp.ByQuery))
		tasks = append(tasks, chromedp.EvaluateAsDevTools(clickThinkButton, nil))
	}
	tasks = append(tasks, chromedp.WaitEnabled(grokSendButtonSelector, chromedp.ByQuery))
	tasks = append(tasks, chromedp.EvaluateAsDevTools(clickSendButton, nil))
	err = chromedp.Run(*s.ctx, tasks)
	if err != nil {
		log.Printf("Failed to send keys: %v", err)
		cancelListen()
		return err
	}
	return nil
}

func (s *Session) listenForResponse(responseChan chan string, listenCtx context.Context) error {
	defer close(responseChan)
	listenURL := "https://grok.com/rest/app-chat/conversations"
	log.Printf("Listening for response at %s", listenURL)

	var muId sync.Mutex
	var listenRequestID network.RequestID
	requestIDFound := false

	// FIXME: this is a workaround, the correct way is to make this channel one-time usable
	// which means once someone sends a message, the channel is closed
	done := make(chan error, 3)
	head := false
	timer := time.NewTimer(TIMEOUT)
	defer timer.Stop()
	dataChannel := make(chan string, 20)
	defer close(dataChannel)
	processExit := make(chan bool, 1)
	defer close(processExit)
	processCtx, cancelProcess := context.WithCancel(listenCtx)
	defer cancelProcess()
	chromedp.ListenTarget(listenCtx, func(event interface{}) {
		if listenCtx.Err() != nil {
			return
		}
		switch event := event.(type) {
		case *network.EventRequestWillBeSent:
			muId.Lock()
			predication := !requestIDFound && event.Request.Method == "POST" && strings.Contains(event.Request.URL, listenURL) && strings.HasSuffix(event.Request.URL, "/new")
			muId.Unlock()
			if predication {
				log.Printf("Streaming request identified: %s %s (ID: %s)", event.Request.Method, event.Request.URL, event.RequestID)
				muId.Lock()
				listenRequestID = event.RequestID
				requestIDFound = true
				muId.Unlock()
				timer.Stop()

				go func() {
					task := chromedp.ActionFunc(func(ctx context.Context) error {
						_, err := network.StreamResourceContent(event.RequestID).Do(ctx)
						return err
					})
					err := chromedp.Run(listenCtx, task)
					if err != nil {
						log.Printf("Error streaming resource content: %v", err)
						done <- err
					}
				}()

				go ProcessData(dataChannel, processCtx, cancelProcess, responseChan, processExit)
			}
		case *network.EventDataReceived:
			muId.Lock()
			predication := requestIDFound && event.RequestID == listenRequestID
			muId.Unlock()
			if predication {
				go func() {
					muId.Lock()
					if head {
						muId.Unlock()
						select {
						case dataChannel <- event.Data:
						case <-processCtx.Done():
							return
						}
					} else {
						head = true
						muId.Unlock()
					}
				}()
			}
		case *network.EventLoadingFinished:
			muId.Lock()
			predication := requestIDFound && event.RequestID == listenRequestID
			muId.Unlock()
			if predication {
				log.Printf("Loading finished for request ID %s", event.RequestID)
				done <- nil
				return
			}
		case *network.EventLoadingFailed:
			muId.Lock()
			predication := requestIDFound && event.RequestID == listenRequestID
			muId.Unlock()
			if predication {
				log.Printf("Loading failed for request ID %s: %s", event.RequestID, event.ErrorText)
				done <- fmt.Errorf("loading failed for request ID %s: %s", event.RequestID, event.ErrorText)
				return
			}
		}
	})

	for {
		select {
		case err := <-done:
			if err != nil {
				log.Printf("ListenForResponse completed with error: %v", err)
				return err
			}
			cancelProcess()
			<-processExit
			return nil
		case <-timer.C:
			muId.Lock()
			predication := !requestIDFound
			muId.Unlock()
			if predication {
				errMsg := fmt.Sprintf("Timeout waiting for response after %v seconds", TIMEOUT.Seconds())
				log.Println(errMsg)
				return errors.New(errMsg)
			}
		case <-listenCtx.Done():
			<-processExit
			log.Printf("ListenForResponse cancelled by parent context before timeout or completion.")
			return listenCtx.Err()
		}
	}
}

func (s *Session) SendMessage(prompt *string, filename *string, private bool, think bool, responseChan chan string, listenCtx context.Context, cancelListen context.CancelFunc) error {
	err := s.navigateToHomepage()
	if err != nil {
		log.Printf("Failed to navigate to homepage: %v", err)
		return err
	}
	ch := make(chan error, 1)
	go func() {
		err := s.listenForResponse(responseChan, listenCtx)
		ch <- err
	}()
	err = s.sendPrompt(prompt, filename, private, think, cancelListen)
	if err != nil {
		log.Printf("Failed to send prompt: %v", err)
		return err
	}
	err = <-ch
	if err != nil {
		log.Printf("Failed to listen for response: %v", err)
		return err
	}
	log.Printf("Message sent successfully.")
	return nil
}

func (s *Session) Close() {
	if s.release != nil {
		s.release()
	}
	log.Printf("Session %d closed.", s.id)
}

func ProcessData(dataChannel chan string, ctx context.Context, cancel context.CancelFunc, responseChan chan string, processExit chan bool) {
	lineChannel := make(chan string, 20)
	parseExit := make(chan bool, 1)
	go ParseData(lineChannel, ctx, cancel, responseChan, parseExit)
	for data := range dataChannel {
		bytes, err := utils.Base64Decode(data)
		if err != nil {
			log.Printf("Failed to decode data: %v", err)
			continue
		}
		for _, line := range strings.Split(string(*bytes), "\n") {
			select {
			case <-ctx.Done():
				<-parseExit
				close(lineChannel)
				processExit <- true
				return
			case lineChannel <- line:
			}
		}
	}
	close(lineChannel)
}

func ParseData(lineChannel chan string, ctx context.Context, cancel context.CancelFunc, responseChan chan string, parseExit chan bool) {
	think := false
	thinkTag := "<think>"
	for line := range lineChannel {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		f, err := os.OpenFile("./response.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open file: %v", err)
			continue
		}
		if _, err := f.WriteString(line + "\n"); err != nil {
			log.Printf("Failed to write to file: %v", err)
			continue
		}
		f.Close()
		response, _ := utils.ParseGrokResponse(line)
		if response == nil {
			continue
		}
		delta := ""
		if response.IsThinking != think {
			delta += fmt.Sprintf("\n%s\n", thinkTag)
			think = !think
			thinkTag = "</think>"
		}
		delta += response.Token
		fmt.Print(delta)
		select {
		case responseChan <- delta:
		case <-ctx.Done():
			parseExit <- true
			return
		}
		if response.IsSoftStop {
			fmt.Println()
			parseExit <- true
			cancel()
			log.Printf("Finished processing data, cancelling context.")
			return
		}
	}
	parseExit <- true
	cancel()
	log.Printf("Error processing data, cancelling context.")
}
