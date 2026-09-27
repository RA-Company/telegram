//go:build integration

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
	url, api := GetAPIData(t)

	str := os.Getenv("TEST_TG_USER")
	require.NotEqual(t, "", str, "TEST_TG_USER is not set")
	userID, err := strconv.ParseInt(str, 10, 64)
	require.NoError(t, err, "TEST_TG_USER is not a number")
	require.NotZero(t, userID, "TEST_TG_USER is incorrect")

	faker := gofakeit.New(0)
	text := faker.Sentence(5)

	t.Run("1 correct api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		data, err := tg.SendMessage(ctx, userID, text)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
	})

	t.Run("2 incorrect api", func(t *testing.T) {
		tg := Telegram{URL: url, Token: "invalid"}
		_, err := tg.SendMessage(ctx, userID, text)
		require.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("3 incorrect user", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		_, err := tg.SendMessage(ctx, userID+1, text)
		require.ErrorIs(t, err, ErrInvalidChat)
	})

	t.Run("4 send inline button message", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		buttons := fakeKeyboard(faker, func() MenuButton {
			return MenuButton{Text: faker.Phrase(), CallbackData: faker.Noun()}
		})
		data, err := tg.SendInlineButtonsMessage(ctx, userID, text, buttons)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
	})

	t.Run("5 send button message", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		buttons := fakeKeyboard(faker, func() MenuButton {
			return MenuButton{Text: faker.Phrase()}
		})
		data, err := tg.SendReplyButtonsMessage(ctx, userID, text, buttons)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
	})

	t.Run("6 send inline button message with url", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
		buttons := fakeKeyboard(faker, func() MenuButton {
			return MenuButton{Text: faker.Phrase(), URL: faker.URL()}
		})
		data, err := tg.SendInlineButtonsMessage(ctx, userID, text, buttons)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
	})

	t.Run("7 send ireply buttons in 3 row", func(t *testing.T) {
		tg := Telegram{URL: url, Token: api}
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
		data, err := tg.SendReplyButtonsMessage(ctx, userID, text, buttons)
		require.NoError(t, err)
		require.NotNil(t, data)
		require.True(t, data.OK)
	})
}

// fakeKeyboard returns 1-3 rows of 1-3 buttons created by button.
func fakeKeyboard(faker *gofakeit.Faker, button func() MenuButton) [][]MenuButton {
	rows := make([][]MenuButton, faker.IntRange(1, 3))
	for i := range rows {
		rows[i] = make([]MenuButton, faker.IntRange(1, 3))
		for j := range rows[i] {
			rows[i][j] = button()
		}
	}
	return rows
}
