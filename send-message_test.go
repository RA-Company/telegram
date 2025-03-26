package telegram

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"
)

func TestTelegram_SendMessage(t *testing.T) {
	ctx := context.Background()
	url, api := GetApiData(t)

	str := os.Getenv("TEST_TG_USER")
	require.NotEqual(t, "", str, "TEST_TG_USER is not set")
	userId, err := strconv.Atoi(str)
	require.NoError(t, err, "strconv.Atoi(str)")
	require.NotEqual(t, 0, userId, "TEST_TG_USER is incorrect")

	faker := gofakeit.New(0)
	text := faker.Sentence(5)

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		data, err := tg.SendMessage(ctx, int64(userId), text)
		require.NoError(t, err, "ctd.SendMessage(ctx)")
		require.NotNil(t, data, "ctd.SendMessage(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SendMessage(ctx)")
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{Url: url, Token: "invalid"}
		_, err := tg.SendMessage(ctx, int64(userId), text)
		require.ErrorIs(t, err, ErrorInvalidToken, "ctd.SendMessage(ctx)")
	})

	t.Run("3 incorrect user", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		_, err := tg.SendMessage(ctx, int64(userId)+1, text)
		require.ErrorIs(t, err, ErrorInvalidChat, "ctd.SendMessage(ctx)")
	})

	t.Run("4 send inline button message", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		var buttons [][]MenuButton
		faker.Slice(&buttons)
		data, err := tg.SendInlineButtonsMessage(ctx, int64(userId), text, buttons)
		require.NoError(t, err, "ctd.SendInlineButton(ctx)")
		require.NotNil(t, data, "ctd.SendInlineButton(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SendInlineButton(ctx)")
	})

	t.Run("5 send button message", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		var buttons [][]MenuButton
		faker.Slice(&buttons)
		data, err := tg.SendReplyButtonsMessage(ctx, int64(userId), text, buttons)
		require.NoError(t, err, "ctd.SendInlineButton(ctx)")
		require.NotNil(t, data, "ctd.SendInlineButton(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SendInlineButton(ctx)")
	})

	t.Run("6 send inline button message with url", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		var buttons [][]MenuButton
		faker.Slice(&buttons)
		for _, line := range buttons {
			for btn := range line {
				line[btn].Url = faker.URL()
				line[btn].CallbackData = ""
			}
		}
		data, err := tg.SendInlineButtonsMessage(ctx, int64(userId), text, buttons)
		require.NoError(t, err, "ctd.SendInlineButton(ctx)")
		require.NotNil(t, data, "ctd.SendInlineButton(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SendInlineButton(ctx)")
	})

	t.Run("7 send ireply buttons in 3 row", func(t *testing.T) {
		tg := Telegram{Url: url, Token: api}
		buttons := [][]MenuButton{
			{
				{Text: faker.Sentence(2)},
			},
			{
				{Text: faker.Sentence(2)},
				{Text: faker.Sentence(2)},
			},
			{
				{Text: faker.Sentence(2)},
			},
		}
		data, err := tg.SendReplyButtonsMessage(ctx, int64(userId), text, buttons)
		require.NoError(t, err, "ctd.SendInlineButton(ctx)")
		require.NotNil(t, data, "ctd.SendInlineButton(ctx)")
		require.Equal(t, true, data.Ok, "ctd.SendInlineButton(ctx)")
	})
}
