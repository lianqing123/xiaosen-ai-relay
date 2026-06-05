package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKiroGatewayAdminServiceImportRefreshTokenRedactsStatus(t *testing.T) {
	home := t.TempDir()
	svc := NewKiroGatewayAdminServiceWithHome(home)

	result, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:          "refresh_token",
		RefreshToken:  "refresh-token-secret",
		TwoFactorCode: "123456",
		Region:        "us-east-1",
	})
	if err != nil {
		t.Fatalf("ImportCredentials() error = %v", err)
	}
	if result.CredentialCount != 1 {
		t.Fatalf("CredentialCount = %d, want 1", result.CredentialCount)
	}
	if !result.TwoFactorAccepted {
		t.Fatalf("TwoFactorAccepted = false, want true")
	}

	raw, err := os.ReadFile(filepath.Join(home, "secrets", "credentials.json"))
	if err != nil {
		t.Fatalf("read credentials.json: %v", err)
	}
	if !strings.Contains(string(raw), "refresh-token-secret") {
		t.Fatalf("credentials.json should contain the refresh token")
	}
	if strings.Contains(string(raw), "123456") {
		t.Fatalf("credentials.json must not persist the 2FA code: %s", raw)
	}

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.CredentialCount != 1 {
		t.Fatalf("CredentialCount = %d, want 1", status.CredentialCount)
	}
	if len(status.Credentials) != 1 {
		t.Fatalf("len(Credentials) = %d, want 1", len(status.Credentials))
	}
	entry := status.Credentials[0]
	if entry.RefreshTokenSHA256 == "" {
		t.Fatalf("RefreshTokenSHA256 is empty")
	}
	encoded, _ := json.Marshal(status)
	if strings.Contains(string(encoded), "refresh-token-secret") {
		t.Fatalf("Status() must not expose refresh token: %s", encoded)
	}
}

func TestKiroGatewayAdminServiceStatusFetchesOfficialCreditQuota(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "secrets"), 0o755); err != nil {
		t.Fatalf("mkdir secrets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "secrets", "credentials.json"), []byte(`[
		{"type":"refresh_token","refresh_token":"refresh-one","enabled":true,"region":"us-east-1"},
		{"type":"refresh_token","refresh_token":"refresh-two","enabled":true,"region":"us-east-1"}
	]`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	refreshCalls := 0
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshCalls++
		if r.Method != http.MethodPost {
			t.Fatalf("refresh method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"accessToken":"access-token","expiresIn":3600,"profileArn":"arn:aws:codewhisperer:us-east-1:123456789012:profile/test"}`))
	}))
	defer refreshServer.Close()

	usageCalls := 0
	usageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		usageCalls++
		if got := r.Header.Get("x-amz-target"); got != "AmazonCodeWhispererService.GetUsageLimits" {
			t.Fatalf("x-amz-target = %q, want GetUsageLimits", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode usage body: %v", err)
		}
		if got := body["resourceType"]; got != "AGENTIC_REQUEST" {
			t.Fatalf("resourceType = %v, want AGENTIC_REQUEST", got)
		}
		if body["profileArn"] == "" {
			t.Fatalf("profileArn should be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"nextDateReset": 1780272000,
			"subscriptionInfo": {"subscriptionTitle":"KIRO PRO","type":"Q_DEVELOPER_STANDALONE_PRO"},
			"overageConfiguration": {"overageStatus":"DISABLED"},
			"usageBreakdownList": [{
				"resourceType":"CREDIT",
				"displayName":"Credit",
				"displayNamePlural":"Credits",
				"currency":"USD",
				"currentUsage":225,
				"currentUsageWithPrecision":225.09,
				"usageLimit":1000,
				"usageLimitWithPrecision":1000,
				"overageRate":0.04
			}],
			"userInfo": {"userId":"user-1"}
		}`))
	}))
	defer usageServer.Close()

	t.Setenv("KIRO_DESKTOP_AUTH_BASE_URL", refreshServer.URL)
	t.Setenv("KIRO_CODEWHISPERER_USAGE_URL", usageServer.URL)

	svc := NewKiroGatewayAdminServiceWithHome(home)
	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if refreshCalls != 2 {
		t.Fatalf("refresh calls = %d, want 2", refreshCalls)
	}
	if usageCalls != 2 {
		t.Fatalf("usage calls = %d, want 2", usageCalls)
	}
	if status.OfficialQuota == nil || !status.OfficialQuota.Available {
		t.Fatalf("OfficialQuota unavailable: %#v", status.OfficialQuota)
	}
	if got := status.OfficialQuota.CurrentUsage; got != 225.09 {
		t.Fatalf("CurrentUsage = %v, want 225.09", got)
	}
	if got := status.OfficialQuota.UsageLimit; got != 1000 {
		t.Fatalf("UsageLimit = %v, want 1000", got)
	}
	if got := status.OfficialQuota.Remaining; got != 774.91 {
		t.Fatalf("Remaining = %v, want 774.91", got)
	}
	if got := status.OfficialQuota.SubscriptionTitle; got != "KIRO PRO" {
		t.Fatalf("SubscriptionTitle = %q, want KIRO PRO", got)
	}
	if got := status.OfficialQuota.UniqueUserCount; got != 1 {
		t.Fatalf("UniqueUserCount = %d, want deduped 1", got)
	}
}

func TestKiroGatewayAdminServiceImportJSONCopiesToContainerPath(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(t.TempDir(), "kiro-auth-token.json")
	if err := os.WriteFile(source, []byte(`{"accessToken":"json-access","refreshToken":"json-refresh","expiresAt":"2026-01-12T23:00:00.000Z","profileArn":"arn:aws:codewhisperer:us-east-1:123456789012:profile/test","region":"us-east-1"}`), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	svc := NewKiroGatewayAdminServiceWithHome(home)
	result, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:     "json",
		FileName: "kiro-auth-token.json",
		FilePath: source,
	})
	if err != nil {
		t.Fatalf("ImportCredentials() error = %v", err)
	}
	if result.CredentialCount != 1 {
		t.Fatalf("CredentialCount = %d, want 1", result.CredentialCount)
	}
	if len(result.Credentials) != 1 {
		t.Fatalf("len(Credentials) = %d, want 1", len(result.Credentials))
	}
	if !strings.HasPrefix(result.Credentials[0].Path, "/app/secrets/imports/") {
		t.Fatalf("credential path = %q, want container imports path", result.Credentials[0].Path)
	}

	imports, err := os.ReadDir(filepath.Join(home, "secrets", "imports"))
	if err != nil {
		t.Fatalf("read imports dir: %v", err)
	}
	if len(imports) != 1 {
		t.Fatalf("imported files = %d, want 1", len(imports))
	}
}

func TestKiroGatewayAdminServiceImportJSONAcceptsAWSOIDCTokenFile(t *testing.T) {
	home := t.TempDir()
	source := filepath.Join(t.TempDir(), "aws-sso-token.json")
	if err := os.WriteFile(source, []byte(`{"accessToken":"json-access","refreshToken":"json-refresh","expiresAt":"2026-01-12T23:00:00Z","region":"us-east-1","clientId":"client-id","clientSecret":"client-secret"}`), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	svc := NewKiroGatewayAdminServiceWithHome(home)
	result, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:     "json",
		FileName: "aws-sso-token.json",
		FilePath: source,
	})
	if err != nil {
		t.Fatalf("ImportCredentials() error = %v", err)
	}
	if result.CredentialCount != 1 {
		t.Fatalf("CredentialCount = %d, want 1", result.CredentialCount)
	}
	if got := result.Credentials[0].Type; got != "json" {
		t.Fatalf("credential type = %q, want json", got)
	}
}

func TestKiroGatewayAdminServiceImportJSONAcceptsGoogleProviderRefreshTokenFile(t *testing.T) {
	source := filepath.Join(t.TempDir(), "loose-token.json")
	if err := os.WriteFile(source, []byte(`{"refreshToken":"json-refresh","provider":"Google"}`), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	svc := NewKiroGatewayAdminServiceWithHome(t.TempDir())
	result, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:     "json",
		FileName: "loose-token.json",
		FilePath: source,
	})
	if err != nil {
		t.Fatalf("ImportCredentials() error = %v", err)
	}
	if got := result.Credentials[0].Type; got != "json" {
		t.Fatalf("credential type = %q, want json", got)
	}
}

func TestKiroGatewayAdminServiceImportJSONRejectsSnakeCaseTokenFile(t *testing.T) {
	source := filepath.Join(t.TempDir(), "snake-token.json")
	if err := os.WriteFile(source, []byte(`{"access_token":"json-access","refresh_token":"json-refresh","expires_at":"2026-01-12T23:00:00Z","region":"us-east-1","client_id":"client-id","client_secret":"client-secret"}`), 0o600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	svc := NewKiroGatewayAdminServiceWithHome(t.TempDir())
	_, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:     "json",
		FileName: "snake-token.json",
		FilePath: source,
	})
	if err == nil {
		t.Fatalf("ImportCredentials() error = nil, want camelCase error")
	}
	if !strings.Contains(err.Error(), "camelCase") {
		t.Fatalf("error = %q, want mention camelCase", err.Error())
	}
}

func TestKiroGatewayAdminServiceRejectsInvalidTwoFactorCode(t *testing.T) {
	svc := NewKiroGatewayAdminServiceWithHome(t.TempDir())

	_, err := svc.ImportCredentials(context.Background(), KiroCredentialImportInput{
		Type:          "refresh_token",
		RefreshToken:  "refresh-token-secret",
		TwoFactorCode: "12 3456",
	})
	if err == nil {
		t.Fatalf("ImportCredentials() error = nil, want invalid 2FA error")
	}
	if !strings.Contains(err.Error(), "2FA") {
		t.Fatalf("error = %q, want mention 2FA", err.Error())
	}
}

func TestKiroGatewayAdminServiceDeviceLoginImportsCLICredential(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	cliPath := filepath.Join(binDir, "kiro-cli")
	if err := os.WriteFile(cliPath, []byte(`#!/bin/sh
set -eu
if [ "${1:-}" != "login" ]; then
  echo "unexpected command: $*" >&2
  exit 2
fi
mkdir -p "$HOME/.local/share/kiro-cli"
echo "Open https://device.sso.us-east-1.amazonaws.com/ and enter code ABCD-EFGH"
printf 'SQLite format 3\000fake-login-db' > "$HOME/.local/share/kiro-cli/data.sqlite3"
echo "Login successful"
`), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	t.Setenv("KIRO_CLI_PATH", cliPath)

	svc := NewKiroGatewayAdminServiceWithHome(home)
	session, err := svc.StartCLILogin(context.Background(), KiroCLILoginInput{
		Provider: "builder",
		Append:   true,
	})
	if err != nil {
		t.Fatalf("StartCLILogin() error = %v", err)
	}

	session = waitForKiroLoginStatus(t, svc, session.ID, "imported")
	if session.VerificationURL != "https://device.sso.us-east-1.amazonaws.com/" {
		t.Fatalf("VerificationURL = %q", session.VerificationURL)
	}
	if session.UserCode != "ABCD-EFGH" {
		t.Fatalf("UserCode = %q", session.UserCode)
	}

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.CredentialCount != 1 {
		t.Fatalf("CredentialCount = %d, want 1", status.CredentialCount)
	}
	if got := status.Credentials[0].Type; got != "sqlite" {
		t.Fatalf("credential type = %q, want sqlite", got)
	}
	if strings.Contains(session.Output, "fake-login-db") {
		t.Fatalf("session output should not expose credential file content: %s", session.Output)
	}
}

func waitForKiroLoginStatus(t *testing.T, svc *KiroGatewayAdminService, sessionID, want string) KiroCLILoginSession {
	t.Helper()
	for i := 0; i < 250; i++ {
		session, err := svc.GetCLILoginSession(sessionID)
		if err != nil {
			t.Fatalf("GetCLILoginSession() error = %v", err)
		}
		if session.Status == want {
			return session
		}
		if session.Status == "failed" {
			t.Fatalf("login session failed: %+v", session)
		}
		time.Sleep(20 * time.Millisecond)
	}
	session, _ := svc.GetCLILoginSession(sessionID)
	t.Fatalf("session status = %q, want %q; session = %+v", session.Status, want, session)
	return KiroCLILoginSession{}
}
