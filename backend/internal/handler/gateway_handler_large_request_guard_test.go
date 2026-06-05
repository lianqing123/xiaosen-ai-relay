package handler

import (
	"bytes"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestClaimLargeClaudeRequestDeduplicatesInFlight(t *testing.T) {
	clearLargeClaudeRequestInFlightForTest()
	t.Cleanup(clearLargeClaudeRequestInFlightForTest)

	body := bytes.Repeat([]byte("x"), largeClaudeRequestGuardMinBytes)
	parsed := &service.ParsedRequest{Model: "claude-opus-4.7"}

	release, ok := claimLargeClaudeRequest(354, parsed, body)
	require.True(t, ok)
	require.NotNil(t, release)

	duplicateRelease, ok := claimLargeClaudeRequest(354, parsed, body)
	require.False(t, ok)
	require.Nil(t, duplicateRelease)

	release()

	retryRelease, ok := claimLargeClaudeRequest(354, parsed, body)
	require.True(t, ok)
	require.NotNil(t, retryRelease)
	retryRelease()
}

func TestClaimLargeClaudeRequestSkipsSmallOrStreamingRequests(t *testing.T) {
	clearLargeClaudeRequestInFlightForTest()
	t.Cleanup(clearLargeClaudeRequestInFlightForTest)

	smallBody := bytes.Repeat([]byte("x"), largeClaudeRequestGuardMinBytes-1)
	largeBody := bytes.Repeat([]byte("x"), largeClaudeRequestGuardMinBytes)

	release, ok := claimLargeClaudeRequest(354, &service.ParsedRequest{Model: "claude-opus-4.7"}, smallBody)
	require.True(t, ok)
	require.Nil(t, release)

	release, ok = claimLargeClaudeRequest(354, &service.ParsedRequest{Model: "claude-opus-4.7", Stream: true}, largeBody)
	require.True(t, ok)
	require.Nil(t, release)

	release, ok = claimLargeClaudeRequest(354, &service.ParsedRequest{Model: "gpt-5.1"}, largeBody)
	require.True(t, ok)
	require.Nil(t, release)
}

func clearLargeClaudeRequestInFlightForTest() {
	largeClaudeRequestInFlight.Range(func(key, _ any) bool {
		largeClaudeRequestInFlight.Delete(key)
		return true
	})
}
