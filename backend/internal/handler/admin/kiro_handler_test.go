package admin

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func TestKiroHandlerImportsRefreshTokenWithoutExposingSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	home := t.TempDir()
	h := NewKiroHandler(service.NewKiroGatewayAdminServiceWithHome(home))
	router := gin.New()
	router.POST("/api/v1/admin/kiro/credentials", h.ImportCredentials)
	router.GET("/api/v1/admin/kiro/status", h.GetStatus)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("type", "refresh_token")
	_ = writer.WriteField("refresh_token", "refresh-token-secret")
	_ = writer.WriteField("two_factor_code", "123456")
	_ = writer.WriteField("region", "us-east-1")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kiro/credentials", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	raw, err := os.ReadFile(filepath.Join(home, "secrets", "credentials.json"))
	if err != nil {
		t.Fatalf("read credentials.json: %v", err)
	}
	if strings.Contains(string(raw), "123456") {
		t.Fatalf("2FA code must not be persisted: %s", raw)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/kiro/status", nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	encoded, _ := json.Marshal(resp)
	if strings.Contains(string(encoded), "refresh-token-secret") {
		t.Fatalf("status response leaked refresh token: %s", encoded)
	}
}

func TestKiroHandlerRejectsInvalidTwoFactorCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewKiroHandler(service.NewKiroGatewayAdminServiceWithHome(t.TempDir()))
	router := gin.New()
	router.POST("/api/v1/admin/kiro/credentials", h.ImportCredentials)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("type", "refresh_token")
	_ = writer.WriteField("refresh_token", "refresh-token-secret")
	_ = writer.WriteField("two_factor_code", "12 3456")
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kiro/credentials", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body = %s", rec.Code, rec.Body.String())
	}
}

func TestKiroHandlerStartsAndPollsDeviceLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	home := t.TempDir()
	binDir := t.TempDir()
	cliPath := filepath.Join(binDir, "kiro-cli")
	if err := os.WriteFile(cliPath, []byte(`#!/bin/sh
set -eu
mkdir -p "$HOME/.local/share/kiro-cli"
echo "Open https://device.sso.us-east-1.amazonaws.com/ and enter code WXYZ-1234"
printf 'SQLite format 3\000fake-login-db' > "$HOME/.local/share/kiro-cli/data.sqlite3"
echo "Login successful"
`), 0o755); err != nil {
		t.Fatalf("write fake cli: %v", err)
	}
	t.Setenv("KIRO_CLI_PATH", cliPath)

	h := NewKiroHandler(service.NewKiroGatewayAdminServiceWithHome(home))
	router := gin.New()
	router.POST("/api/v1/admin/kiro/login/start", h.StartCLILogin)
	router.GET("/api/v1/admin/kiro/login/:id", h.GetCLILoginSession)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kiro/login/start", strings.NewReader(`{"provider":"builder","append":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var start response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &start); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	payload, ok := start.Data.(map[string]any)
	if !ok {
		t.Fatalf("start data = %#v", start.Data)
	}
	sessionID, _ := payload["id"].(string)
	if sessionID == "" {
		t.Fatalf("session id is empty: %#v", payload)
	}

	var polled map[string]any
	for i := 0; i < 250; i++ {
		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/kiro/login/"+sessionID, nil)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("poll status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var poll response.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &poll); err != nil {
			t.Fatalf("decode poll response: %v", err)
		}
		polled, _ = poll.Data.(map[string]any)
		if polled["status"] == "imported" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if polled["status"] != "imported" {
		t.Fatalf("status = %#v, want imported", polled)
	}
	if polled["verification_url"] != "https://device.sso.us-east-1.amazonaws.com/" {
		t.Fatalf("verification_url = %#v", polled["verification_url"])
	}
	if polled["user_code"] != "WXYZ-1234" {
		t.Fatalf("user_code = %#v", polled["user_code"])
	}
}
