package telegram_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ra-company/telegram"
)

func ExampleTelegram_SendMessage() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}

	// Plain text by default: Markdown and HTML are not parsed.
	res, err := tg.SendMessage(context.Background(), 123456789, "Hello!")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("message id:", res.Result.MessageID)
}

func ExampleTelegram_SendMessage_parseMode() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN"), ParseMode: telegram.ParseModeHTML}
	userInput := "<script>"

	_, err := tg.SendMessage(context.Background(), 123456789, "<b>Order:</b> "+telegram.EscapeHTML(userInput))
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleTelegram_SendInlineButtonsMessage() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}

	buttons := [][]telegram.MenuButton{
		{{Text: "Yes", CallbackData: "yes"}, {Text: "No", CallbackData: "no"}},
		{{Text: "Website", URL: "https://example.com"}},
	}
	_, err := tg.SendInlineButtonsMessage(context.Background(), 123456789, "Continue?", buttons)
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleTelegram_SendReplyButtonsMessage() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}
	ctx := context.Background()

	buttons := [][]telegram.MenuButton{
		{{Text: "Menu"}, {Text: "Help"}},
	}
	if _, err := tg.SendReplyButtonsMessage(ctx, 123456789, "Choose an option", buttons); err != nil {
		log.Fatal(err)
	}

	// Hide the keyboard later.
	if _, err := tg.RemoveReplyKeyboard(ctx, 123456789, "Done"); err != nil {
		log.Fatal(err)
	}
}

func ExampleTelegram_SetWebhook() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}
	secret := os.Getenv("TG_WEBHOOK_SECRET") // 1-256 chars: A-Z, a-z, 0-9, _ and -

	_, err := tg.SetWebhook(context.Background(), "https://example.com/telegram", []string{"message", "callback_query"}, secret)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/telegram", func(w http.ResponseWriter, r *http.Request) {
		if !telegram.VerifyWebhookSecret(r, secret) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		// decode and handle the update from r.Body
	})
}

func ExampleAPIError() {
	tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}

	_, err := tg.SendMessage(context.Background(), 123456789, "Hello!")

	var apiErr *telegram.APIError
	switch {
	case errors.Is(err, telegram.ErrInvalidChat):
		// chat not found
	case errors.Is(err, telegram.ErrInvalidToken):
		// 401 or 404: invalid token (404 is also returned for an unknown method)
	case errors.As(err, &apiErr) && apiErr.Code == http.StatusTooManyRequests:
		time.Sleep(apiErr.RetryAfter)
	case err != nil:
		log.Fatal(err)
	}
}

func ExampleEscapeHTML() {
	fmt.Println(telegram.EscapeHTML(`<b>"Tom" & Jerry</b>`))
	// Output: &lt;b&gt;&#34;Tom&#34; &amp; Jerry&lt;/b&gt;
}

func ExampleEscapeMarkdownV2() {
	fmt.Println(telegram.EscapeMarkdownV2("Price: 1.5*2 = 3 (approx.)"))
	// Output: Price: 1\.5\*2 \= 3 \(approx\.\)
}
