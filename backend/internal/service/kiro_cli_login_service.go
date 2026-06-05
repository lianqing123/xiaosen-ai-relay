package service

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const kiroCLILoginTimeout = 10 * time.Minute

var (
	ansiEscapePattern      = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	kiroDeviceURLPattern   = regexp.MustCompile(`https?://[^\s"')]+`)
	kiroDeviceCodePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:verification|user|device)?\s*code(?:\s+is)?\s*[:：]?\s*([A-Z0-9][A-Z0-9-]{3,})`),
		regexp.MustCompile(`(?i)enter\s+(?:the\s+)?code\s+([A-Z0-9][A-Z0-9-]{3,})`),
	}
)

type KiroCLILoginInput struct {
	Provider            string `json:"provider"`
	IdentityProviderURL string `json:"identity_provider_url,omitempty"`
	Region              string `json:"region,omitempty"`
	Append              bool   `json:"append"`
	Activate            bool   `json:"activate"`
}

type KiroCLILoginSession struct {
	ID              string                 `json:"id"`
	Status          string                 `json:"status"`
	Provider        string                 `json:"provider"`
	VerificationURL string                 `json:"verification_url,omitempty"`
	UserCode        string                 `json:"user_code,omitempty"`
	Message         string                 `json:"message,omitempty"`
	Output          string                 `json:"output,omitempty"`
	Error           string                 `json:"error,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CredentialCount int                    `json:"credential_count,omitempty"`
	Credentials     []KiroCredentialEntry  `json:"credentials,omitempty"`
	Activation      *KiroActivationResult  `json:"activation,omitempty"`
	Meta            map[string]interface{} `json:"meta,omitempty"`
}

func (s *KiroGatewayAdminService) StartCLILogin(ctx context.Context, input KiroCLILoginInput) (KiroCLILoginSession, error) {
	if s == nil {
		return KiroCLILoginSession{}, errors.New("kiro gateway service is nil")
	}
	input.Provider = normalizeKiroLoginProvider(input.Provider)
	if err := validateKiroCLILoginInput(input); err != nil {
		return KiroCLILoginSession{}, err
	}
	if err := s.ensureDirs(); err != nil {
		return KiroCLILoginSession{}, err
	}
	cliPath, err := s.kiroCLIPath()
	if err != nil {
		return KiroCLILoginSession{}, err
	}

	s.loginMu.Lock()
	if existing := s.activeLoginLocked(); existing != nil {
		snapshot := *existing
		s.loginMu.Unlock()
		return snapshot, nil
	}
	session := &KiroCLILoginSession{
		ID:        newKiroLoginID(),
		Status:    "starting",
		Provider:  input.Provider,
		Message:   "Kiro CLI login starting",
		StartedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.logins[session.ID] = session
	s.loginMu.Unlock()

	go s.runCLILogin(cliPath, session.ID, input)

	return s.GetCLILoginSession(session.ID)
}

func (s *KiroGatewayAdminService) GetCLILoginSession(id string) (KiroCLILoginSession, error) {
	if s == nil {
		return KiroCLILoginSession{}, errors.New("kiro gateway service is nil")
	}
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	if id == "" {
		for _, session := range s.logins {
			if session != nil {
				return *session, nil
			}
		}
		return KiroCLILoginSession{}, errors.New("kiro login session not found")
	}
	session := s.logins[id]
	if session == nil {
		return KiroCLILoginSession{}, errors.New("kiro login session not found")
	}
	return *session, nil
}

func (s *KiroGatewayAdminService) activeLoginLocked() *KiroCLILoginSession {
	for _, session := range s.logins {
		if session == nil {
			continue
		}
		switch session.Status {
		case "starting", "waiting", "importing":
			return session
		}
	}
	return nil
}

func (s *KiroGatewayAdminService) runCLILogin(cliPath, sessionID string, input KiroCLILoginInput) {
	ctx, cancel := context.WithTimeout(context.Background(), kiroCLILoginTimeout)
	defer cancel()

	cliHome := s.cliHomePath()
	if err := os.MkdirAll(filepath.Join(cliHome, ".local", "share", "kiro-cli"), 0o700); err != nil {
		s.failCLILogin(sessionID, err)
		return
	}
	args := buildKiroCLILoginArgs(input)
	cmd := exec.CommandContext(ctx, cliPath, args...)
	cmd.Dir = s.home
	cmd.Env = append(os.Environ(),
		"HOME="+cliHome,
		"XDG_DATA_HOME="+filepath.Join(cliHome, ".local", "share"),
		"XDG_CONFIG_HOME="+filepath.Join(cliHome, ".config"),
		"KIRO_CHAT_LOG_FILE="+filepath.Join(s.secretsPath(), "kiro-cli-login.log"),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.failCLILogin(sessionID, err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		s.failCLILogin(sessionID, err)
		return
	}
	if err := cmd.Start(); err != nil {
		s.failCLILogin(sessionID, err)
		return
	}

	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		session.Status = "waiting"
		session.Message = "Open the official Kiro login URL and enter the device code"
	})

	var wg sync.WaitGroup
	wg.Add(2)
	go s.scanCLILoginOutput(sessionID, stdout, &wg)
	go s.scanCLILoginOutput(sessionID, stderr, &wg)
	waitErr := cmd.Wait()
	wg.Wait()
	if ctx.Err() != nil {
		s.failCLILogin(sessionID, ctx.Err())
		return
	}
	if waitErr != nil {
		s.failCLILogin(sessionID, waitErr)
		return
	}

	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		session.Status = "importing"
		session.Message = "Kiro login complete, importing CLI credential"
	})

	credentialType, credentialPath, err := s.findCLILoginCredential()
	if err != nil {
		s.failCLILogin(sessionID, err)
		return
	}
	importCtx, importCancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer importCancel()
	result, err := s.ImportCredentials(importCtx, KiroCredentialImportInput{
		Type:     credentialType,
		FileName: filepath.Base(credentialPath),
		FilePath: credentialPath,
		Region:   input.Region,
		Append:   input.Append,
		Activate: input.Activate,
	})
	if err != nil {
		s.failCLILogin(sessionID, err)
		return
	}
	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		if result.Activation != nil {
			session.Status = "activated"
			session.Message = "Kiro credential imported and gateway activated"
			session.Activation = result.Activation
			if strings.TrimSpace(result.Activation.Output) != "" {
				session.Output = result.Activation.Output
			}
		} else {
			session.Status = "imported"
			session.Message = "Kiro credential imported"
		}
		session.CredentialCount = result.CredentialCount
		session.Credentials = result.Credentials
	})
}

func (s *KiroGatewayAdminService) scanCLILoginOutput(sessionID string, reader io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := cleanKiroCLILoginLine(scanner.Text())
		s.appendCLILoginOutput(sessionID, line+"\n")
		s.updateCLILoginHints(sessionID, line)
	}
}

func (s *KiroGatewayAdminService) appendCLILoginOutput(sessionID, output string) {
	output = redactKiroOutput(output)
	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		session.Output = truncateActivationOutput(session.Output + output)
	})
}

func (s *KiroGatewayAdminService) updateCLILoginHints(sessionID, line string) {
	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		if session.VerificationURL == "" {
			if match := kiroDeviceURLPattern.FindString(line); match != "" {
				session.VerificationURL = strings.TrimRight(match, ".,")
			}
		}
		if session.UserCode == "" {
			for _, pattern := range kiroDeviceCodePatterns {
				match := pattern.FindStringSubmatch(line)
				if len(match) > 1 {
					session.UserCode = strings.TrimSpace(match[1])
					break
				}
			}
		}
	})
}

func (s *KiroGatewayAdminService) updateCLILogin(sessionID string, mutate func(*KiroCLILoginSession)) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	session := s.logins[sessionID]
	if session == nil {
		return
	}
	mutate(session)
	session.UpdatedAt = time.Now().UTC()
}

func (s *KiroGatewayAdminService) failCLILogin(sessionID string, err error) {
	if err == nil {
		return
	}
	s.updateCLILogin(sessionID, func(session *KiroCLILoginSession) {
		session.Status = "failed"
		session.Message = "Kiro CLI login failed"
		session.Error = redactKiroOutput(err.Error())
	})
}

func (s *KiroGatewayAdminService) findCLILoginCredential() (string, string, error) {
	cliHome := s.cliHomePath()
	sqliteCandidates := []string{
		filepath.Join(cliHome, ".local", "share", "kiro-cli", "data.sqlite3"),
		filepath.Join(cliHome, ".local", "share", "kiro-cli", "data.sqlite"),
	}
	for _, candidate := range sqliteCandidates {
		if looksLikeFile(candidate) {
			return "sqlite", candidate, nil
		}
	}

	cacheDir := filepath.Join(cliHome, ".aws", "sso", "cache")
	matches, _ := filepath.Glob(filepath.Join(cacheDir, "*.json"))
	for _, candidate := range matches {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if _, ok := payload["refreshToken"]; ok {
			return "json", candidate, nil
		}
		if _, ok := payload["refresh_token"]; ok {
			return "json", candidate, nil
		}
		if _, ok := payload["clientId"]; ok {
			return "json", candidate, nil
		}
	}
	return "", "", errors.New("kiro cli login completed but no credential file was found")
}

func (s *KiroGatewayAdminService) kiroCLIPath() (string, error) {
	candidates := []string{
		strings.TrimSpace(os.Getenv("KIRO_CLI_PATH")),
		filepath.Join(s.home, "bin", "kiro-cli"),
		filepath.Join(s.home, ".local", "bin", "kiro-cli"),
		"/usr/local/bin/kiro-cli",
		"/usr/bin/kiro-cli",
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath("kiro-cli"); err == nil {
		return path, nil
	}
	return "", errors.New("kiro-cli not found; install Kiro CLI on the server first")
}

func (s *KiroGatewayAdminService) cliHomePath() string {
	return filepath.Join(s.secretsPath(), "kiro-cli-home")
}

func buildKiroCLILoginArgs(input KiroCLILoginInput) []string {
	args := []string{"login", "--use-device-flow"}
	switch normalizeKiroLoginProvider(input.Provider) {
	case "builder":
		args = append(args, "--license", "free")
	case "google":
		args = append(args, "--social", "google")
	case "github":
		args = append(args, "--social", "github")
	case "identity_center":
		args = append(args, "--license", "pro", "--identity-provider", strings.TrimSpace(input.IdentityProviderURL))
		if region := strings.TrimSpace(input.Region); region != "" {
			args = append(args, "--region", region)
		}
	}
	return args
}

func validateKiroCLILoginInput(input KiroCLILoginInput) error {
	switch normalizeKiroLoginProvider(input.Provider) {
	case "builder", "google", "github":
		return nil
	case "identity_center":
		if strings.TrimSpace(input.IdentityProviderURL) == "" {
			return errors.New("identity provider URL is required")
		}
		return nil
	default:
		return errors.New("kiro login provider must be builder, google, github, or identity_center")
	}
}

func normalizeKiroLoginProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "", "builder_id", "aws_builder_id", "aws-builder-id":
		return "builder"
	case "identity", "identity-center", "iam_identity_center", "iam-identity-center", "sso":
		return "identity_center"
	default:
		return provider
	}
}

func cleanKiroCLILoginLine(line string) string {
	return strings.TrimSpace(ansiEscapePattern.ReplaceAllString(line, ""))
}

func looksLikeFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func newKiroLoginID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "kiro_" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("kiro_%d", time.Now().UnixNano())
}
