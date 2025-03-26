package telegram

import (
	"context"
	"encoding/json"

	"github.com/RA-Company/logging"
)

// SetWebhook method
//
// Parameters:
//   - ctx (context.Context): Context
//   - hook_url (string): Webhook URL
//   - events ([]string): Allowed updates
//   - secret (string): Secret token
func (dst *Telegram) SetWebhook(ctx context.Context, hook_url string, events []string, secret string) (*SimpleResponse, error) {
	url := "setWebhook"

	payload := struct {
		Url            string   `json:"url"`
		AllowedUpdates []string `json:"allowed_updates"`
		SecretToken    string   `json:"secret_token"`
	}{
		Url:            hook_url,
		AllowedUpdates: events,
		SecretToken:    secret,
	}

	body, err := dst.doRequest(ctx, "POST", url, payload)
	if err != nil {
		logging.Logs.Errorf("doRequest() error: %v", err)
		return nil, err
	}

	result := SimpleResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		logging.Logs.Errorf("json.Unmarshal() error: %v", err)
		return nil, ErrorInvalidResult
	}

	return &result, err
}
