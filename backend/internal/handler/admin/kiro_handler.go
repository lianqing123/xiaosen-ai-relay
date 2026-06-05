package admin

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const maxKiroCredentialUploadSize int64 = 20 << 20

type KiroHandler struct {
	kiroService *service.KiroGatewayAdminService
}

type startKiroCLILoginRequest struct {
	Provider            string `json:"provider"`
	IdentityProviderURL string `json:"identity_provider_url"`
	Region              string `json:"region"`
	Append              bool   `json:"append"`
	Activate            bool   `json:"activate"`
}

func NewKiroHandler(kiroService *service.KiroGatewayAdminService) *KiroHandler {
	return &KiroHandler{kiroService: kiroService}
}

// GetStatus returns the local Kiro Gateway credential and sidecar status.
func (h *KiroHandler) GetStatus(c *gin.Context) {
	if h.kiroService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Kiro Gateway service not available")
		return
	}
	status, err := h.kiroService.Status(c.Request.Context())
	if err != nil {
		response.Error(c, mapKiroErrorStatus(err), err.Error())
		return
	}
	response.Success(c, status)
}

// ImportCredentials accepts a refresh token or credential file and writes
// Kiro Gateway credentials.json without persisting transient 2FA codes.
func (h *KiroHandler) ImportCredentials(c *gin.Context) {
	if h.kiroService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Kiro Gateway service not available")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxKiroCredentialUploadSize)
	if err := c.Request.ParseMultipartForm(maxKiroCredentialUploadSize); err != nil {
		response.BadRequest(c, "Invalid credential form")
		return
	}

	input := service.KiroCredentialImportInput{
		Type:          strings.TrimSpace(c.PostForm("type")),
		RefreshToken:  strings.TrimSpace(c.PostForm("refresh_token")),
		ProfileARN:    strings.TrimSpace(c.PostForm("profile_arn")),
		Region:        strings.TrimSpace(c.PostForm("region")),
		APIRegion:     strings.TrimSpace(c.PostForm("api_region")),
		Append:        parseKiroFormBool(c.PostForm("append")),
		Activate:      parseKiroFormBool(c.PostForm("activate")),
		TwoFactorCode: strings.TrimSpace(c.PostForm("two_factor_code")),
	}

	cleanup, err := attachKiroCredentialFile(c, &input)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if cleanup != nil {
		defer cleanup()
	}
	if input.Type == "" {
		input.Type = inferKiroCredentialType(input.FileName, input.RefreshToken)
	}

	result, err := h.kiroService.ImportCredentials(c.Request.Context(), input)
	if err != nil {
		response.Error(c, mapKiroErrorStatus(err), err.Error())
		return
	}
	response.Success(c, result)
}

// Activate starts the sidecar and enables the prepared Kiro account after
// credentials have been imported.
func (h *KiroHandler) Activate(c *gin.Context) {
	if h.kiroService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Kiro Gateway service not available")
		return
	}
	result, err := h.kiroService.Activate(c.Request.Context())
	if err != nil {
		response.Error(c, mapKiroErrorStatus(err), err.Error())
		return
	}
	response.Success(c, result)
}

// StartCLILogin starts an official Kiro CLI device-flow login and returns a
// short-lived session that the admin UI can poll for the URL, code, and import
// result.
func (h *KiroHandler) StartCLILogin(c *gin.Context) {
	if h.kiroService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Kiro Gateway service not available")
		return
	}
	var req startKiroCLILoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid Kiro login request")
		return
	}
	session, err := h.kiroService.StartCLILogin(c.Request.Context(), service.KiroCLILoginInput{
		Provider:            req.Provider,
		IdentityProviderURL: req.IdentityProviderURL,
		Region:              req.Region,
		Append:              req.Append,
		Activate:            req.Activate,
	})
	if err != nil {
		response.Error(c, mapKiroErrorStatus(err), err.Error())
		return
	}
	response.Success(c, session)
}

// GetCLILoginSession returns the latest state for a device-flow login session.
func (h *KiroHandler) GetCLILoginSession(c *gin.Context) {
	if h.kiroService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Kiro Gateway service not available")
		return
	}
	session, err := h.kiroService.GetCLILoginSession(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.Error(c, mapKiroErrorStatus(err), err.Error())
		return
	}
	response.Success(c, session)
}

func attachKiroCredentialFile(c *gin.Context, input *service.KiroCredentialImportInput) (func(), error) {
	file, header, err := c.Request.FormFile("credential_file")
	if errors.Is(err, http.ErrMissingFile) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("credential file is invalid")
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	tmp, err := os.CreateTemp("", "kiro-credential-*"+ext)
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}
	defer func() {
		_ = tmp.Close()
	}()
	if _, err := io.Copy(tmp, file); err != nil {
		cleanup()
		return nil, err
	}
	if err := tmp.Chmod(0o600); err != nil {
		cleanup()
		return nil, err
	}
	input.FileName = header.Filename
	input.FilePath = tmpPath
	return cleanup, nil
}

func inferKiroCredentialType(fileName, refreshToken string) string {
	if strings.TrimSpace(refreshToken) != "" {
		return "refresh_token"
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	switch ext {
	case ".sqlite", ".sqlite3", ".db":
		return "sqlite"
	case ".json":
		return "json"
	default:
		return ""
	}
}

func parseKiroFormBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func mapKiroErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "activation") {
		return http.StatusInternalServerError
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "permission") || strings.Contains(msg, "timed out") {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}
