# telegram
Simple telegram library

## Installation

```bash
go get github.com/ra-company/telegram
```

Requires Go 1.26.4 or newer.

## Quick start

```go
tg := &telegram.Telegram{Token: os.Getenv("TG_TOKEN")}

res, err := tg.SendMessage(context.Background(), chatID, "Hello!")
if err != nil {
	return err
}
fmt.Println("message id:", res.Result.MessageID)
```

More examples (formatting, keyboards, webhook, error handling) are in [example_test.go](example_test.go)
and on [pkg.go.dev](https://pkg.go.dev/github.com/ra-company/telegram#pkg-examples). They are compiled
with the tests, so they always match the current API.

## Things to know

- **Client.** `URL` defaults to `telegram.DefaultURL` and must use `https` (plain `http` is allowed only
  for `localhost`). The client is safe for concurrent use. Never log or marshal the `Telegram` struct:
  it contains the bot token.
- **Timeout.** `Timeout` is a `time.Duration`, 30s by default. When the struct is loaded from JSON, the
  value is in **nanoseconds**: `"timeout": 30` means 30ns and every request fails immediately. Use
  `"timeout": 30000000000` for 30 seconds.
- **Formatting.** Messages are sent as plain text by default. When you set `ParseMode`, escape untrusted
  input with `EscapeHTML` or `EscapeMarkdownV2`. Legacy `ParseModeMarkdown` cannot be escaped reliably.
- **Keyboards.** Rows must not be empty and every button must have a text. Each inline button also needs
  exactly one of `URL`, `CallbackData` (up to 64 bytes) or `SwitchInlineQuery`. Otherwise
  `ErrInvalidButton` is returned and nothing is sent. `SendReplyButtonsMessage` without buttons keeps
  the current keyboard; use `RemoveReplyKeyboard` to hide it.
- **Webhook.** `SetWebhook` requires a secret (1-256 chars: `A-Z`, `a-z`, `0-9`, `_`, `-`); check it
  in your handler with `VerifyWebhookSecret`. `DeleteWebhook` takes `dropPendingUpdates`.
- **Raw calls.** `Get` and `Post` call any Bot API method. `Get` is retried once on timeout, so use it
  only for read-only methods; `Post` is never retried.
- **Errors.** API errors are returned as `*APIError` (with `Code`, `Description` and `RetryAfter`) and
  match `ErrInvalidToken` (401, 404), `ErrInvalidChat` or `ErrInvalidRequest` via `errors.Is`.
  Telegram returns 404 both for an invalid token and for an unknown method.

## Testing

```bash
make tests
```

Runs unit tests and examples against a local mock server; no network or credentials are needed.

```bash
make tests-integration
```

Also runs integration tests against the real Bot API. Copy `.sample.env` to `.test.env` and fill in the variables first. Use a dedicated test bot: the tests set and delete its webhook and drop pending updates.

## License

[MIT](LICENSE)

## Dependencies

**Runtime**

| Library | Purpose | License |
|---------|---------|---------|
| [github.com/ra-company/logging](https://github.com/ra-company/logging) | Logging | MIT |

**Tests only**

| Library | Purpose | License |
|---------|---------|---------|
| [github.com/brianvoe/gofakeit](https://github.com/brianvoe/gofakeit) | Random test data | MIT |
| [github.com/google/uuid](https://github.com/google/uuid) | RFC 4122 UUID generation | BSD-3-Clause |
| [github.com/stretchr/testify](https://github.com/stretchr/testify) | Test assertions | MIT |
