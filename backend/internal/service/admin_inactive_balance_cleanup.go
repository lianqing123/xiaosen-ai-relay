package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	inactiveBalanceCleanupDefaultDays  = 3
	inactiveBalanceCleanupDefaultLimit = 50
	inactiveBalanceCleanupMaxLimit     = 1000

	InactiveBalanceCleanupConfirmToken = "CLEAR_INACTIVE_BALANCES"
)

type InactiveBalanceCleanupRequest struct {
	Days  int `json:"days"`
	Limit int `json:"limit"`
}

type InactiveBalanceCleanupCandidate struct {
	ID            int64      `json:"id"`
	Email         string     `json:"email"`
	Balance       float64    `json:"balance"`
	CreatedAt     time.Time  `json:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	InactiveSince time.Time  `json:"inactive_since"`
}

type InactiveBalanceCleanupPreview struct {
	Days         int                               `json:"days"`
	CutoffAt     time.Time                         `json:"cutoff_at"`
	Count        int64                             `json:"count"`
	TotalBalance float64                           `json:"total_balance"`
	Candidates   []InactiveBalanceCleanupCandidate `json:"candidates"`
}

type InactiveBalanceCleanupResult struct {
	Days           int                               `json:"days"`
	CutoffAt       time.Time                         `json:"cutoff_at"`
	ClearedCount   int64                             `json:"cleared_count"`
	ClearedBalance float64                           `json:"cleared_balance"`
	Records        []InactiveBalanceCleanupCandidate `json:"records"`
}

func NormalizeInactiveBalanceCleanupRequest(req InactiveBalanceCleanupRequest) (InactiveBalanceCleanupRequest, error) {
	if req.Days == 0 {
		req.Days = inactiveBalanceCleanupDefaultDays
	}
	if req.Limit == 0 {
		req.Limit = inactiveBalanceCleanupDefaultLimit
	}
	if req.Days < inactiveBalanceCleanupDefaultDays {
		return req, infraerrors.BadRequest("INVALID_INACTIVE_DAYS", "inactive cleanup days must be at least 3")
	}
	if req.Days > 365 {
		return req, infraerrors.BadRequest("INVALID_INACTIVE_DAYS", "inactive cleanup days must be less than or equal to 365")
	}
	if req.Limit < 1 || req.Limit > inactiveBalanceCleanupMaxLimit {
		return req, infraerrors.BadRequest("INVALID_INACTIVE_LIMIT", "inactive cleanup limit must be between 1 and 1000")
	}
	return req, nil
}

func (s *adminServiceImpl) PreviewInactiveBalanceCleanup(ctx context.Context, req InactiveBalanceCleanupRequest) (*InactiveBalanceCleanupPreview, error) {
	normalized, err := NormalizeInactiveBalanceCleanupRequest(req)
	if err != nil {
		return nil, err
	}
	db, err := s.inactiveBalanceCleanupDB()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().UTC().Add(-time.Duration(normalized.Days) * 24 * time.Hour)

	count, totalBalance, err := queryInactiveBalanceCleanupSummary(ctx, db, cutoff)
	if err != nil {
		return nil, err
	}
	candidates, err := queryInactiveBalanceCleanupCandidates(ctx, db, cutoff, normalized.Limit)
	if err != nil {
		return nil, err
	}

	return &InactiveBalanceCleanupPreview{
		Days:         normalized.Days,
		CutoffAt:     cutoff,
		Count:        count,
		TotalBalance: totalBalance,
		Candidates:   candidates,
	}, nil
}

func (s *adminServiceImpl) ExecuteInactiveBalanceCleanup(ctx context.Context, req InactiveBalanceCleanupRequest, operatorID int64) (*InactiveBalanceCleanupResult, error) {
	normalized, err := NormalizeInactiveBalanceCleanupRequest(req)
	if err != nil {
		return nil, err
	}
	db, err := s.inactiveBalanceCleanupDB()
	if err != nil {
		return nil, err
	}
	cutoff := time.Now().UTC().Add(-time.Duration(normalized.Days) * 24 * time.Hour)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin inactive balance cleanup transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	records, err := executeInactiveBalanceCleanupUpdate(ctx, tx, cutoff, normalized.Limit)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	for i := range records {
		code, err := GenerateRedeemCode()
		if err != nil {
			return nil, fmt.Errorf("generate inactive balance cleanup audit code: %w", err)
		}
		note := buildInactiveBalanceCleanupAuditNote(normalized.Days*24, cutoff, records[i].LastUsedAt, operatorID)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`,
			code,
			AdjustmentTypeAdminBalance,
			-records[i].Balance,
			StatusUsed,
			records[i].ID,
			now,
			note,
			now,
		); err != nil {
			return nil, fmt.Errorf("create inactive balance cleanup audit record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit inactive balance cleanup transaction: %w", err)
	}

	var clearedBalance float64
	userIDs := make([]int64, 0, len(records))
	for i := range records {
		clearedBalance += records[i].Balance
		userIDs = append(userIDs, records[i].ID)
	}
	s.invalidateInactiveBalanceCleanupCaches(ctx, userIDs)

	return &InactiveBalanceCleanupResult{
		Days:           normalized.Days,
		CutoffAt:       cutoff,
		ClearedCount:   int64(len(records)),
		ClearedBalance: clearedBalance,
		Records:        records,
	}, nil
}

func (s *adminServiceImpl) inactiveBalanceCleanupDB() (*sql.DB, error) {
	if s == nil || s.entClient == nil {
		return nil, infraerrors.InternalServer("INACTIVE_BALANCE_CLEANUP_UNAVAILABLE", "database client is unavailable")
	}
	drv, ok := s.entClient.Driver().(*entsql.Driver)
	if !ok || drv.DB() == nil {
		return nil, infraerrors.InternalServer("INACTIVE_BALANCE_CLEANUP_UNAVAILABLE", "database driver is unavailable")
	}
	return drv.DB(), nil
}

func inactiveBalanceCleanupEligibilityCTE() string {
	return `
WITH last_usage AS (
	SELECT user_id, MAX(created_at) AS last_used_at
	FROM usage_logs
	GROUP BY user_id
),
eligible AS (
	SELECT
		u.id,
		u.email,
		u.balance,
		u.created_at,
		lu.last_used_at,
		COALESCE(lu.last_used_at, u.created_at) AS inactive_since
	FROM users u
	LEFT JOIN last_usage lu ON lu.user_id = u.id
	WHERE u.deleted_at IS NULL
		AND u.role = $2
		AND u.status = $3
		AND u.balance > 0
		AND COALESCE(lu.last_used_at, u.created_at) <= $1
		AND NOT EXISTS (
			SELECT 1
			FROM user_subscriptions us
			WHERE us.user_id = u.id
				AND us.deleted_at IS NULL
				AND us.status = $4
				AND us.expires_at > NOW()
		)
` + inactiveBalanceCleanupPaidBalanceExclusionSQL() + `
)`
}

func inactiveBalanceCleanupPaidBalanceExclusionSQL() string {
	return `
		AND NOT EXISTS (
			SELECT 1
			FROM payment_orders po
			WHERE po.user_id = u.id
				AND po.status = 'COMPLETED'
				AND po.order_type = 'balance'
		)`
}

func queryInactiveBalanceCleanupSummary(ctx context.Context, db *sql.DB, cutoff time.Time) (int64, float64, error) {
	var count int64
	var totalBalance sql.NullFloat64
	query := inactiveBalanceCleanupEligibilityCTE() + `
SELECT COUNT(*), COALESCE(SUM(balance), 0)
FROM eligible
`
	if err := db.QueryRowContext(ctx, query, cutoff, RoleUser, StatusActive, SubscriptionStatusActive).Scan(&count, &totalBalance); err != nil {
		return 0, 0, fmt.Errorf("query inactive balance cleanup summary: %w", err)
	}
	return count, totalBalance.Float64, nil
}

func queryInactiveBalanceCleanupCandidates(ctx context.Context, db *sql.DB, cutoff time.Time, limit int) ([]InactiveBalanceCleanupCandidate, error) {
	query := inactiveBalanceCleanupEligibilityCTE() + `
SELECT id, email, balance, created_at, last_used_at, inactive_since
FROM eligible
ORDER BY inactive_since ASC, id ASC
LIMIT $5
`
	rows, err := db.QueryContext(ctx, query, cutoff, RoleUser, StatusActive, SubscriptionStatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("query inactive balance cleanup candidates: %w", err)
	}
	defer rows.Close()
	return scanInactiveBalanceCleanupRows(rows)
}

func executeInactiveBalanceCleanupUpdate(ctx context.Context, tx *sql.Tx, cutoff time.Time, limit int) ([]InactiveBalanceCleanupCandidate, error) {
	query := `
WITH last_usage AS (
	SELECT user_id, MAX(created_at) AS last_used_at
	FROM usage_logs
	GROUP BY user_id
),
eligible AS (
	SELECT
		u.id,
		u.email,
		u.balance,
		u.created_at,
		lu.last_used_at,
		COALESCE(lu.last_used_at, u.created_at) AS inactive_since
	FROM users u
	LEFT JOIN last_usage lu ON lu.user_id = u.id
	WHERE u.deleted_at IS NULL
		AND u.role = $2
		AND u.status = $3
		AND u.balance > 0
		AND COALESCE(lu.last_used_at, u.created_at) <= $1
		AND NOT EXISTS (
			SELECT 1
			FROM user_subscriptions us
			WHERE us.user_id = u.id
				AND us.deleted_at IS NULL
				AND us.status = $4
				AND us.expires_at > NOW()
		)
` + inactiveBalanceCleanupPaidBalanceExclusionSQL() + `
	ORDER BY inactive_since ASC, u.id ASC
	LIMIT $5
	FOR UPDATE OF u SKIP LOCKED
),
updated AS (
	UPDATE users u
	SET balance = 0, updated_at = NOW()
	FROM eligible e
	WHERE u.id = e.id AND u.balance > 0
	RETURNING u.id, e.email, e.balance, e.created_at, e.last_used_at, e.inactive_since
)
SELECT id, email, balance, created_at, last_used_at, inactive_since
FROM updated
ORDER BY inactive_since ASC, id ASC
`
	rows, err := tx.QueryContext(ctx, query, cutoff, RoleUser, StatusActive, SubscriptionStatusActive, limit)
	if err != nil {
		return nil, fmt.Errorf("execute inactive balance cleanup update: %w", err)
	}
	defer rows.Close()
	return scanInactiveBalanceCleanupRows(rows)
}

type inactiveBalanceCleanupRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanInactiveBalanceCleanupRows(rows inactiveBalanceCleanupRows) ([]InactiveBalanceCleanupCandidate, error) {
	var candidates []InactiveBalanceCleanupCandidate
	for rows.Next() {
		var candidate InactiveBalanceCleanupCandidate
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&candidate.ID,
			&candidate.Email,
			&candidate.Balance,
			&candidate.CreatedAt,
			&lastUsedAt,
			&candidate.InactiveSince,
		); err != nil {
			return nil, fmt.Errorf("scan inactive balance cleanup row: %w", err)
		}
		if lastUsedAt.Valid {
			t := lastUsedAt.Time
			candidate.LastUsedAt = &t
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inactive balance cleanup rows: %w", err)
	}
	if candidates == nil {
		candidates = []InactiveBalanceCleanupCandidate{}
	}
	return candidates, nil
}

func buildInactiveBalanceCleanupAuditNote(hours int, cutoff time.Time, lastUsedAt *time.Time, operatorID int64) string {
	lastUsed := "null"
	if lastUsedAt != nil {
		lastUsed = lastUsedAt.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"inactive_balance_cleanup: hours=%d cutoff=%s last_used_at=%s operator=%d",
		hours,
		cutoff.UTC().Format(time.RFC3339),
		lastUsed,
		operatorID,
	)
}

func (s *adminServiceImpl) invalidateInactiveBalanceCleanupCaches(ctx context.Context, userIDs []int64) {
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if s.authCacheInvalidator != nil {
			s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
		}
		if s.billingCacheService != nil {
			go func(id int64) {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, id); err != nil {
					logger.LegacyPrintf("service.admin", "invalidate user balance cache failed after inactive cleanup: user_id=%d err=%v", id, err)
				}
			}(userID)
		}
	}
}
