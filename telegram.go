package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ra-company/logging"
)

var (
	// ErrInvalidToken is matched by API errors 401 and 404. Telegram also answers 404
	// to an unknown method name, and the two cases cannot be told apart by the response.
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidResult    = errors.New("invalid result")
	ErrInvalidChat      = errors.New("invalid chat")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrResponseTooLarge = errors.New("response too large")
	ErrInvalidURL       = errors.New("invalid API url: https required")
	ErrInvalidMethod    = errors.New("invalid API method")
	ErrInvalidSecret    = errors.New("invalid webhook secret: 1-256 characters A-Z, a-z, 0-9, _ and - required")
	ErrInvalidButton    = errors.New("invalid button")
)

// APIError is returned when the Bot API responds with "ok": false, or with an error HTTP status
// and a body that is not a Bot API response (not JSON or without the "ok" field).
// It unwraps to ErrInvalidToken, ErrInvalidChat or ErrInvalidRequest, so errors.Is
// keeps working, while errors.As gives access to the details.
//
// Note that code 404 is reported as ErrInvalidToken: Telegram returns 404 "Not Found"
// both for an invalid token and for an unknown method name.
type APIError struct {
	Code        int           // Code: error_code from the response, or the HTTP status
	Description string        // Description: Human-readable description of the error
	RetryAfter  time.Duration // RetryAfter: time to wait before retrying (flood control, code 429)
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram API error %d: %s", e.Code, e.Description)
}

func (e *APIError) Unwrap() error {
	switch {
	case e.Code == http.StatusUnauthorized, e.Code == http.StatusNotFound:
		return ErrInvalidToken
	case e.Code == http.StatusBadRequest && strings.Contains(e.Description, "chat not found"):
		return ErrInvalidChat
	default:
		return ErrInvalidRequest
	}
}

const (
	// DefaultURL is the Bot API URL used when Telegram.URL is empty.
	DefaultURL = "https://api.telegram.org/"

	maxResponseSize    = 8 << 20          // 8 MiB: well above any Bot API JSON response
	defaultTimeout     = 30 * time.Second // Used when Telegram.Timeout is not set
	maxCallbackDataLen = 64               // Bot API limit for callback_data, in bytes
)

// methodRe matches a Bot API method name, e.g. "sendMessage".
var methodRe = regexp.MustCompile(`^[A-Za-z]+$`)

// httpClient is shared by all Telegram instances to reuse connections.
// Timeouts are applied per request via context. Redirects are not followed,
// so the token and payload are never resent to another host.
var httpClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// ParseMode defines how Telegram parses entities in the message text.
type ParseMode string

const (
	ParseModeNone       ParseMode = ""           // Plain text, no entities parsing (default)
	ParseModeMarkdown   ParseMode = "Markdown"   // Legacy Markdown, cannot be escaped reliably
	ParseModeMarkdownV2 ParseMode = "MarkdownV2" // Use EscapeMarkdownV2 for untrusted input
	ParseModeHTML       ParseMode = "HTML"       // Use EscapeHTML for untrusted input
)

var markdownV2Replacer = strings.NewReplacer(
	`\`, `\\`, "_", `\_`, "*", `\*`, "[", `\[`, "]", `\]`, "(", `\(`, ")", `\)`,
	"~", `\~`, "`", "\\`", ">", `\>`, "#", `\#`, "+", `\+`, "-", `\-`,
	"=", `\=`, "|", `\|`, "{", `\{`, "}", `\}`, ".", `\.`, "!", `\!`,
)

// EscapeMarkdownV2 escapes s for safe insertion into a MarkdownV2 message as plain text.
// It must not be used inside code entities (`...` and ```...```): there only ` and \
// have to be escaped, and the extra backslashes would be shown as is.
func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

// EscapeHTML escapes s for safe insertion into an HTML message.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// SimpleResponse is a Bot API response with a boolean result.
type SimpleResponse struct {
	OK          bool   `json:"ok"`          // OK: True on success
	Result      bool   `json:"result"`      // Result: Result
	Description string `json:"description"` // Description: Human-readable description, e.g. "Webhook was set"
}

// MenuButton is a button of an inline or reply keyboard.
type MenuButton struct {
	Text              string `json:"text"`                          // Text: Button text
	URL               string `json:"url,omitempty"`                 // URL: Button url (inline keyboard only)
	CallbackData      string `json:"callback_data,omitempty"`       // CallbackData: Button callback data (inline keyboard only)
	SwitchInlineQuery string `json:"switch_inline_query,omitempty"` // SwitchInlineQuery: Button switch inline query (inline keyboard only)
}

// validateKeyboard checks that every row has buttons and every button is valid
// for an inline or a reply keyboard.
func validateKeyboard(buttons [][]MenuButton, inline bool) error {
	for i, row := range buttons {
		if len(row) == 0 {
			return fmt.Errorf("%w: row %d is empty", ErrInvalidButton, i)
		}
		for j := range row {
			var err error
			if inline {
				err = row[j].validateInline()
			} else {
				err = row[j].validateReply()
			}
			if err != nil {
				return fmt.Errorf("row %d, button %d: %w", i, j, err)
			}
		}
	}
	return nil
}

// validateReply checks that the button can be used in a reply keyboard: it has a text.
func (b *MenuButton) validateReply() error {
	if b.Text == "" {
		return fmt.Errorf("%w: empty text", ErrInvalidButton)
	}
	return nil
}

// validateInline checks that the button can be used in an inline keyboard:
// it has a text and exactly one action (URL, CallbackData or SwitchInlineQuery).
func (b *MenuButton) validateInline() error {
	if b.Text == "" {
		return fmt.Errorf("%w: empty text", ErrInvalidButton)
	}
	actions := 0
	for _, v := range []string{b.URL, b.CallbackData, b.SwitchInlineQuery} {
		if v != "" {
			actions++
		}
	}
	if actions != 1 {
		return fmt.Errorf("%w %q: exactly one of URL, CallbackData or SwitchInlineQuery is required", ErrInvalidButton, b.Text)
	}
	if len(b.CallbackData) > maxCallbackDataLen {
		return fmt.Errorf("%w %q: callback data longer than %d bytes", ErrInvalidButton, b.Text, maxCallbackDataLen)
	}
	return nil
}

// Copy returns a copy of the button, or nil for a nil button.
func (b *MenuButton) Copy() *MenuButton {
	if b == nil {
		return nil
	}
	c := *b
	return &c
}

// Equal reports whether b and other have the same fields. Two nil buttons are equal.
func (b *MenuButton) Equal(other *MenuButton) bool {
	if b == nil || other == nil {
		return b == other
	}
	return *b == *other
}

// Telegram is a Bot API client. It is safe for concurrent use.
type Telegram struct {
	URL       string        `json:"url"`        // URL for the Telegram API, DefaultURL if empty
	Token     string        `json:"token"`      // Bot token for the Telegram API; never log or marshal this struct
	Timeout   time.Duration `json:"timeout"`    // Request timeout, 30s by default; in JSON it is set in nanoseconds (30s = 30000000000)
	ParseMode ParseMode     `json:"parse_mode"` // Parse mode for sent messages, plain text by default
}

// Get calls a Bot API method via GET, e.g. "getMe" or "getUpdates?offset=1".
// The request is retried once on timeout. Use it only for read-only methods:
// a retried state-changing call may be executed twice.
func (tg *Telegram) Get(ctx context.Context, path string) ([]byte, error) {
	result, err := tg.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil && isRequestTimeout(ctx, err) {
		result, err = tg.doRequest(ctx, http.MethodGet, path, nil)
	}
	return result, err
}

// Post calls a Bot API method via POST with data encoded as JSON.
// A string or []byte is sent as is and must already contain valid JSON.
// The request is not retried on timeout: Telegram may have already executed it,
// and a retry could duplicate side effects such as sent messages.
func (tg *Telegram) Post(ctx context.Context, path string, data any) ([]byte, error) {
	return tg.doRequest(ctx, http.MethodPost, path, data)
}

// doRequest calls a Bot API method and returns the response body if it has "ok": true.
func (tg *Telegram) doRequest(ctx context.Context, method, path string, payload any) ([]byte, error) {
	start := time.Now()
	timeout := defaultTimeout
	if tg.Timeout > 0 {
		timeout = tg.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reqURL, err := tg.buildURL(path)
	if err != nil {
		return nil, tg.redactError(err)
	}

	var body io.Reader
	if payload != nil {
		var data []byte
		switch v := payload.(type) {
		case string:
			data = []byte(v)
		case []byte:
			data = v
		default:
			data, err = json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("marshal payload: %w", err)
			}
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, tg.redactError(err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	logging.Logs.Debugf(ctx, "API %s %s (%.2f ms)", method, tg.redact(reqURL), float64(time.Since(start))/float64(time.Millisecond))
	if err != nil {
		return nil, tg.redactError(err)
	}
	defer res.Body.Close()

	result, err := io.ReadAll(io.LimitReader(res.Body, maxResponseSize+1))
	if err != nil {
		return nil, tg.redactError(err)
	}
	if len(result) > maxResponseSize {
		return nil, ErrResponseTooLarge
	}

	if err := checkResponse(res.StatusCode, result); err != nil {
		return nil, err
	}

	return result, nil
}

// call calls a Bot API method via POST and decodes the response into T.
func call[T any](ctx context.Context, tg *Telegram, method string, payload any) (*T, error) {
	body, err := tg.doRequest(ctx, http.MethodPost, method, payload)
	if err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResult, err)
	}
	return &result, nil
}

// checkResponse returns an error unless body is a Bot API response with "ok": true.
// A body that is not a Bot API response (not JSON, or without the "ok" field) is reported
// as APIError with the HTTP status for error statuses and as ErrInvalidResult otherwise.
func checkResponse(status int, body []byte) error {
	var resp struct {
		OK          *bool  `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Parameters  struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	err := json.Unmarshal(body, &resp)
	if err == nil && resp.OK == nil {
		err = errors.New(`missing "ok" field`)
	}
	if err != nil {
		if status < 200 || status > 299 {
			return &APIError{Code: status, Description: http.StatusText(status)}
		}
		return fmt.Errorf("%w: %v", ErrInvalidResult, err)
	}
	if *resp.OK {
		return nil
	}

	code := resp.ErrorCode
	if code == 0 {
		code = status
	}
	return &APIError{
		Code:        code,
		Description: resp.Description,
		RetryAfter:  time.Duration(resp.Parameters.RetryAfter) * time.Second,
	}
}

// buildURL returns the Bot API URL for path, e.g. "sendMessage" or "getUpdates?offset=1".
// The API url must use https (plain http is allowed only for loopback hosts),
// and path must be a bare method name with an optional query.
func (tg *Telegram) buildURL(path string) (string, error) {
	apiURL := tg.URL
	if apiURL == "" {
		apiURL = DefaultURL
	}
	base, err := url.Parse(apiURL)
	if err != nil {
		return "", err
	}
	if base.Scheme != "https" && (base.Scheme != "http" || !isLoopback(base.Hostname())) {
		return "", ErrInvalidURL
	}

	ref, err := url.Parse(path)
	if err != nil || ref.Scheme != "" || ref.Host != "" || ref.Fragment != "" || !methodRe.MatchString(ref.Path) {
		return "", ErrInvalidMethod
	}

	u := base.JoinPath("bot"+tg.Token, ref.Path)
	u.RawQuery = ref.RawQuery
	return u.String(), nil
}

// isLoopback reports whether host is localhost or a loopback IP.
func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isRequestTimeout reports whether err is a per-request timeout while ctx itself is still alive.
func isRequestTimeout(ctx context.Context, err error) bool {
	return errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil
}

// redact masks the bot token in s.
func (tg *Telegram) redact(s string) string {
	if tg.Token == "" {
		return s
	}
	return strings.ReplaceAll(s, tg.Token, "<token>")
}

// redactError masks the bot token in the URL carried by *url.Error,
// keeping the error type, its cause and Timeout() intact.
func (tg *Telegram) redactError(err error) error {
	if uerr, ok := errors.AsType[*url.Error](err); ok {
		uerr.URL = tg.redact(uerr.URL)
	}
	return err
}
