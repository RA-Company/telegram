package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegram_DeletWebhook(t *testing.T) {
	ctx := context.Background()
	url, api := GetApiData(t)

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		data, err := tg.DeleteWebhook(ctx)
		require.NoError(t, err, "ctd.DeleteWebhook(ctx)")
		require.NotNil(t, data, "ctd.DeleteWebhook(ctx)")
		require.Equal(t, true, data.Ok, "ctd.DeleteWebhook(ctx)")
		require.Equal(t, true, data.Result, "ctd.DeleteWebhook(ctx)")
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: "invalid"}
		_, err := tg.DeleteWebhook(ctx)
		require.ErrorIs(t, err, ErrorInvalidToken, "ctd.DeleteWebhook(ctx)")
	})
}
