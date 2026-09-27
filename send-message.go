package telegram

import (
	"context"
)

// SendMessageResponse is the Bot API response to sendMessage.
type SendMessageResponse struct {
	OK     bool `json:"ok"` // OK: True on success
	Result struct {
		MessageID int64 `json:"message_id"` // MessageID: Unique message identifier
		From      struct {
			ID        int64  `json:"id"`         // ID: Unique identifier for this user or bot
			IsBot     bool   `json:"is_bot"`     // IsBot: True, if this user is a bot
			FirstName string `json:"first_name"` // FirstName: User's or bot's first name
			LastName  string `json:"last_name"`  // LastName: User's or bot's last name
			Username  string `json:"username"`   // Username: User's or bot's username
		} `json:"from"` // From: Sender
		Chat struct {
			ID        int64  `json:"id"`         // ID: Unique identifier for this chat
			Type      string `json:"type"`       // Type: Type of chat: private, group, supergroup or channel
			Title     string `json:"title"`      // Title: Title, for supergroups, channels and group chats
			Username  string `json:"username"`   // Username: Username, for private chats, supergroups and channels
			FirstName string `json:"first_name"` // FirstName: First name of the other party in a private chat
			LastName  string `json:"last_name"`  // LastName: Last name of the other party in a private chat
		} `json:"chat"` // Chat: Chat
		Date int64  `json:"date"` // Date: Date the message was sent in Unix time
		Text string `json:"text"` // Text: Text of the message
	} `json:"result"` // Result: Result
}

type inlineKeyboardMarkup struct {
	InlineKeyboard [][]MenuButton `json:"inline_keyboard"`
}

type replyKeyboardMarkup struct {
	ResizeKeyboard bool           `json:"resize_keyboard"`
	Keyboard       [][]MenuButton `json:"keyboard"`
}

type replyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

// SendMessage sends a text message to the chat.
func (tg *Telegram) SendMessage(ctx context.Context, chatID int64, text string) (*SendMessageResponse, error) {
	return tg.sendMessage(ctx, chatID, text, nil)
}

// SendInlineButtonsMessage sends a text message with an inline keyboard.
// Without buttons it is sent as a plain message. Rows must not be empty, and each button
// must have a text and exactly one of URL, CallbackData or SwitchInlineQuery, otherwise
// ErrInvalidButton is returned and nothing is sent.
func (tg *Telegram) SendInlineButtonsMessage(ctx context.Context, chatID int64, text string, buttons [][]MenuButton) (*SendMessageResponse, error) {
	if err := validateKeyboard(buttons, true); err != nil {
		return nil, err
	}

	var markup any
	if len(buttons) > 0 {
		markup = inlineKeyboardMarkup{InlineKeyboard: buttons}
	}
	return tg.sendMessage(ctx, chatID, text, markup)
}

// SendReplyButtonsMessage sends a text message with a reply keyboard.
// Without buttons the keyboard currently shown to the user is kept; use RemoveReplyKeyboard to hide it.
// Rows must not be empty and each button must have a text, otherwise ErrInvalidButton is returned
// and nothing is sent. URL, CallbackData and SwitchInlineQuery are ignored by Telegram here.
func (tg *Telegram) SendReplyButtonsMessage(ctx context.Context, chatID int64, text string, buttons [][]MenuButton) (*SendMessageResponse, error) {
	if err := validateKeyboard(buttons, false); err != nil {
		return nil, err
	}
	var markup any
	if len(buttons) > 0 {
		markup = replyKeyboardMarkup{ResizeKeyboard: true, Keyboard: buttons}
	}
	return tg.sendMessage(ctx, chatID, text, markup)
}

// RemoveReplyKeyboard sends a message and removes the reply keyboard shown to the user.
func (tg *Telegram) RemoveReplyKeyboard(ctx context.Context, chatID int64, text string) (*SendMessageResponse, error) {
	return tg.sendMessage(ctx, chatID, text, replyKeyboardRemove{RemoveKeyboard: true})
}

// sendMessage calls the sendMessage method. markup is sent as reply_markup unless it is nil.
func (tg *Telegram) sendMessage(ctx context.Context, chatID int64, text string, markup any) (*SendMessageResponse, error) {
	payload := struct {
		ChatID      int64     `json:"chat_id"`
		Text        string    `json:"text"`
		ParseMode   ParseMode `json:"parse_mode,omitempty"`
		ReplyMarkup any       `json:"reply_markup,omitempty"`
	}{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   tg.ParseMode,
		ReplyMarkup: markup,
	}

	return call[SendMessageResponse](ctx, tg, "sendMessage", payload)
}
