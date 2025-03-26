package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/RA-Company/logging"
)

var (
	ErrorInvalidToken   = fmt.Errorf("invalid token")
	ErrorInvalidResult  = fmt.Errorf("invalid result")
	ErrorInvalidChat    = fmt.Errorf("invalid chat")
	ErrorInvalidRequest = fmt.Errorf("invalid request")
	ErrorInvalidMessage = fmt.Errorf("invalid message")
)

type SimpleResponse struct {
	Ok          bool   `json:"ok"`          // Ok: True on success
	Result      bool   `json:"result"`      // Result: Result
	Description string `json:"description"` // Description: Human-readable description of the result
}

type MenuButton struct {
	Text              string `json:"text" fake:"{phrase}"`                        // Text: Button text
	Url               string `json:"url,omitempty" fake:"skip"`                   // Url: Button url
	CallbackData      string `json:"callback_data,omitempty" fake:"{noun}"`       // CallbackData: Button callback data
	SwitchInlineQuery string `json:"switch_inline_query,omitempty" fake:"{noun}"` // SwitchInlineQuery: Button switch inline query
}

func (dst *MenuButton) Copy() *MenuButton {
	return &MenuButton{
		Text:         dst.Text,
		CallbackData: dst.CallbackData,
	}
}

func (dst *MenuButton) Equal(src *MenuButton) bool {
	return dst.Text == src.Text &&
		dst.CallbackData == src.CallbackData
}

type Telegram struct {
	Url     string `json:"url"`     // URL for the Telegram API
	Token   string `json:"token"`   // Bot token for the Telegram API
	Timeout int    `json:"timeout"` // Timeout for the Telegram API
}

func (dst *Telegram) Get(ctx context.Context, path string) ([]byte, error) {
	result, err := dst.doRequest(ctx, "GET", path, nil)
	if err != nil {
		if strings.Contains(err.Error(), "Client.Timeout exceeded") {
			result, err = dst.doRequest(ctx, "GET", path, nil)
		}
	}
	return result, err
}

func (dst *Telegram) Post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	url := dst.Url + path

	result, err := dst.doRequest(ctx, "POST", url, data)
	if err != nil {
		if strings.Contains(err.Error(), "Client.Timeout exceeded") {
			result, err = dst.doRequest(ctx, "POST", url, data)
		}
	}
	return result, err
}

// Do request to telegram API
func (dst *Telegram) doRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	start := time.Now()
	if dst.Timeout == 0 {
		dst.Timeout = 30
	}
	client := &http.Client{
		Timeout: time.Duration(dst.Timeout) * time.Second,
	}

	url := fmt.Sprintf("%sbot%s/%s", dst.Url, dst.Token, path)

	var req *http.Request
	var err error
	if payload == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		var data []byte
		switch v := payload.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			data, _ = json.Marshal(v)
		}
		req, err = http.NewRequest(method, url, bytes.NewBuffer(data))
	}
	if err != nil {
		logging.Logs.Errorf(ctx, "%v", err)
		return nil, err
	}

	if dst.Token != "" {
		req.Header.Set("Authorization", dst.Token)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	logging.Logs.Debugf(ctx, "\033[1m\033[36mAPI %s (%.2f ms)\033[1m \033[35m%s\033[0m", method, float64(time.Since(start))/1000000, url)
	if err != nil {
		return nil, err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "Unauthorized") {
		return nil, ErrorInvalidToken
	}

	if strings.Contains(string(body), "\"Not Found\"") {
		return nil, ErrorInvalidToken
	}

	return body, nil
}
