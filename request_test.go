package telegram

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSlowServer starts a server that does not answer until the request is canceled.
func newSlowServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	calls := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// The server notices a client disconnect only after the body is consumed.
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	return srv, calls
}

func TestTelegram_BuildURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		path string
		want string
		err  error
	}{
		{"default url", "", "sendMessage", "https://api.telegram.org/bot123:SECRET/sendMessage", nil},
		{"base with slash", "https://api.telegram.org/", "sendMessage", "https://api.telegram.org/bot123:SECRET/sendMessage", nil},
		{"base without slash", "https://api.telegram.org", "sendMessage", "https://api.telegram.org/bot123:SECRET/sendMessage", nil},
		{"query kept", "https://api.telegram.org/", "getUpdates?offset=5&timeout=1", "https://api.telegram.org/bot123:SECRET/getUpdates?offset=5&timeout=1", nil},
		{"http loopback ip", "http://127.0.0.1:8080/", "getMe", "http://127.0.0.1:8080/bot123:SECRET/getMe", nil},
		{"http localhost", "http://localhost/", "getMe", "http://localhost/bot123:SECRET/getMe", nil},
		{"http external host", "http://api.telegram.org/", "getMe", "", ErrInvalidURL},
		{"other scheme", "ftp://api.telegram.org/", "getMe", "", ErrInvalidURL},
		{"path traversal", "https://api.telegram.org/", "../getMe", "", ErrInvalidMethod},
		{"absolute url", "https://api.telegram.org/", "https://evil.com/getMe", "", ErrInvalidMethod},
		{"fragment", "https://api.telegram.org/", "getMe#x", "", ErrInvalidMethod},
		{"nested path", "https://api.telegram.org/", "bot/getMe", "", ErrInvalidMethod},
		{"empty", "https://api.telegram.org/", "", "", ErrInvalidMethod},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tg := Telegram{URL: tt.base, Token: mockToken}
			got, err := tg.buildURL(tt.path)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTelegram_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		response   string
		is         error
		code       int
		retryAfter time.Duration
	}{
		{"unauthorized", 401, `{"ok":false,"error_code":401,"description":"Unauthorized"}`, ErrInvalidToken, 401, 0},
		{"not found", 404, `{"ok":false,"error_code":404,"description":"Not Found"}`, ErrInvalidToken, 404, 0},
		{"chat not found", 400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`, ErrInvalidChat, 400, 0},
		{"forbidden", 403, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`, ErrInvalidRequest, 403, 0},
		{"flood control", 429, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 7","parameters":{"retry_after":7}}`, ErrInvalidRequest, 429, 7 * time.Second},
		{"ok false with status 200", 200, `{"ok":false,"error_code":400,"description":"Bad Request"}`, ErrInvalidRequest, 400, 0},
		{"non-json error status", 502, `<html>Bad Gateway</html>`, ErrInvalidRequest, 502, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api, tg := newMockAPI(t)
			api.reply(tt.status, tt.response)

			body, err := tg.Get(context.Background(), "getMe")
			require.Nil(t, body)
			require.ErrorIs(t, err, tt.is)

			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			require.Equal(t, tt.code, apiErr.Code)
			require.Equal(t, tt.retryAfter, apiErr.RetryAfter)
		})
	}

	t.Run("non-json success", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(200, `not json`)
		_, err := tg.Get(context.Background(), "getMe")
		require.ErrorIs(t, err, ErrInvalidResult)
	})

	for _, response := range []string{`null`, `{}`, `{"result":true}`} {
		t.Run("missing ok "+response, func(t *testing.T) {
			api, tg := newMockAPI(t)
			api.reply(200, response)
			_, err := tg.Get(context.Background(), "getMe")
			require.ErrorIs(t, err, ErrInvalidResult)
			require.ErrorContains(t, err, `missing "ok" field`)
		})
	}

	t.Run("json without ok on error status", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(502, `{"error":"bad gateway"}`)
		_, err := tg.Get(context.Background(), "getMe")

		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		require.Equal(t, 502, apiErr.Code)
		require.Equal(t, "Bad Gateway", apiErr.Description)
	})

	t.Run("error words in successful response", func(t *testing.T) {
		api, tg := newMockAPI(t)
		api.reply(200, `{"ok":true,"result":{"text":"Unauthorized \"Not Found\" chat not found"}}`)
		_, err := tg.Get(context.Background(), "getMe")
		require.NoError(t, err)
	})
}

func TestTelegram_TokenNotLeaked(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	slow, _ := newSlowServer(t)

	tests := []struct {
		name string
		tg   Telegram
	}{
		{"connection refused", Telegram{URL: closed.URL, Token: mockToken}},
		{"invalid url", Telegram{URL: "http://[::1:bad/", Token: mockToken}},
		{"timeout", Telegram{URL: slow.URL, Token: mockToken, Timeout: 200 * time.Millisecond}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.tg.Post(context.Background(), "sendMessage", map[string]int{"chat_id": 1})
			require.Error(t, err)
			require.NotContains(t, err.Error(), "SECRET")
		})
	}
}

func TestTelegram_Redact(t *testing.T) {
	tg := Telegram{Token: mockToken}
	require.Equal(t, "https://x/bot<token>/getMe", tg.redact("https://x/bot"+mockToken+"/getMe"))

	empty := Telegram{}
	require.Equal(t, "https://x/bot/getMe", empty.redact("https://x/bot/getMe"))
}

func TestTelegram_ResponseTooLarge(t *testing.T) {
	api, tg := newMockAPI(t)
	api.reply(200, strings.Repeat("x", maxResponseSize+1))

	_, err := tg.Get(context.Background(), "getMe")
	require.ErrorIs(t, err, ErrResponseTooLarge)
}

func TestTelegram_NoRedirects(t *testing.T) {
	target, _ := newMockAPI(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/steal", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)

	tg := Telegram{URL: srv.URL, Token: mockToken}
	_, err := tg.Post(context.Background(), "sendMessage", map[string]string{"text": "secret"})
	require.Error(t, err)
	require.Zero(t, target.count(), "redirect must not be followed")
}

func TestTelegram_ContextCanceled(t *testing.T) {
	slow, calls := newSlowServer(t)
	tg := Telegram{URL: slow.URL, Token: mockToken}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := tg.Get(ctx, "getMe")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
	require.EqualValues(t, 1, calls.Load(), "no retry when the caller's context is done")
}

func TestTelegram_Retry(t *testing.T) {
	t.Run("get retries once on timeout", func(t *testing.T) {
		slow, calls := newSlowServer(t)
		tg := Telegram{URL: slow.URL, Token: mockToken, Timeout: 200 * time.Millisecond}
		_, err := tg.Get(context.Background(), "getMe")
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.EqualValues(t, 2, calls.Load())
	})

	t.Run("post does not retry", func(t *testing.T) {
		slow, calls := newSlowServer(t)
		tg := Telegram{URL: slow.URL, Token: mockToken, Timeout: 200 * time.Millisecond}
		_, err := tg.Post(context.Background(), "sendMessage", map[string]int{"chat_id": 1})
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.EqualValues(t, 1, calls.Load())
	})
}

func TestTelegram_Post(t *testing.T) {
	api, tg := newMockAPI(t)

	body, err := tg.Post(context.Background(), "setMyName", map[string]string{"name": "bot"})
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true,"result":true}`, string(body))

	req := api.last(t)
	require.Equal(t, http.MethodPost, req.Method)
	require.Equal(t, "/bot"+mockToken+"/setMyName", req.Path)
	require.JSONEq(t, `{"name":"bot"}`, req.Body)
}

func TestTelegram_GetQuery(t *testing.T) {
	api, tg := newMockAPI(t)

	_, err := tg.Get(context.Background(), "getUpdates?offset=5")
	require.NoError(t, err)

	req := api.last(t)
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, "/bot"+mockToken+"/getUpdates", req.Path)
	require.Equal(t, "offset=5", req.Query)
}

func TestTelegram_MarshalError(t *testing.T) {
	api, tg := newMockAPI(t)

	_, err := tg.Post(context.Background(), "sendMessage", map[string]any{"c": make(chan int)})
	require.ErrorContains(t, err, "marshal payload")
	require.Zero(t, api.count())
}

func TestTelegram_Concurrent(t *testing.T) {
	_, tg := newMockAPI(t)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := tg.Get(context.Background(), "getMe")
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	require.Zero(t, tg.Timeout, "doRequest must not modify the client")
}

func TestEscape(t *testing.T) {
	require.Equal(t,
		`a\_b\*c\[d\]\(e\)\~f\`+"`"+`g\>\#\+\-\=\|\{\}\.\!\\`,
		EscapeMarkdownV2(`a_b*c[d](e)~f`+"`"+`g>#+-=|{}.!\`))
	require.Equal(t, "plain text", EscapeMarkdownV2("plain text"))

	require.Equal(t, `&lt;b&gt;x&lt;/b&gt; &amp; y`, EscapeHTML(`<b>x</b> & y`))
}

func TestTelegram_InvalidResultCause(t *testing.T) {
	api, tg := newMockAPI(t)
	api.reply(200, `{"ok":true,"result":"not a message"}`)

	_, err := tg.SendMessage(context.Background(), 1, "hi")
	require.ErrorIs(t, err, ErrInvalidResult)
	require.ErrorContains(t, err, "cannot unmarshal")
}

func TestMenuButton(t *testing.T) {
	b := &MenuButton{Text: "a", URL: "https://x", CallbackData: "cb", SwitchInlineQuery: "q"}

	c := b.Copy()
	require.Equal(t, b, c)
	require.NotSame(t, b, c)
	require.True(t, b.Equal(c))

	for _, change := range []func(*MenuButton){
		func(m *MenuButton) { m.Text = "b" },
		func(m *MenuButton) { m.URL = "https://y" },
		func(m *MenuButton) { m.CallbackData = "other" },
		func(m *MenuButton) { m.SwitchInlineQuery = "other" },
	} {
		other := b.Copy()
		change(other)
		require.False(t, b.Equal(other))
	}

	var nilButton *MenuButton
	require.Nil(t, nilButton.Copy())
	require.False(t, b.Equal(nil))
	require.False(t, nilButton.Equal(b))
	require.True(t, nilButton.Equal(nil))
}
