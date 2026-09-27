package telegram

import (
	"context"
)

// DeleteWebhook removes the webhook. If dropPendingUpdates is true, updates
// that Telegram has not delivered yet are discarded; otherwise they are kept
// and can be received later with getUpdates or a new webhook.
func (tg *Telegram) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (*SimpleResponse, error) {
	payload := struct {
		DropPendingUpdates bool `json:"drop_pending_updates"`
	}{
		DropPendingUpdates: dropPendingUpdates,
	}

	return call[SimpleResponse](ctx, tg, "deleteWebhook", payload)
}
