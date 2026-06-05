package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultKiroGatewayHome = "/opt/kiro-gateway"
	kiroContainerPrefix    = "/app/secrets/imports/"
	defaultKiroUsageRegion = "us-east-1"
)

var twoFactorCodePattern = regexp.MustCompile(`^[0-9]{6,8}$`)

type KiroGatewayAdminService struct {
	home       string
	httpClient *http.Client
	loginMu    sync.Mutex
	logins     map[string]*KiroCLILoginSession
}

type KiroCredentialImportInput struct {
	Type          string
	FileName      string
	FilePath      string
	RefreshToken  string
	ProfileARN    string
	Region        string
	APIRegion     string
	Append        bool
	Activate      bool
	TwoFactorCode string
}

type KiroCredentialEntry struct {
	Type                string `json:"type"`
	Path                string `json:"path,omitempty"`
	Enabled             *bool  `json:"enabled,omitempty"`
	ProfileARN          string `json:"profile_arn,omitempty"`
	Region              string `json:"region,omitempty"`
	APIRegion           string `json:"api_region,omitempty"`
	Comment             string `json:"comment,omitempty"`
	HasRefreshToken     bool   `json:"has_refresh_token,omitempty"`
	RefreshTokenSHA256  string `json:"refresh_token_sha256,omitempty"`
	RefreshTokenPresent bool   `json:"refresh_token_present,omitempty"`
}

type KiroGatewayStatus struct {
	KiroHome               string                `json:"kiro_home"`
	CredentialsFileExists  bool                  `json:"credentials_file_exists"`
	CredentialCount        int                   `json:"credential_count"`
	Credentials            []KiroCredentialEntry `json:"credentials"`
	OfficialQuota          *KiroOfficialQuota    `json:"official_quota,omitempty"`
	ActivationScriptExists bool                  `json:"activation_script_exists"`
	CLIAvailable           bool                  `json:"cli_available"`
	CLIPath                string                `json:"cli_path,omitempty"`
	GatewayHealthy         bool                  `json:"gateway_healthy"`
	GatewayHealthMessage   string                `json:"gateway_health_message,omitempty"`
}

type KiroOfficialQuota struct {
	Available         bool                       `json:"available"`
	Source            string                     `json:"source"`
	FetchedAt         time.Time                  `json:"fetched_at"`
	Error             string                     `json:"error,omitempty"`
	AccountCount      int                        `json:"account_count"`
	UniqueUserCount   int                        `json:"unique_user_count"`
	SubscriptionTitle string                     `json:"subscription_title,omitempty"`
	SubscriptionType  string                     `json:"subscription_type,omitempty"`
	OverageStatus     string                     `json:"overage_status,omitempty"`
	NextResetAt       *time.Time                 `json:"next_reset_at,omitempty"`
	DisplayName       string                     `json:"display_name,omitempty"`
	Currency          string                     `json:"currency,omitempty"`
	CurrentUsage      float64                    `json:"current_usage"`
	UsageLimit        float64                    `json:"usage_limit"`
	Remaining         float64                    `json:"remaining"`
	OverageCharges    float64                    `json:"overage_charges"`
	OverageRate       float64                    `json:"overage_rate"`
	Accounts          []KiroOfficialQuotaAccount `json:"accounts,omitempty"`
}

type KiroOfficialQuotaAccount struct {
	CredentialSHA256  string     `json:"credential_sha256,omitempty"`
	UserID            string     `json:"user_id,omitempty"`
	Available         bool       `json:"available"`
	Error             string     `json:"error,omitempty"`
	SubscriptionTitle string     `json:"subscription_title,omitempty"`
	SubscriptionType  string     `json:"subscription_type,omitempty"`
	OverageStatus     string     `json:"overage_status,omitempty"`
	NextResetAt       *time.Time `json:"next_reset_at,omitempty"`
	DisplayName       string     `json:"display_name,omitempty"`
	Currency          string     `json:"currency,omitempty"`
	CurrentUsage      float64    `json:"current_usage"`
	UsageLimit        float64    `json:"usage_limit"`
	Remaining         float64    `json:"remaining"`
	OverageCharges    float64    `json:"overage_charges"`
	OverageRate       float64    `json:"overage_rate"`
	Deduplicated      bool       `json:"deduplicated,omitempty"`
}

type KiroCredentialImportResult struct {
	Message           string                `json:"message"`
	CredentialCount   int                   `json:"credential_count"`
	Credentials       []KiroCredentialEntry `json:"credentials"`
	TwoFactorAccepted bool                  `json:"two_factor_accepted"`
	Activation        *KiroActivationResult `json:"activation,omitempty"`
}

type KiroActivationResult struct {
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

type rawKiroCredential map[string]any

func NewKiroGatewayAdminService() *KiroGatewayAdminService {
	home := strings.TrimSpace(os.Getenv("KIRO_GATEWAY_HOME"))
	if home == "" {
		home = defaultKiroGatewayHome
	}
	return NewKiroGatewayAdminServiceWithHome(home)
}

func NewKiroGatewayAdminServiceWithHome(home string) *KiroGatewayAdminService {
	return &KiroGatewayAdminService{
		home: strings.TrimRight(home, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logins: make(map[string]*KiroCLILoginSession),
	}
}

func (s *KiroGatewayAdminService) Status(ctx context.Context) (*KiroGatewayStatus, error) {
	if s == nil {
		return nil, errors.New("kiro gateway service is nil")
	}
	credentials, err := s.readCredentials()
	if err != nil {
		return nil, err
	}
	_, credentialsErr := os.Stat(s.credentialsPath())
	_, activationErr := os.Stat(s.activationScriptPath())
	cliPath, cliErr := s.kiroCLIPath()

	healthy, message := s.checkGatewayHealth(ctx)
	return &KiroGatewayStatus{
		KiroHome:               s.home,
		CredentialsFileExists:  credentialsErr == nil,
		CredentialCount:        len(credentials),
		Credentials:            sanitizeKiroCredentials(credentials),
		OfficialQuota:          s.fetchOfficialQuota(ctx, credentials),
		ActivationScriptExists: activationErr == nil,
		CLIAvailable:           cliErr == nil,
		CLIPath:                cliPath,
		GatewayHealthy:         healthy,
		GatewayHealthMessage:   message,
	}, nil
}

func (s *KiroGatewayAdminService) ImportCredentials(ctx context.Context, input KiroCredentialImportInput) (*KiroCredentialImportResult, error) {
	if s == nil {
		return nil, errors.New("kiro gateway service is nil")
	}
	if err := validateTwoFactorCode(input.TwoFactorCode); err != nil {
		return nil, err
	}
	if err := s.ensureDirs(); err != nil {
		return nil, err
	}

	entry, err := s.buildCredentialEntry(input)
	if err != nil {
		return nil, err
	}

	credentials := make([]rawKiroCredential, 0, 1)
	if input.Append {
		existing, err := s.readCredentials()
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, existing...)
	}
	credentials = append(credentials, entry)
	if err := s.writeCredentials(credentials); err != nil {
		return nil, err
	}

	result := &KiroCredentialImportResult{
		Message:           "credentials imported",
		CredentialCount:   len(credentials),
		Credentials:       sanitizeKiroCredentials(credentials),
		TwoFactorAccepted: strings.TrimSpace(input.TwoFactorCode) != "",
	}
	if input.Activate {
		activation, err := s.Activate(ctx)
		if err != nil {
			return nil, err
		}
		result.Activation = activation
	}
	return result, nil
}

func (s *KiroGatewayAdminService) Activate(ctx context.Context) (*KiroActivationResult, error) {
	if s == nil {
		return nil, errors.New("kiro gateway service is nil")
	}
	credentials, err := s.readCredentials()
	if err != nil {
		return nil, err
	}
	if len(credentials) == 0 {
		return nil, errors.New("credentials.json is empty")
	}
	script := s.activationScriptPath()
	if info, err := os.Stat(script); err != nil {
		return nil, fmt.Errorf("activation script not found: %w", err)
	} else if info.IsDir() {
		return nil, errors.New("activation script path is a directory")
	}

	execCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(execCtx, script)
	cmd.Dir = s.home
	output, err := cmd.CombinedOutput()
	cleanOutput := truncateActivationOutput(redactKiroOutput(string(output)))
	if err != nil {
		if execCtx.Err() != nil {
			return nil, fmt.Errorf("activation timed out: %w", execCtx.Err())
		}
		return nil, fmt.Errorf("activation failed: %w: %s", err, cleanOutput)
	}
	return &KiroActivationResult{
		Message: "kiro gateway activated",
		Output:  cleanOutput,
	}, nil
}

func (s *KiroGatewayAdminService) buildCredentialEntry(input KiroCredentialImportInput) (rawKiroCredential, error) {
	credentialType := strings.ToLower(strings.TrimSpace(input.Type))
	switch credentialType {
	case "refresh_token":
		token := strings.TrimSpace(input.RefreshToken)
		if token == "" {
			return nil, errors.New("refresh token is required")
		}
		entry := rawKiroCredential{
			"type":          "refresh_token",
			"refresh_token": token,
			"enabled":       true,
			"comment":       "Imported from admin Kiro page",
		}
		applyOptionalCredentialFields(entry, input)
		return entry, nil
	case "json":
		containerPath, err := s.copyJSONCredential(input.FilePath, input.FileName)
		if err != nil {
			return nil, err
		}
		entry := rawKiroCredential{
			"type":    "json",
			"path":    containerPath,
			"enabled": true,
			"comment": "Imported from admin Kiro page",
		}
		applyOptionalCredentialFields(entry, input)
		return entry, nil
	case "sqlite":
		containerPath, err := s.copySQLiteCredential(input.FilePath, input.FileName)
		if err != nil {
			return nil, err
		}
		entry := rawKiroCredential{
			"type":    "sqlite",
			"path":    containerPath,
			"enabled": true,
			"comment": "Imported from admin Kiro page",
		}
		applyOptionalCredentialFields(entry, input)
		return entry, nil
	default:
		return nil, errors.New("credential type must be json, sqlite, or refresh_token")
	}
}

func applyOptionalCredentialFields(entry rawKiroCredential, input KiroCredentialImportInput) {
	if v := strings.TrimSpace(input.ProfileARN); v != "" {
		entry["profile_arn"] = v
	}
	if v := strings.TrimSpace(input.Region); v != "" {
		entry["region"] = v
	} else if entry["type"] == "refresh_token" {
		entry["region"] = "us-east-1"
	}
	if v := strings.TrimSpace(input.APIRegion); v != "" {
		entry["api_region"] = v
	}
}

func (s *KiroGatewayAdminService) copyJSONCredential(sourcePath, originalName string) (string, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return "", errors.New("json credential file is required")
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("invalid json credential file: %w", err)
	}
	if err := validateKiroJSONCredentialPayload(payload); err != nil {
		return "", err
	}
	return s.writeImportFile(originalName, ".json", raw)
}

func validateKiroJSONCredentialPayload(payload map[string]any) error {
	if len(payload) == 0 {
		return errors.New("json credential file is empty")
	}
	for _, snakeKey := range []string{"access_token", "refresh_token", "expires_at", "client_id", "client_secret", "profile_arn"} {
		if _, ok := payload[snakeKey]; ok {
			return fmt.Errorf("json credential must use Kiro camelCase fields, found %q", snakeKey)
		}
	}
	if _, err := requiredJSONString(payload, "refreshToken"); err != nil {
		return err
	}
	if expiresAt := strings.TrimSpace(stringFromAny(payload["expiresAt"])); expiresAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, expiresAt); err != nil {
			return fmt.Errorf("json credential field %q must be ISO 8601/RFC3339 time", "expiresAt")
		}
	}

	hasClientID := strings.TrimSpace(stringFromAny(payload["clientId"])) != ""
	hasClientSecret := strings.TrimSpace(stringFromAny(payload["clientSecret"])) != ""
	if hasClientID || hasClientSecret {
		if _, err := requiredJSONString(payload, "clientId"); err != nil {
			return err
		}
		if _, err := requiredJSONString(payload, "clientSecret"); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func requiredJSONString(payload map[string]any, key string) (string, error) {
	value, ok := payload[key]
	if !ok {
		return "", fmt.Errorf("json credential missing required field %q", key)
	}
	text := strings.TrimSpace(stringFromAny(value))
	if text == "" {
		return "", fmt.Errorf("json credential field %q must be a non-empty string", key)
	}
	return text, nil
}

func stringFromAny(value any) string {
	text, _ := value.(string)
	return text
}

func (s *KiroGatewayAdminService) copySQLiteCredential(sourcePath, originalName string) (string, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return "", errors.New("sqlite credential file is required")
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	if !bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		return "", errors.New("sqlite file does not look like a SQLite database")
	}
	return s.writeImportFile(originalName, ".sqlite3", raw)
}

func (s *KiroGatewayAdminService) writeImportFile(originalName, fallbackExt string, raw []byte) (string, error) {
	if err := s.ensureDirs(); err != nil {
		return "", err
	}
	name := sanitizeImportFileName(originalName)
	if name == "" {
		name = "credential" + fallbackExt
	}
	if filepath.Ext(name) == "" && fallbackExt != "" {
		name += fallbackExt
	}
	name = fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102150405"), name)
	hostPath := filepath.Join(s.importsPath(), name)
	if err := os.WriteFile(hostPath, raw, 0o640); err != nil {
		return "", err
	}
	if err := os.Chmod(hostPath, 0o640); err != nil {
		return "", err
	}
	return kiroContainerPrefix + name, nil
}

func (s *KiroGatewayAdminService) readCredentials() ([]rawKiroCredential, error) {
	raw, err := os.ReadFile(s.credentialsPath())
	if errors.Is(err, os.ErrNotExist) {
		return []rawKiroCredential{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return []rawKiroCredential{}, nil
	}
	var credentials []rawKiroCredential
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials.json: %w", err)
	}
	if credentials == nil {
		return []rawKiroCredential{}, nil
	}
	return credentials, nil
}

func (s *KiroGatewayAdminService) writeCredentials(credentials []rawKiroCredential) error {
	if err := s.ensureDirs(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(s.credentialsPath(), raw, 0o640); err != nil {
		return err
	}
	return os.Chmod(s.credentialsPath(), 0o640)
}

func sanitizeKiroCredentials(credentials []rawKiroCredential) []KiroCredentialEntry {
	out := make([]KiroCredentialEntry, 0, len(credentials))
	for _, raw := range credentials {
		entry := KiroCredentialEntry{
			Type:       stringFromMap(raw, "type"),
			Path:       stringFromMap(raw, "path"),
			ProfileARN: stringFromMap(raw, "profile_arn"),
			Region:     stringFromMap(raw, "region"),
			APIRegion:  stringFromMap(raw, "api_region"),
			Comment:    stringFromMap(raw, "comment"),
		}
		if enabled, ok := raw["enabled"].(bool); ok {
			entry.Enabled = &enabled
		}
		if token := stringFromMap(raw, "refresh_token"); token != "" {
			sum := sha256.Sum256([]byte(token))
			entry.RefreshTokenPresent = true
			entry.HasRefreshToken = true
			entry.RefreshTokenSHA256 = hex.EncodeToString(sum[:])[:12]
		}
		out = append(out, entry)
	}
	return out
}

func (s *KiroGatewayAdminService) fetchOfficialQuota(ctx context.Context, credentials []rawKiroCredential) *KiroOfficialQuota {
	quota := &KiroOfficialQuota{
		Available: false,
		Source:    "AmazonCodeWhispererService.GetUsageLimits",
		FetchedAt: time.Now().UTC(),
		Accounts:  []KiroOfficialQuotaAccount{},
	}
	if s == nil {
		quota.Error = "kiro gateway service is nil"
		return quota
	}

	enabled := make([]rawKiroCredential, 0, len(credentials))
	for _, credential := range credentials {
		if enabledValue, ok := credential["enabled"].(bool); ok && !enabledValue {
			continue
		}
		if strings.TrimSpace(stringFromMap(credential, "refresh_token")) == "" && strings.TrimSpace(stringFromMap(credential, "path")) == "" {
			continue
		}
		enabled = append(enabled, credential)
	}
	quota.AccountCount = len(enabled)
	if len(enabled) == 0 {
		quota.Error = "no enabled Kiro credentials"
		return quota
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	type indexedResult struct {
		index int
		item  KiroOfficialQuotaAccount
	}
	results := make(chan indexedResult, len(enabled))
	sem := make(chan struct{}, 4)
	for index, credential := range enabled {
		index, credential := index, credential
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			results <- indexedResult{index: index, item: s.fetchOfficialQuotaForCredential(ctx, credential)}
		}()
	}

	items := make([]KiroOfficialQuotaAccount, len(enabled))
	for range enabled {
		result := <-results
		items[result.index] = result.item
	}

	seenUsers := map[string]bool{}
	var firstErr string
	for _, item := range items {
		if !item.Available {
			if firstErr == "" {
				firstErr = item.Error
			}
			quota.Accounts = append(quota.Accounts, item)
			continue
		}

		userKey := strings.TrimSpace(item.UserID)
		if userKey == "" {
			userKey = strings.TrimSpace(item.CredentialSHA256)
		}
		if userKey != "" && seenUsers[userKey] {
			item.Deduplicated = true
			quota.Accounts = append(quota.Accounts, item)
			continue
		}
		if userKey != "" {
			seenUsers[userKey] = true
		}

		quota.Available = true
		quota.UniqueUserCount++
		quota.CurrentUsage += item.CurrentUsage
		quota.UsageLimit += item.UsageLimit
		quota.OverageCharges += item.OverageCharges
		if item.OverageRate > quota.OverageRate {
			quota.OverageRate = item.OverageRate
		}
		if quota.SubscriptionTitle == "" {
			quota.SubscriptionTitle = item.SubscriptionTitle
			quota.SubscriptionType = item.SubscriptionType
			quota.OverageStatus = item.OverageStatus
			quota.DisplayName = item.DisplayName
			quota.Currency = item.Currency
			quota.NextResetAt = item.NextResetAt
		} else {
			if quota.SubscriptionTitle != item.SubscriptionTitle && item.SubscriptionTitle != "" {
				quota.SubscriptionTitle = "Mixed"
			}
			if quota.SubscriptionType != item.SubscriptionType && item.SubscriptionType != "" {
				quota.SubscriptionType = "Mixed"
			}
			if quota.OverageStatus != item.OverageStatus && item.OverageStatus != "" {
				quota.OverageStatus = "Mixed"
			}
			if quota.NextResetAt == nil || (item.NextResetAt != nil && item.NextResetAt.Before(*quota.NextResetAt)) {
				quota.NextResetAt = item.NextResetAt
			}
		}
		quota.Accounts = append(quota.Accounts, item)
	}
	quota.CurrentUsage = roundKiroQuota(quota.CurrentUsage)
	quota.UsageLimit = roundKiroQuota(quota.UsageLimit)
	quota.Remaining = roundKiroQuota(maxFloat64(quota.UsageLimit-quota.CurrentUsage, 0))
	quota.OverageCharges = roundKiroQuota(quota.OverageCharges)
	if !quota.Available && firstErr != "" {
		quota.Error = firstErr
	}
	return quota
}

func (s *KiroGatewayAdminService) fetchOfficialQuotaForCredential(ctx context.Context, credential rawKiroCredential) KiroOfficialQuotaAccount {
	out := KiroOfficialQuotaAccount{Available: false}
	refreshToken := strings.TrimSpace(stringFromMap(credential, "refresh_token"))
	if refreshToken != "" {
		sum := sha256.Sum256([]byte(refreshToken))
		out.CredentialSHA256 = hex.EncodeToString(sum[:])[:16]
	}

	credentialType := strings.ToLower(strings.TrimSpace(stringFromMap(credential, "type")))
	if refreshToken == "" && credentialType == "json" {
		payload, err := s.readKiroJSONCredential(credential)
		if err != nil {
			out.Error = err.Error()
			return out
		}
		refreshToken = strings.TrimSpace(stringFromMap(payload, "refreshToken"))
		if refreshToken != "" {
			sum := sha256.Sum256([]byte(refreshToken))
			out.CredentialSHA256 = hex.EncodeToString(sum[:])[:16]
		}
		for _, key := range []string{"profileArn", "region", "apiRegion"} {
			if _, exists := credential[strings.TrimSuffix(strings.ToLower(key), "arn")]; !exists {
				if value := strings.TrimSpace(stringFromMap(payload, key)); value != "" {
					credential[key] = value
				}
			}
		}
	}
	if refreshToken == "" {
		out.Error = "refresh token is required for official Kiro quota"
		return out
	}

	access, profileARN, err := s.refreshKiroAccessToken(ctx, credential, refreshToken)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	usage, err := s.getKiroUsageLimits(ctx, credential, access, profileARN)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	return mergeKiroUsageIntoAccount(out, usage)
}

func (s *KiroGatewayAdminService) readKiroJSONCredential(credential rawKiroCredential) (rawKiroCredential, error) {
	path := strings.TrimSpace(stringFromMap(credential, "path"))
	if path == "" {
		return nil, errors.New("json credential path is empty")
	}
	if strings.HasPrefix(path, kiroContainerPrefix) {
		path = filepath.Join(s.importsPath(), strings.TrimPrefix(path, kiroContainerPrefix))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload rawKiroCredential
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

type kiroDesktopRefreshResponse struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	ExpiresIn    float64 `json:"expiresIn"`
	ProfileARN   string  `json:"profileArn"`
}

func (s *KiroGatewayAdminService) refreshKiroAccessToken(ctx context.Context, credential rawKiroCredential, refreshToken string) (string, string, error) {
	region := kiroCredentialRegion(credential)
	refreshURL := kiroDesktopRefreshURL(region)
	payload, _ := json.Marshal(map[string]string{"refreshToken": refreshToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL, bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "KiroIDE-0.7.45-sub2api")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("refresh Kiro token failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("refresh Kiro token status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result kiroDesktopRefreshResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("decode Kiro token response: %w", err)
	}
	if strings.TrimSpace(result.AccessToken) == "" {
		return "", "", errors.New("Kiro token response missing accessToken")
	}
	profileARN := strings.TrimSpace(result.ProfileARN)
	if profileARN == "" {
		profileARN = strings.TrimSpace(stringFromMap(credential, "profile_arn"))
	}
	if profileARN == "" {
		profileARN = strings.TrimSpace(stringFromMap(credential, "profileArn"))
	}
	return result.AccessToken, profileARN, nil
}

func (s *KiroGatewayAdminService) getKiroUsageLimits(ctx context.Context, credential rawKiroCredential, accessToken, profileARN string) (map[string]any, error) {
	body := rawKiroCredential{"resourceType": "AGENTIC_REQUEST"}
	if strings.TrimSpace(profileARN) != "" {
		body["profileArn"] = strings.TrimSpace(profileARN)
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kiroUsageEndpoint(kiroCredentialRegion(credential)), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	fingerprint := kiroRandomHex(8)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("x-amz-target", "AmazonCodeWhispererService.GetUsageLimits")
	req.Header.Set("User-Agent", "aws-sdk-js/1.0.27 ua/2.1 os/win32#10.0.19044 lang/js md/nodejs#22.21.1 api/codewhisperer#1.0.27 m/E KiroIDE-0.7.45-"+fingerprint)
	req.Header.Set("x-amz-user-agent", "aws-sdk-js/1.0.27 KiroIDE-0.7.45-"+fingerprint)
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	req.Header.Set("amz-sdk-invocation-id", kiroRandomHex(16))
	req.Header.Set("amz-sdk-request", "attempt=1; max=3")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Kiro usage limits failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch Kiro usage limits status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode Kiro usage limits: %w", err)
	}
	return result, nil
}

func mergeKiroUsageIntoAccount(out KiroOfficialQuotaAccount, usage map[string]any) KiroOfficialQuotaAccount {
	out.Available = true
	if userInfo, ok := usage["userInfo"].(map[string]any); ok {
		out.UserID = strings.TrimSpace(stringFromAny(userInfo["userId"]))
	}
	if info, ok := usage["subscriptionInfo"].(map[string]any); ok {
		out.SubscriptionTitle = strings.TrimSpace(stringFromAny(info["subscriptionTitle"]))
		out.SubscriptionType = strings.TrimSpace(stringFromAny(info["type"]))
	}
	if overage, ok := usage["overageConfiguration"].(map[string]any); ok {
		out.OverageStatus = strings.TrimSpace(stringFromAny(overage["overageStatus"]))
	}
	out.NextResetAt = epochSecondsToTime(float64FromAny(usage["nextDateReset"]))
	if list, ok := usage["usageBreakdownList"].([]any); ok {
		for _, entry := range list {
			breakdown, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(stringFromAny(breakdown["resourceType"])), "CREDIT") || out.DisplayName == "" {
				out.DisplayName = firstNonEmptyKiroString(stringFromAny(breakdown["displayNamePlural"]), stringFromAny(breakdown["displayName"]))
				out.Currency = strings.TrimSpace(stringFromAny(breakdown["currency"]))
				out.CurrentUsage = firstPositiveFloat(float64FromAny(breakdown["currentUsageWithPrecision"]), float64FromAny(breakdown["currentUsage"]))
				out.UsageLimit = firstPositiveFloat(float64FromAny(breakdown["usageLimitWithPrecision"]), float64FromAny(breakdown["usageLimit"]))
				out.Remaining = roundKiroQuota(maxFloat64(out.UsageLimit-out.CurrentUsage, 0))
				out.OverageCharges = firstPositiveFloat(float64FromAny(breakdown["overageChargesWithPrecision"]), float64FromAny(breakdown["overageCharges"]))
				out.OverageRate = float64FromAny(breakdown["overageRate"])
				if resetAt := epochSecondsToTime(float64FromAny(breakdown["nextDateReset"])); resetAt != nil {
					out.NextResetAt = resetAt
				}
				if strings.EqualFold(strings.TrimSpace(stringFromAny(breakdown["resourceType"])), "CREDIT") {
					break
				}
			}
		}
	}
	out.CurrentUsage = roundKiroQuota(out.CurrentUsage)
	out.UsageLimit = roundKiroQuota(out.UsageLimit)
	out.Remaining = roundKiroQuota(out.Remaining)
	out.OverageCharges = roundKiroQuota(out.OverageCharges)
	return out
}

func kiroCredentialRegion(credential rawKiroCredential) string {
	for _, key := range []string{"api_region", "apiRegion", "region"} {
		if region := strings.TrimSpace(stringFromMap(credential, key)); region != "" {
			return region
		}
	}
	return defaultKiroUsageRegion
}

func kiroDesktopRefreshURL(region string) string {
	if override := strings.TrimSpace(os.Getenv("KIRO_DESKTOP_AUTH_BASE_URL")); override != "" {
		return strings.TrimRight(override, "/") + "/refreshToken"
	}
	if region == "" {
		region = defaultKiroUsageRegion
	}
	return fmt.Sprintf("https://prod.%s.auth.desktop.kiro.dev/refreshToken", region)
}

func kiroUsageEndpoint(region string) string {
	if override := strings.TrimSpace(os.Getenv("KIRO_CODEWHISPERER_USAGE_URL")); override != "" {
		return override
	}
	if region == "" {
		region = defaultKiroUsageRegion
	}
	return fmt.Sprintf("https://codewhisperer.%s.amazonaws.com", region)
}

func epochSecondsToTime(value float64) *time.Time {
	if value <= 0 {
		return nil
	}
	seconds := int64(value)
	nanos := int64((value - float64(seconds)) * 1_000_000_000)
	t := time.Unix(seconds, nanos).UTC()
	return &t
}

func float64FromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	default:
		return 0
	}
}

func firstPositiveFloat(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptyKiroString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func maxFloat64(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func roundKiroQuota(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func kiroRandomHex(bytesLen int) string {
	if bytesLen <= 0 {
		return ""
	}
	raw := make([]byte, bytesLen)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw)
}

func stringFromMap(raw rawKiroCredential, key string) string {
	v, _ := raw[key].(string)
	return v
}

func validateTwoFactorCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	if !twoFactorCodePattern.MatchString(code) {
		return errors.New("2FA code must be 6 to 8 digits")
	}
	return nil
}

func sanitizeImportFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == "/" {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), ".-")
}

func (s *KiroGatewayAdminService) checkGatewayHealth(ctx context.Context) (bool, string) {
	if s.httpClient == nil {
		return false, "http client not configured"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8000/health", nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Sprintf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = "healthy"
	}
	return true, msg
}

func (s *KiroGatewayAdminService) ensureDirs() error {
	if err := os.MkdirAll(s.importsPath(), 0o770); err != nil {
		return err
	}
	if err := os.Chmod(s.secretsPath(), 0o2770); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Chmod(s.importsPath(), 0o2770)
}

func (s *KiroGatewayAdminService) secretsPath() string {
	return filepath.Join(s.home, "secrets")
}

func (s *KiroGatewayAdminService) importsPath() string {
	return filepath.Join(s.secretsPath(), "imports")
}

func (s *KiroGatewayAdminService) credentialsPath() string {
	return filepath.Join(s.secretsPath(), "credentials.json")
}

func (s *KiroGatewayAdminService) activationScriptPath() string {
	return filepath.Join(s.home, "activate_after_credentials.sh")
}

func redactKiroOutput(output string) string {
	return regexp.MustCompile(`(?i)(refresh[_-]?token|access[_-]?token|api[_-]?key|client[_-]?secret)(["'=:\s]+)([^,"\s]+)`).ReplaceAllString(output, "$1$2***")
}

func truncateActivationOutput(output string) string {
	output = strings.TrimSpace(output)
	const max = 12000
	if len(output) <= max {
		return output
	}
	return output[:max] + "\n... truncated ..."
}
