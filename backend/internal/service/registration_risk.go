package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var ErrRegistrationRiskLimited = infraerrors.TooManyRequests("REGISTRATION_RISK_LIMITED", "registration rate limit exceeded")

const (
	registrationRiskIPHourlyLimit         = 5
	registrationRiskIPDailyLimit          = 12
	registrationRiskEmailShapeHourlyLimit = 2
	registrationRiskEmailShapeDailyLimit  = 3
)

type RegistrationRiskCheckInput struct {
	Email     string
	ClientIP  string
	UserAgent string
}

type RegistrationRiskRecordInput struct {
	RegistrationRiskCheckInput
	UserID int64
	Status string
	Reason string
}

type registrationRiskSnapshot struct {
	IPHourSuccesses      int
	IPDaySuccesses       int
	ShapeIPHourSuccesses int
	ShapeIPDaySuccesses  int
}

type registrationRiskDecision struct {
	Blocked    bool
	Err        error
	Reason     string
	EmailShape string
}

func assessRegistrationRisk(input RegistrationRiskCheckInput, snapshot registrationRiskSnapshot) registrationRiskDecision {
	shape := registrationEmailShape(input.Email)
	decision := registrationRiskDecision{EmailShape: shape}

	if snapshot.IPHourSuccesses >= registrationRiskIPHourlyLimit {
		decision.Blocked = true
		decision.Reason = "ip_hourly_limit"
		decision.Err = ErrRegistrationRiskLimited.WithMetadata(map[string]string{"reason": decision.Reason})
		return decision
	}
	if snapshot.IPDaySuccesses >= registrationRiskIPDailyLimit {
		decision.Blocked = true
		decision.Reason = "ip_daily_limit"
		decision.Err = ErrRegistrationRiskLimited.WithMetadata(map[string]string{"reason": decision.Reason})
		return decision
	}

	if isHighRiskRegistrationEmailShape(shape) {
		if snapshot.ShapeIPHourSuccesses >= registrationRiskEmailShapeHourlyLimit {
			decision.Blocked = true
			decision.Reason = "email_shape_hourly_limit"
			decision.Err = ErrRegistrationRiskLimited.WithMetadata(map[string]string{"reason": decision.Reason, "email_shape": shape})
			return decision
		}
		if snapshot.ShapeIPDaySuccesses >= registrationRiskEmailShapeDailyLimit {
			decision.Blocked = true
			decision.Reason = "email_shape_daily_limit"
			decision.Err = ErrRegistrationRiskLimited.WithMetadata(map[string]string{"reason": decision.Reason, "email_shape": shape})
			return decision
		}
	}

	return decision
}

func registrationEmailShape(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" || domain == "" {
		return "invalid"
	}
	if domain == "qq.com" && isAllDigits(local) {
		return "numeric_qq"
	}
	switch domain {
	case "tentwind.top", "hidevak.com", "2925.com":
		return "throwaway_domain"
	}
	if isAllDigits(local) && len(local) >= 4 {
		return "numeric_local"
	}
	return "normal"
}

func isHighRiskRegistrationEmailShape(shape string) bool {
	switch shape {
	case "numeric_qq", "throwaway_domain":
		return true
	default:
		return false
	}
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeRegistrationRiskIP(clientIP string) string {
	clientIP = strings.TrimSpace(clientIP)
	if parsed := net.ParseIP(clientIP); parsed != nil {
		return parsed.String()
	}
	return ""
}

func (s *AuthService) CheckRegistrationRisk(ctx context.Context, input RegistrationRiskCheckInput) error {
	if s == nil || s.entClient == nil {
		return nil
	}
	clientIP := normalizeRegistrationRiskIP(input.ClientIP)
	if clientIP == "" {
		return nil
	}

	shape := registrationEmailShape(input.Email)
	snapshot, err := s.loadRegistrationRiskSnapshot(ctx, clientIP, shape)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[RegistrationRisk] fail-open snapshot error ip=%s shape=%s err=%v", clientIP, shape, err)
		return nil
	}

	decision := assessRegistrationRisk(RegistrationRiskCheckInput{
		Email:    input.Email,
		ClientIP: clientIP,
	}, snapshot)
	if !decision.Blocked {
		return nil
	}
	s.RecordRegistrationRiskEvent(ctx, RegistrationRiskRecordInput{
		RegistrationRiskCheckInput: RegistrationRiskCheckInput{
			Email:     input.Email,
			ClientIP:  clientIP,
			UserAgent: input.UserAgent,
		},
		Status: "blocked",
		Reason: decision.Reason,
	})
	return decision.Err
}

func (s *AuthService) loadRegistrationRiskSnapshot(ctx context.Context, clientIP, emailShape string) (registrationRiskSnapshot, error) {
	var snapshot registrationRiskSnapshot
	rows, err := s.entClient.QueryContext(ctx, `
SELECT
  COUNT(*) FILTER (WHERE created_at >= now() - interval '1 hour') AS ip_hour_successes,
  COUNT(*) FILTER (WHERE created_at >= now() - interval '24 hours') AS ip_day_successes,
  COUNT(*) FILTER (WHERE email_shape = $2 AND created_at >= now() - interval '1 hour') AS shape_ip_hour_successes,
  COUNT(*) FILTER (WHERE email_shape = $2 AND created_at >= now() - interval '24 hours') AS shape_ip_day_successes
FROM registration_risk_events
WHERE status = 'success'
  AND client_ip = $1::inet
  AND created_at >= now() - interval '24 hours'
`, clientIP, strings.TrimSpace(emailShape))
	if err != nil {
		return snapshot, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return snapshot, rows.Err()
	}
	if err := rows.Scan(
		&snapshot.IPHourSuccesses,
		&snapshot.IPDaySuccesses,
		&snapshot.ShapeIPHourSuccesses,
		&snapshot.ShapeIPDaySuccesses,
	); err != nil {
		return snapshot, err
	}
	return snapshot, rows.Err()
}

func (s *AuthService) RecordRegistrationRiskEvent(ctx context.Context, input RegistrationRiskRecordInput) {
	if s == nil || s.entClient == nil {
		return
	}
	clientIP := normalizeRegistrationRiskIP(input.ClientIP)
	if clientIP == "" {
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "success"
	}
	domain := ""
	if _, d, ok := strings.Cut(email, "@"); ok {
		domain = d
	}
	metadata, _ := json.Marshal(map[string]any{
		"user_agent": strings.TrimSpace(input.UserAgent),
	})

	rows, err := s.entClient.QueryContext(ctx, `
INSERT INTO registration_risk_events (
  created_at, user_id, email, email_domain, email_shape, client_ip, status, reason, metadata
) VALUES (
  $1, NULLIF($2, 0), $3, $4, $5, $6::inet, $7, $8, $9::jsonb
)
RETURNING id
`,
		time.Now().UTC(),
		input.UserID,
		email,
		domain,
		registrationEmailShape(email),
		clientIP,
		status,
		strings.TrimSpace(input.Reason),
		string(metadata),
	)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[RegistrationRisk] audit insert failed ip=%s email=%s status=%s err=%v", clientIP, email, status, err)
		return
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			logger.LegacyPrintf("service.auth", "[RegistrationRisk] audit scan failed ip=%s email=%s status=%s err=%v", clientIP, email, status, err)
			return
		}
	}
	if err := rows.Err(); err != nil {
		logger.LegacyPrintf("service.auth", "[RegistrationRisk] audit rows failed ip=%s email=%s status=%s err=%v", clientIP, email, status, err)
	}
}

func FormatRegistrationRiskLimits() string {
	return fmt.Sprintf("ip=%d/hour,%d/day; high_risk_email_shape=%d/hour,%d/day",
		registrationRiskIPHourlyLimit,
		registrationRiskIPDailyLimit,
		registrationRiskEmailShapeHourlyLimit,
		registrationRiskEmailShapeDailyLimit,
	)
}
