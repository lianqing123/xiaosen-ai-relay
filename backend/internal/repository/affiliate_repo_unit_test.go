package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestListAffiliateInviteeRebatesWithClient_ReturnsRechargeAndRebateTotals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 5, 8, 10, 30, 0, 0, time.UTC)
	lastRechargeAt := time.Date(2026, 5, 8, 11, 45, 0, 0, time.UTC)

	mock.ExpectQuery("(?s)SELECT COUNT\\(\\*\\).*FROM user_affiliates ua.*WHERE ua.inviter_id IS NOT NULL").
		WithArgs("%alice%", "%alice%", "%alice%", "%alice%", "%alice%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))

	mock.ExpectQuery("(?s)SELECT ua\\.inviter_id.*SUM\\(CASE\\s+WHEN order_type = 'balance'.*pay_amount / \\(1 \\+ GREATEST\\(fee_rate, 0\\) / 100\\).*WHERE order_type IN \\('balance', 'subscription'\\).*status = 'COMPLETED'").
		WithArgs("%alice%", "%alice%", "%alice%", "%alice%", "%alice%", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"inviter_id",
			"inviter_email",
			"inviter_username",
			"inviter_aff_code",
			"invitee_id",
			"invitee_email",
			"invitee_username",
			"invitee_created_at",
			"total_recharged_amount",
			"total_paid_amount",
			"total_rebate_amount",
			"last_recharge_at",
		}).AddRow(
			int64(10),
			"inviter@example.com",
			"inviter",
			"VIP2026",
			int64(20),
			"alice@example.com",
			"alice",
			createdAt,
			float64(188.5),
			float64(190.1),
			float64(37.7),
			lastRechargeAt,
		))

	items, total, err := listAffiliateInviteeRebatesWithClient(
		context.Background(),
		db,
		service.AffiliateInviteeRebateFilter{Search: "alice", Page: 1, PageSize: 20},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)

	got := items[0]
	require.Equal(t, int64(10), got.InviterID)
	require.Equal(t, "inviter@example.com", got.InviterEmail)
	require.Equal(t, "VIP2026", got.InviterAffCode)
	require.Equal(t, int64(20), got.InviteeID)
	require.Equal(t, "alice@example.com", got.InviteeEmail)
	require.Equal(t, createdAt, got.InviteeCreatedAt)
	require.InDelta(t, 188.5, got.TotalRechargedAmount, 1e-9)
	require.InDelta(t, 190.1, got.TotalPaidAmount, 1e-9)
	require.InDelta(t, 37.7, got.TotalRebateAmount, 1e-9)
	require.NotNil(t, got.LastRechargeAt)
	require.Equal(t, lastRechargeAt, *got.LastRechargeAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAffiliateInviteesWithClient_ReturnsRechargeTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	createdAt := time.Date(2026, 5, 8, 9, 30, 0, 0, time.UTC)

	mock.ExpectQuery("(?s)SELECT ua\\.user_id.*SUM\\(CASE\\s+WHEN order_type = 'balance'.*pay_amount / \\(1 \\+ GREATEST\\(fee_rate, 0\\) / 100\\).*WHERE order_type IN \\('balance', 'subscription'\\).*status = 'COMPLETED'").
		WithArgs(int64(10), 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id",
			"email",
			"username",
			"created_at",
			"total_rebate",
			"total_recharged_amount",
		}).AddRow(
			int64(20),
			"invitee@example.com",
			"invitee",
			createdAt,
			float64(12.5),
			float64(188.5),
		))

	items, err := listAffiliateInviteesWithClient(context.Background(), db, 10, 20)
	require.NoError(t, err)
	require.Len(t, items, 1)

	got := items[0]
	require.Equal(t, int64(20), got.UserID)
	require.Equal(t, "invitee@example.com", got.Email)
	require.Equal(t, "invitee", got.Username)
	require.NotNil(t, got.CreatedAt)
	require.Equal(t, createdAt, *got.CreatedAt)
	require.InDelta(t, 12.5, got.TotalRebate, 1e-9)
	require.InDelta(t, 188.5, got.TotalRechargedAmount, 1e-9)
	require.NoError(t, mock.ExpectationsWereMet())
}
