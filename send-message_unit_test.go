package telegram

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const sendMessageOK = `{"ok":true,"result":{"message_id":42,"chat":{"id":1,"type":"group","title":"Team"},"text":"hi"}}`

func TestTelegram_SendMessageRequest(t *testing.T) {
	buttons := [][]MenuButton{{{Text: "a", CallbackData: "x"}}}

	tests := []struct {
		name      string
		parseMode ParseMode
		send      func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error)
		want      string
	}{
		{
			name: "plain text by default",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendMessage(ctx, 1, "hi")
			},
			want: `{"chat_id":1,"text":"hi"}`,
		},
		{
			name:      "parse mode",
			parseMode: ParseModeHTML,
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendMessage(ctx, 1, "<b>hi</b>")
			},
			want: `{"chat_id":1,"text":"<b>hi</b>","parse_mode":"HTML"}`,
		},
		{
			name: "inline buttons",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendInlineButtonsMessage(ctx, 1, "hi", buttons)
			},
			want: `{"chat_id":1,"text":"hi","reply_markup":{"inline_keyboard":[[{"text":"a","callback_data":"x"}]]}}`,
		},
		{
			name: "inline buttons nil",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendInlineButtonsMessage(ctx, 1, "hi", nil)
			},
			want: `{"chat_id":1,"text":"hi"}`,
		},
		{
			name: "inline buttons empty",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendInlineButtonsMessage(ctx, 1, "hi", [][]MenuButton{})
			},
			want: `{"chat_id":1,"text":"hi"}`,
		},
		{
			name: "reply buttons",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendReplyButtonsMessage(ctx, 1, "hi", buttons)
			},
			want: `{"chat_id":1,"text":"hi","reply_markup":{"resize_keyboard":true,"keyboard":[[{"text":"a","callback_data":"x"}]]}}`,
		},
		{
			name: "reply buttons nil keeps keyboard",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendReplyButtonsMessage(ctx, 1, "hi", nil)
			},
			want: `{"chat_id":1,"text":"hi"}`,
		},
		{
			name: "reply buttons empty keeps keyboard",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.SendReplyButtonsMessage(ctx, 1, "hi", [][]MenuButton{})
			},
			want: `{"chat_id":1,"text":"hi"}`,
		},
		{
			name: "remove reply keyboard",
			send: func(ctx context.Context, tg *Telegram) (*SendMessageResponse, error) {
				return tg.RemoveReplyKeyboard(ctx, 1, "hi")
			},
			want: `{"chat_id":1,"text":"hi","reply_markup":{"remove_keyboard":true}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, tg := newMockAPI(t)
			api.reply(200, sendMessageOK)
			tg.ParseMode = tt.parseMode

			res, err := tt.send(context.Background(), tg)
			require.NoError(t, err)
			require.True(t, res.OK)
			require.EqualValues(t, 42, res.Result.MessageID)
			require.Equal(t, "Team", res.Result.Chat.Title)

			req := api.last(t)
			require.Equal(t, "/bot"+mockToken+"/sendMessage", req.Path)
			require.JSONEq(t, tt.want, req.Body)
		})
	}
}

func TestTelegram_SendMessageErrors(t *testing.T) {
	t.Run("chat not found", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`)

		res, err := tg.SendMessage(context.Background(), 1, "hi")
		require.Nil(t, res)
		require.ErrorIs(t, err, ErrInvalidChat)
	})

	t.Run("ok false", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(400, `{"ok":false,"error_code":400,"description":"Bad Request: message is too long"}`)

		res, err := tg.SendMessage(context.Background(), 1, "hi")
		require.Nil(t, res)
		require.ErrorIs(t, err, ErrInvalidRequest)

		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, "Bad Request: message is too long", apiErr.Description)
	})

	t.Run("invalid inline buttons", func(t *testing.T) {
		tests := []struct {
			name   string
			button MenuButton
			msg    string
		}{
			{"empty text", MenuButton{CallbackData: "x"}, "empty text"},
			{"no action", MenuButton{Text: "a"}, "exactly one"},
			{"two actions", MenuButton{Text: "a", URL: "https://x", CallbackData: "x"}, "exactly one"},
			{"long callback data", MenuButton{Text: "a", CallbackData: strings.Repeat("x", 65)}, "longer than 64 bytes"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				api, tg := newMockAPI(t)
				buttons := [][]MenuButton{{{Text: "ok", CallbackData: "ok"}}, {tt.button}}

				res, err := tg.SendInlineButtonsMessage(context.Background(), 1, "hi", buttons)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidButton)
				require.ErrorContains(t, err, tt.msg)
				require.Zero(t, api.count(), "invalid buttons must not be sent")
			})
		}
	})

	t.Run("invalid keyboards", func(t *testing.T) {
		tests := []struct {
			name    string
			buttons [][]MenuButton
			send    func(tg *Telegram, buttons [][]MenuButton) (*SendMessageResponse, error)
			msg     string
		}{
			{"inline empty row", [][]MenuButton{{{Text: "a", CallbackData: "x"}}, {}}, inlineSend, "row 1 is empty"},
			{"reply empty row", [][]MenuButton{{}}, replySend, "row 0 is empty"},
			{"reply empty text", [][]MenuButton{{{Text: "a"}, {Text: ""}}}, replySend, "row 0, button 1: invalid button: empty text"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				api, tg := newMockAPI(t)
				res, err := tt.send(tg, tt.buttons)
				require.Nil(t, res)
				require.ErrorIs(t, err, ErrInvalidButton)
				require.ErrorContains(t, err, tt.msg)
				require.Zero(t, api.count())
			})
		}
	})

	t.Run("valid inline button actions", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(200, sendMessageOK)
		buttons := [][]MenuButton{{
			{Text: "url", URL: "https://x"},
			{Text: "cb", CallbackData: strings.Repeat("x", 64)},
			{Text: "inline", SwitchInlineQuery: "q"},
		}}

		_, err := tg.SendInlineButtonsMessage(context.Background(), 1, "hi", buttons)
		require.NoError(t, err)
	})

	t.Run("text with error words", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(200, `{"ok":true,"result":{"message_id":1,"text":"Unauthorized: chat not found"}}`)

		res, err := tg.SendMessage(context.Background(), 1, "Unauthorized: chat not found")
		require.NoError(t, err)
		require.Equal(t, "Unauthorized: chat not found", res.Result.Text)
	})
}

func inlineSend(tg *Telegram, buttons [][]MenuButton) (*SendMessageResponse, error) {
	return tg.SendInlineButtonsMessage(context.Background(), 1, "hi", buttons)
}

func replySend(tg *Telegram, buttons [][]MenuButton) (*SendMessageResponse, error) {
	return tg.SendReplyButtonsMessage(context.Background(), 1, "hi", buttons)
}
