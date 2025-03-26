package telegram

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/RA-Company/logging"
)

type SendMessageResponse struct {
	Ok     bool `json:"ok"` // Ok: True on success
	Result struct {
		MessageID int64 `json:"message_id"` // MessageID: Unique message identifier
		From      struct {
			ID        int64  `json:"id"`         // ID: Unique identifier for this user or bot
			IsBot     bool   `json:"is_bot"`     // IsBot: True, if this user is a bot
			FirstName string `json:"first_name"` // FirstName: User's or bot's first name
			Username  string `json:"username"`   // Username: User's or bot's username
		} `json:"from"` // From: Sender
		Chat struct {
			ID        int64  `json:"id"`         // ID: Unique identifier for this chat
			FirstName string `json:"first_name"` // FirstName: Chat title
			Username  string `json:"username"`   // Username: Username of the chat
			Type      string `json:"type"`       // Type: Type of chat
		} `json:"chat"` // Chat: Chat
		Date int    `json:"date"` // Date: Date the message was sent in Unix time
		Text string `json:"text"` // Text: Text of the message
	} `json:"result"` // Result: Result
	Description string `json:"description"` // Description: Human-readable description of the result
}

func (dst *Telegram) SendMessage(ctx context.Context, chat_id int64, text string) (*SendMessageResponse, error) {
	url := "sendMessage"

	payload := struct {
		ChatID    int64  `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    chat_id,
		Text:      text,
		ParseMode: "markdown",
	}

	body, err := dst.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "chat not found") {
		return nil, ErrorInvalidChat
	}

	result := SendMessageResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		logging.Logs.Errorf("SendMessage JSON error: %v", err)
		return nil, ErrorInvalidResult
	}

	return &result, err
}

func (dst *Telegram) SendInlineButtonsMessage(ctx context.Context, chat_id int64, text string, buttons [][]MenuButton) (*SendMessageResponse, error) {
	url := "sendMessage"

	payload := struct {
		ChatID      int64  `json:"chat_id"`
		Text        string `json:"text"`
		ParseMode   string `json:"parse_mode"`
		ReplyMarkup struct {
			InlineKeyboard [][]MenuButton `json:"inline_keyboard"`
		} `json:"reply_markup"`
	}{
		ChatID:    chat_id,
		Text:      text,
		ParseMode: "markdown",
	}
	payload.ReplyMarkup.InlineKeyboard = slices.Clone(buttons)

	body, err := dst.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "chat not found") {
		return nil, ErrorInvalidChat
	}

	result := SendMessageResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		logging.Logs.Errorf("SendMessage JSON error: %v", err)
		return nil, ErrorInvalidResult
	}

	if !result.Ok {
		logging.Logs.Errorf("SendMessage error: %v", result.Description)
		return &result, ErrorInvalidRequest
	}

	return &result, err
}

func (dst *Telegram) SendReplyButtonsMessage(ctx context.Context, chat_id int64, text string, buttons [][]MenuButton) (*SendMessageResponse, error) {
	url := "sendMessage"

	payload := struct {
		ChatID      int64  `json:"chat_id"`
		Text        string `json:"text"`
		ParseMode   string `json:"parse_mode"`
		ReplyMarkup struct {
			ResizeKeyboard bool           `json:"resize_keyboard"`
			Keyboard       [][]MenuButton `json:"keyboard"`
		} `json:"reply_markup"`
	}{
		ChatID:    chat_id,
		Text:      text,
		ParseMode: "markdown",
	}
	payload.ReplyMarkup.ResizeKeyboard = true
	payload.ReplyMarkup.Keyboard = slices.Clone(buttons)

	body, err := dst.doRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, err
	}

	if strings.Contains(string(body), "chat not found") {
		return nil, ErrorInvalidChat
	}

	result := SendMessageResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		logging.Logs.Errorf("SendMessage JSON error: %v", err)
		return nil, ErrorInvalidResult
	}

	if !result.Ok {
		logging.Logs.Errorf("SendMessage error: %v", result.Description)
		return &result, ErrorInvalidRequest
	}

	return &result, err
}
