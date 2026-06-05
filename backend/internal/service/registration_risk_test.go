package service

import (
	"errors"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAssessRegistrationRisk_AllowsNormalRegistrationBelowThresholds(t *testing.T) {
	decision := assessRegistrationRisk(RegistrationRiskCheckInput{
		Email:    "normal@example.com",
		ClientIP: "8.8.8.8",
	}, registrationRiskSnapshot{
		IPHourSuccesses:      4,
		IPDaySuccesses:       10,
		ShapeIPHourSuccesses: 10,
		ShapeIPDaySuccesses:  10,
	})

	require.False(t, decision.Blocked)
	require.Equal(t, "normal", decision.EmailShape)
}

func TestAssessRegistrationRisk_BlocksIPHourlyLimit(t *testing.T) {
	decision := assessRegistrationRisk(RegistrationRiskCheckInput{
		Email:    "normal@example.com",
		ClientIP: "8.8.8.8",
	}, registrationRiskSnapshot{
		IPHourSuccesses: 5,
	})

	require.True(t, decision.Blocked)
	require.Equal(t, "ip_hourly_limit", decision.Reason)
	require.True(t, errors.Is(decision.Err, ErrRegistrationRiskLimited))
	require.Equal(t, 429, infraerrors.Code(decision.Err))
}

func TestAssessRegistrationRisk_BlocksHighRiskEmailShapeBurst(t *testing.T) {
	decision := assessRegistrationRisk(RegistrationRiskCheckInput{
		Email:    "123456789@qq.com",
		ClientIP: "8.8.8.8",
	}, registrationRiskSnapshot{
		ShapeIPHourSuccesses: 2,
	})

	require.True(t, decision.Blocked)
	require.Equal(t, "email_shape_hourly_limit", decision.Reason)
	require.Equal(t, "numeric_qq", decision.EmailShape)
}

func TestAssessRegistrationRisk_DoesNotApplyShapeLimitToNormalEmail(t *testing.T) {
	decision := assessRegistrationRisk(RegistrationRiskCheckInput{
		Email:    "person@example.com",
		ClientIP: "8.8.8.8",
	}, registrationRiskSnapshot{
		ShapeIPHourSuccesses: 2,
		ShapeIPDaySuccesses:  3,
	})

	require.False(t, decision.Blocked)
	require.Equal(t, "normal", decision.EmailShape)
}
