// Package api provides a client for the Cloudron REST API.
// Handles authentication via login, and app installation via direct sourceArchive upload.
// Uses only stdlib net/http — no external dependencies.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Err2FARequired is returned by Login when the server requires a TOTP code.
var Err2FARequired = errors.New("2FA required")

// Client wraps HTTP calls to the Cloudron API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// CloudronInfo holds basic server info from GET /api/v1/cloudron/status.
type CloudronInfo struct {
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
	Domain      string `json:"domain"`
}

// NewClient creates a new Cloudron API client.
// If token is provided (legacy flow), it's used directly for Bearer auth.
func NewClient(baseURL, token string, allowSelfSigned bool) *Client {
	transport := &http.Transport{}
	if allowSelfSigned {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout:   300 * time.Second,
			Transport: transport,
		},
	}
}

// Login authenticates with username/password and stores the access token.
// If 2FA is enabled, returns Err2FARequired on the first call.
// Call LoginWith2FA to retry with the TOTP code.
func (c *Client) Login(username, password string) error {
	return c.doLogin(username, password, "")
}

// LoginWith2FA authenticates with username/password and a TOTP code.
func (c *Client) LoginWith2FA(username, password, totpToken string) error {
	return c.doLogin(username, password, totpToken)
}

func (c *Client) doLogin(username, password, totpToken string) error {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	if totpToken != "" {
		payload["totpToken"] = totpToken
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("reading login response: %w", err)
	}

	if resp.StatusCode == 401 {
		// Parse the error to distinguish 2FA-required from invalid credentials.
		// Cloudron returns "A totpToken must be provided" when 2FA is needed
		// but "Invalid totpToken" when the code is wrong.
		var apiErr struct {
			Message string `json:"message"`
		}
		json.Unmarshal(respBody, &apiErr)
		msg := apiErr.Message
		if strings.Contains(msg, "totpToken must be provided") {
			return Err2FARequired
		}
		return fmt.Errorf("invalid username or password")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Token       string `json:"token"`
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("invalid login response: %w", err)
	}

	token := result.AccessToken
	if token == "" {
		token = result.Token
	}
	if token == "" {
		return fmt.Errorf("login succeeded but no token returned")
	}
	c.token = token
	return nil
}

// GetCloudronInfo fetches server info to verify connectivity + auth.
func (c *Client) GetCloudronInfo() (*CloudronInfo, error) {
	profileResp, err := c.get("/api/v1/profile")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid API token (HTTP 401)")
	}
	if profileResp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d from Cloudron API", profileResp.StatusCode)
	}

	statusResp, err := c.get("/api/v1/cloudron/status")
	if err != nil {
		return nil, fmt.Errorf("cannot get Cloudron status: %w", err)
	}
	defer statusResp.Body.Close()

	if statusResp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d from /api/v1/cloudron/status", statusResp.StatusCode)
	}

	var info CloudronInfo
	if err := json.NewDecoder(statusResp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	if info.Domain == "" {
		u, _ := url.Parse(c.baseURL)
		if u != nil {
			host := u.Hostname()
			if strings.HasPrefix(host, "my.") {
				info.Domain = host[3:]
			} else {
				info.Domain = host
			}
		}
	}

	return &info, nil
}

// AppInfo holds basic information about an installed Cloudron app.
type AppInfo struct {
	ID        string `json:"id"`
	Subdomain string `json:"subdomain"`
	Domain    string `json:"domain"`
	FQDN     string `json:"fqdn"`
}

// FindAppBySubdomain checks if an app is already installed at the given subdomain.
func (c *Client) FindAppBySubdomain(subdomain string) (*AppInfo, error) {
	resp, err := c.get("/api/v1/apps")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list apps (HTTP %d)", resp.StatusCode)
	}

	var result struct {
		Apps []AppInfo `json:"apps"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	for _, app := range result.Apps {
		if app.Subdomain == subdomain {
			return &app, nil
		}
	}
	return nil, nil
}

// InstallApp installs a NEW app on Cloudron using direct sourceArchive upload.
// Cloudron v9 API: POST /api/v1/apps (multipart/form-data).
// Fields: manifest (JSON string), subdomain, domain, accessRestriction, sourceArchive (file).
func (c *Client) InstallApp(manifestPath, tarballPath, subdomain string) (string, error) {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("cannot read manifest: %w", err)
	}

	// Validate manifest is valid JSON
	if !json.Valid(manifestData) {
		return "", fmt.Errorf("invalid manifest JSON in %s", manifestPath)
	}

	// Extract domain from baseURL
	domain := extractDomain(c.baseURL)

	// Build multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("manifest", string(manifestData))
	writer.WriteField("subdomain", subdomain)
	writer.WriteField("domain", domain)
	writer.WriteField("accessRestriction", `{"users":[],"groups":[]}`)

	// Add sourceArchive file
	if err := addFileField(writer, "sourceArchive", tarballPath); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/apps", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("install request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		return "", fmt.Errorf("subdomain already in use (HTTP 409)")
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("install failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID   string `json:"id"`
		FQDN string `json:"fqdn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("https://%s.%s", subdomain, domain), nil
	}

	if result.FQDN != "" {
		return "https://" + result.FQDN, nil
	}
	return fmt.Sprintf("https://%s.%s (app ID: %s)", subdomain, domain, result.ID), nil
}

// UpdateApp updates an existing app with a new sourceArchive.
// Cloudron v9 API: POST /api/v1/apps/{id}/update (multipart/form-data).
func (c *Client) UpdateApp(appID, manifestPath, tarballPath string) (string, error) {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("cannot read manifest: %w", err)
	}

	if !json.Valid(manifestData) {
		return "", fmt.Errorf("invalid manifest JSON in %s", manifestPath)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writer.WriteField("manifest", string(manifestData))
	writer.WriteField("force", "true")

	// Add sourceArchive file
	if err := addFileField(writer, "sourceArchive", tarballPath); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/apps/"+appID+"/update", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("update request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("update failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	return fmt.Sprintf("https://%s (updated)", appID), nil
}

// addFileField adds a file from disk as a multipart form file field.
func addFileField(writer *multipart.Writer, fieldName, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", fieldName, err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("multipart error for %s: %w", fieldName, err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return fmt.Errorf("file copy error for %s: %w", fieldName, err)
	}
	return nil
}

// extractDomain returns the bare domain from a Cloudron base URL.
func extractDomain(baseURL string) string {
	u, _ := url.Parse(baseURL)
	if u == nil {
		return ""
	}
	host := u.Hostname()
	if strings.HasPrefix(host, "my.") {
		return host[3:]
	}
	return host
}

// get performs an authenticated GET request.
func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.httpClient.Do(req)
}
