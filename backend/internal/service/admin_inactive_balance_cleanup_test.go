//go:build unit

package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeInactiveBalanceCleanupRequestDefaults(t *testing.T) {
	req, err := NormalizeInactiveBalanceCleanupRequest(InactiveBalanceCleanupRequest{})
	require.NoError(t, err)
	require.Equal(t, 3, req.Days)
	require.Equal(t, 50, req.Limit)
}

func TestNormalizeInactiveBalanceCleanupRequestRejectsUnsafeValues(t *testing.T) {
	_, err := NormalizeInactiveBalanceCleanupRequest(InactiveBalanceCleanupRequest{Days: 2, Limit: 50})
	require.Error(t, err)

	_, err = NormalizeInactiveBalanceCleanupRequest(InactiveBalanceCleanupRequest{Days: 3, Limit: 1001})
	require.Error(t, err)
}

func TestBuildInactiveBalanceCleanupAuditNote(t *testing.T) {
	cutoff := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	lastUsedAt := cutoff.Add(-time.Hour)

	note := buildInactiveBalanceCleanupAuditNote(72, cutoff, &lastUsedAt, 9)

	require.Contains(t, note, "inactive_balance_cleanup")
	require.Contains(t, note, "hours=72")
	require.Contains(t, note, "operator=9")
	require.True(t, strings.Contains(note, "last_used_at=2026-05-01T11:00:00Z"))
}

func TestInactiveBalanceCleanupExcludesPaidBalanceUsers(t *testing.T) {
	eligibility := inactiveBalanceCleanupEligibilityCTE()

	require.Contains(t, eligibility, "FROM payment_orders po")
	require.Contains(t, eligibility, "po.user_id = u.id")
	require.Contains(t, eligibility, "po.status = 'COMPLETED'")
	require.Contains(t, eligibility, "po.order_type = 'balance'")
}
