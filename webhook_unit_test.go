package telegram

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegram_SetWebhookRequest(t *testing.T) {
	api, tg := newMockAPI(t)
	api.reply(200, `{"ok":true,"result":true,"description":"Webhook was set"}`)

	res, err := tg.SetWebhook(context.Background(), "https://example.com/hook", []string{"message"}, "s3cret")
	require.NoError(t, err)
	require.True(t, res.OK)
	require.True(t, res.Result)
	require.Equal(t, "Webhook was set", res.Description)

	req := api.last(t)
	require.Equal(t, "/bot"+mockToken+"/setWebhook", req.Path)
	require.JSONEq(t, `{"url":"https://example.com/hook","allowed_updates":["message"],"secret_token":"s3cret"}`, req.Body)
}

func TestTelegram_DeleteWebhookRequest(t *testing.T) {
	tests := []struct {
		name string
		drop bool
		want string
	}{
		{"drop pending updates", true, `{"drop_pending_updates":true}`},
		{"keep pending updates", false, `{"drop_pending_updates":false}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, tg := newMockAPI(t)

			res, err := tg.DeleteWebhook(context.Background(), tt.drop)
			require.NoError(t, err)
			require.True(t, res.OK)

			req := api.last(t)
			require.Equal(t, "/bot"+mockToken+"/deleteWebhook", req.Path)
			require.JSONEq(t, tt.want, req.Body)
		})
	}
}

func TestTelegram_WebhookErrors(t *testing.T) {
	const badRequest = `{"ok":false,"error_code":400,"description":"Bad Request: bad webhook: HTTPS url must be provided for webhook"}`

	t.Run("set webhook", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(400, badRequest)

		res, err := tg.SetWebhook(context.Background(), "http://example.com/hook", nil, "s3cret")
		require.Nil(t, res)
		require.ErrorIs(t, err, ErrInvalidRequest)
	})

	t.Run("delete webhook", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(401, `{"ok":false,"error_code":401,"description":"Unauthorized"}`)

		res, err := tg.DeleteWebhook(context.Background(), true)
		require.Nil(t, res)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestTelegram_SetWebhookSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		err    error
	}{
		{"valid", "abc_DEF-123", nil},
		{"max length", strings.Repeat("a", 256), nil},
		{"empty", "", ErrInvalidSecret},
		{"too long", strings.Repeat("a", 257), ErrInvalidSecret},
		{"invalid characters", "abc def!", ErrInvalidSecret},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, tg := newMockAPI(t)
			_, err := tg.SetWebhook(context.Background(), "https://example.com/hook", nil, tt.secret)
			require.ErrorIs(t, err, tt.err)
			if tt.err != nil {
				require.Zero(t, api.count(), "invalid secret must not be sent")
			}
		})
	}
}

func TestTelegram_SetWebhookAllowedUpdates(t *testing.T) {
	tests := []struct {
		name   string
		events []string
		want   string
	}{
		{"nil keeps previous setting", nil, `{"url":"https://example.com/hook","secret_token":"s"}`},
		{"empty enables defaults", []string{}, `{"url":"https://example.com/hook","allowed_updates":[],"secret_token":"s"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, tg := newMockAPI(t)
			_, err := tg.SetWebhook(context.Background(), "https://example.com/hook", tt.events, "s")
			require.NoError(t, err)
			require.JSONEq(t, tt.want, api.last(t).Body)
		})
	}
}

func TestVerifyWebhookSecret(t *testing.T) {
	tests := []struct {
		name   string
		header string
		secret string
		want   bool
	}{
		{"match", "s3cret", "s3cret", true},
		{"mismatch", "wrong", "s3cret", false},
		{"missing header", "", "s3cret", false},
		{"empty secret", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/hook", nil)
			if tt.header != "" {
				r.Header.Set(SecretTokenHeader, tt.header)
			}
			require.Equal(t, tt.want, VerifyWebhookSecret(r, tt.secret))
		})
	}
}
