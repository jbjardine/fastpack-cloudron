// Package api provides a client for the Cloudron REST API.
// Handles authentication, image building via Build Service, and app installation.
// Uses only stdlib net/http — no external dependencies.
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
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

// Client wraps HTTP calls to the Cloudron API and Build Service.
type Client struct {
	baseURL         string
	token           string
	buildServiceURL string // separate Build Service (e.g., devtools.DOMAIN)
	buildToken      string // Build Service auth token (may differ from Cloudron token)
	httpClient      *http.Client
}

// CloudronInfo holds basic server info from GET /api/v1/cloudron/status.
type CloudronInfo struct {
	Version     string `json:"version"`
	DisplayName string `json:"displayName"`
	Domain      string `json:"domain"`
}

// NewClient creates a new Cloudron API client.
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

// SetBuildService configures a separate Build Service for image building.
func (c *Client) SetBuildService(url, token string) {
	c.buildServiceURL = url
	c.buildToken = token
}

// GetCloudronInfo fetches server info to verify connectivity + auth.
// Uses /api/v1/cloudron/status for version, then /api/v1/profile for auth check.
func (c *Client) GetCloudronInfo() (*CloudronInfo, error) {
	// First check auth via /api/v1/profile
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

	// Get version info
	statusResp, err := c.get("/api/v1/cloudron/status")
	if err != nil {
		return nil, fmt.Errorf("cannot get Cloudron status: %w", err)
	}
	defer statusResp.Body.Close()

	var info CloudronInfo
	if err := json.NewDecoder(statusResp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	// Extract domain from baseURL for convenience
	if info.Domain == "" {
		u, _ := url.Parse(c.baseURL)
		if u != nil {
			host := u.Hostname()
			// Strip "my." prefix to get the bare domain
			if strings.HasPrefix(host, "my.") {
				info.Domain = host[3:]
			} else {
				info.Domain = host
			}
		}
	}

	return &info, nil
}

// VerifyBuildService checks connectivity and auth to the Build Service.
// Call this before BuildImage to give early, actionable error messages.
// The Build Service uses ?accessToken= query params (not Bearer headers).
func (c *Client) VerifyBuildService() error {
	if c.buildServiceURL == "" {
		return fmt.Errorf("no Build Service configured.\n   Set it with CLOUDRON_BUILD_SERVICE_URL or enter it in the wizard")
	}
	if c.buildToken == "" {
		return fmt.Errorf("no Build Service token provided.\n   Get it from the Build Service Setup page: %s\n   Or set CLOUDRON_BUILD_TOKEN env var", c.buildServiceURL)
	}

	// Use /api/v1/profile with accessToken — same endpoint the cloudron CLI uses
	req, err := http.NewRequest("GET", c.buildServiceURL+"/api/v1/profile?accessToken="+url.QueryEscape(c.buildToken), nil)
	if err != nil {
		return fmt.Errorf("invalid Build Service URL: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach Build Service at %s: %w", c.buildServiceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("Build Service auth failed (HTTP %d).\n   Your Build Service token may be invalid or expired.\n   Get a new token from: %s", resp.StatusCode, c.buildServiceURL)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Build Service returned HTTP %d.\n   Check that %s is the correct URL", resp.StatusCode, c.buildServiceURL)
	}
	return nil
}

// BuildImage uploads a tarball to the Build Service and returns the image tag.
// The Build Service builds the Docker image server-side — no local Docker needed.
// It uses the separate Build Service URL (e.g., devtools.DOMAIN) with its own auth.
func (c *Client) BuildImage(tarballPath string) (string, error) {
	buildURL := c.buildServiceURL
	buildToken := c.buildToken
	if buildURL == "" {
		return "", fmt.Errorf("no Build Service configured.\n   Set it with CLOUDRON_BUILD_SERVICE_URL or enter it in the wizard")
	}
	if buildToken == "" {
		return "", fmt.Errorf("no Build Service token.\n   Get it from the Build Service Setup page: %s\n   Or set CLOUDRON_BUILD_TOKEN env var", buildURL)
	}

	// Open the tarball
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("cannot open tarball: %w", err)
	}
	defer f.Close()

	// Create multipart form matching the Cloudron Build Service API:
	//   sourceArchive: the tarball file
	//   dockerImageRepo: registry/image name
	//   dockerImageTag: image tag
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add metadata fields required by the Cloudron Build Service API.
	// The dockerImageRepo is derived from the Build Service URL (works for
	// builder-registry combo setups). In production with a separate registry,
	// this should be configurable.
	u, _ := url.Parse(buildURL)
	imageRepo := u.Host + "/fastpack-app"
	writer.WriteField("dockerImageRepo", imageRepo)
	writer.WriteField("dockerImageTag", "latest")
	writer.WriteField("buildArgs", "{}")

	// Add the source archive
	part, err := writer.CreateFormFile("sourceArchive", filepath.Base(tarballPath))
	if err != nil {
		return "", fmt.Errorf("multipart error: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("file copy error: %w", err)
	}
	writer.Close()

	// POST to Build Service /api/v1/builds?accessToken=...
	// Cloudron Build Service uses accessToken query param, not Bearer headers
	req, err := http.NewRequest("POST", buildURL+"/api/v1/builds?accessToken="+url.QueryEscape(buildToken), &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 401, 403:
		return "", fmt.Errorf("Build Service auth failed (HTTP %d).\n   Your Build Service token may be invalid or expired.\n   Get a new token from: %s", resp.StatusCode, buildURL)
	case 409:
		return "", fmt.Errorf("build conflict: another build may be in progress (HTTP 409)")
	case 413:
		return "", fmt.Errorf("package too large for Build Service (HTTP 413). Max size may be exceeded")
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("build failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response for build ID and image tag
	var result struct {
		ID    string `json:"id"`
		Image string `json:"image"`
		Tag   string `json:"tag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid build response: %w", err)
	}

	// If the response includes an image tag directly, use it.
	// Otherwise poll for build completion.
	imageTag := result.Image
	if imageTag == "" && result.Tag != "" {
		imageTag = result.Tag
	}

	if imageTag == "" && result.ID != "" {
		imageTag, err = c.waitForBuild(buildURL, buildToken, result.ID)
		if err != nil {
			return "", err
		}
	}

	// If still no image tag, construct it from what we sent.
	// The Cloudron Build Service doesn't always return the tag in poll responses —
	// the official cloudron CLI constructs it from the request parameters.
	if imageTag == "" {
		imageTag = imageRepo + ":latest"
	}

	return imageTag, nil
}

// waitForBuild polls the Build Service until a build completes.
// Uses exponential backoff (3s → 5s → 8s → 10s cap) and prints progress dots.
func (c *Client) waitForBuild(buildURL, token, buildID string) (string, error) {
	interval := 3 * time.Second
	maxInterval := 10 * time.Second
	deadline := time.Now().Add(10 * time.Minute)
	networkErrors := 0

	for time.Now().Before(deadline) {
		time.Sleep(interval)
		fmt.Print(".")

		req, err := http.NewRequest("GET", buildURL+"/api/v1/builds/"+buildID+"?accessToken="+url.QueryEscape(token), nil)
		if err != nil {
			return "", err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			networkErrors++
			if networkErrors > 5 {
				return "", fmt.Errorf("too many network errors polling build status: %w", err)
			}
			interval = min(interval*3/2, maxInterval)
			continue
		}

		var status struct {
			Status string `json:"status"`
			Image  string `json:"image"`
			Tag    string `json:"tag"`
			Error  string `json:"error"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&status); decErr != nil {
			resp.Body.Close()
			interval = min(interval*3/2, maxInterval)
			continue
		}
		resp.Body.Close()
		networkErrors = 0

		switch status.Status {
		case "success", "completed", "pushed":
			tag := status.Image
			if tag == "" {
				tag = status.Tag
			}
			// Build completed — return whatever tag we have (may be empty;
			// caller will construct it from the request parameters)
			return tag, nil
		case "error", "failed":
			return "", fmt.Errorf("build failed: %s", status.Error)
		}
		// "building", "pending" → backoff and keep polling
		interval = min(interval*3/2, maxInterval)
	}
	return "", fmt.Errorf("build timed out after 10 minutes")
}

// InstallApp installs an app on Cloudron using the built image.
// Cloudron v9 API: POST /api/v1/apps with {subdomain, domain, manifest}.
// The dockerImage goes INSIDE the manifest object, not as a separate field.
func (c *Client) InstallApp(manifestPath, imageTag, subdomain string) (string, error) {
	// Read manifest
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("cannot read manifest: %w", err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return "", fmt.Errorf("invalid manifest JSON: %w", err)
	}

	// Set dockerImage inside the manifest (Cloudron v9 API requirement)
	manifest["dockerImage"] = imageTag

	// Extract domain from baseURL
	u, _ := url.Parse(c.baseURL)
	domain := u.Hostname()
	if strings.HasPrefix(domain, "my.") {
		domain = domain[3:]
	}

	// Build install payload (Cloudron v9 format)
	payload := map[string]any{
		"appStoreId":       "",
		"manifest":         manifest,
		"subdomain":        subdomain,
		"domain":           domain,
		"accessRestriction": nil,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// POST /api/v1/apps
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/apps", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

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

	// Parse response for app URL
	var result struct {
		ID       string `json:"id"`
		Location string `json:"location"`
		FQDN     string `json:"fqdn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("https://%s", subdomain), nil
	}

	if result.FQDN != "" {
		return "https://" + result.FQDN, nil
	}
	return fmt.Sprintf("https://%s (app ID: %s)", subdomain, result.ID), nil
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
