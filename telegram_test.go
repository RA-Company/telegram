package telegram

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func GetApiData(t *testing.T) (url, token string) {
	url = os.Getenv("TG_API")
	require.NotEqual(t, "", url, "TG_API is not set")
	token = os.Getenv("TG_TOKEN")
	require.NotEqual(t, "", token, "TG_TOKEN is not set")

	return
}
