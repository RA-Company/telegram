package telegram

import (
	"context"
	"crypto/subtle"
	"net/http"
	"regexp"
)

// SecretTokenHeader is the header Telegram uses to pass the webhook secret token.
const SecretTokenHeader = "X-Telegram-Bot-Api-Secret-Token"

// secretRe matches a valid webhook secret token.
var secretRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)

// SetWebhook sets the URL Telegram sends updates to.
//
// secret is required: Telegram passes it in the SecretTokenHeader of every update,
// so the webhook handler can reject forged requests with VerifyWebhookSecret.
// events lists the allowed update types; nil keeps the previous setting,
// an empty slice enables all types except chat_member, message_reaction and message_reaction_count.
func (tg *Telegram) SetWebhook(ctx context.Context, hookURL string, events []string, secret string) (*SimpleResponse, error) {
	if !secretRe.MatchString(secret) {
		return nil, ErrInvalidSecret
	}

	payload := struct {
		URL            string   `json:"url"`
		AllowedUpdates []string `json:"allowed_updates,omitzero"`
		SecretToken    string   `json:"secret_token"`
	}{
		URL:            hookURL,
		AllowedUpdates: events,
		SecretToken:    secret,
	}

	return call[SimpleResponse](ctx, tg, "setWebhook", payload)
}

// VerifyWebhookSecret reports whether the update request carries the expected secret token.
// It always returns false for an empty secret.
func VerifyWebhookSecret(r *http.Request, secret string) bool {
	if secret == "" {
		return false
	}
	got := r.Header.Get(SecretTokenHeader)
	return subtle.ConstantTimeCompare([]byte(got), []byte(secret)) == 1
}
