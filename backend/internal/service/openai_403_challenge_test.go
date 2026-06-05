package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAI403HTMLChallengeDoesNotTriggerAccountFailover(t *testing.T) {
	svc := &OpenAIGatewayService{}

	require.False(t, svc.shouldFailoverOpenAIUpstreamResponse(
		http.StatusForbidden,
		"",
		[]byte(`<html><head><title>Just a moment...</title></head><body>Cloudflare challenge</body></html>`),
	))
}

func TestOpenAI403JSONStillTriggersAccountFailover(t *testing.T) {
	svc := &OpenAIGatewayService{}

	require.True(t, svc.shouldFailoverOpenAIUpstreamResponse(
		http.StatusForbidden,
		"workspace forbidden",
		[]byte(`{"error":{"message":"workspace forbidden"}}`),
	))
}
