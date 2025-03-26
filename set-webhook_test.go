package telegram

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTelegram_SetWebhook(t *testing.T) {
	ctx := context.Background()
	url, api := GetApiData(t)
	secret := uuid.New().String()

	tg := Telegram{Url: url, Token: api}
	defer tg.DeleteWebhook(ctx)

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		data, err := tg.SetWebhook(ctx, url, []string{"message"}, secret)
		require.NoError(t, err, "ctd.SetWebhook(ctx)")
		require.NotNil(t, data, "ctd.SetWebhook(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SetWebhook(ctx)")
		require.Equal(t, true, data.Result, "ctd.SetWebhook(ctx)")
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: "invalid"}
		_, err := tg.SetWebhook(ctx, url, []string{"message"}, secret)
		require.ErrorIs(t, err, ErrorInvalidToken, "ctd.SetWebhook(ctx)")
	})
}
