//go:build integration

package telegram

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWebhookURL is a fictitious endpoint: Telegram accepts it, and updates never reach a real application.
const testWebhookURL = "https://example.com/telegram-webhook-test"

func TestTelegram_SetWebhook(t *testing.T) {
	ctx := context.Background()
	url, api := GetAPIData(t)
	secret := uuid.New().String()

	t.Cleanup(func() {
		tg := Telegram{URL: url, Token: api}
		_, err := tg.DeleteWebhook(context.Background(), true)
		assert.NoError(t, err, "cleanup: DeleteWebhook")
	})

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		data, err := tg.SetWebhook(ctx, testWebhookURL, []string{"message"}, secret)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
		require.True(t, data.Result)
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: "invalid"}
		_, err := tg.SetWebhook(ctx, testWebhookURL, []string{"message"}, secret)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}
