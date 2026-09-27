//go:build integration

package telegram

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegram_DeleteWebhook(t *testing.T) {
	ctx := context.Background()
	url, api := GetAPIData(t)

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		data, err := tg.DeleteWebhook(ctx, true)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
		require.True(t, data.Result)
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: "invalid"}
		_, err := tg.DeleteWebhook(ctx, true)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}
