package telegram

import (
	"context"
	"encoding/json"

	"github.com/RA-Company/logging"
)

func (dst *Telegram) DeleteWebhook(ctx context.Context) (*SimpleResponse, error) {
	url := "deleteWebhook"

	payload := struct {
		DropPendingUpdates bool `json:"drop_pending_updates"`
	}{
		DropPendingUpdates: true,
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
