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

// BuildImage uploads a tarball to the Build Service and returns the image tag.
// The Build Service builds the Docker image server-side — no local Docker needed.
// It uses the separate Build Service URL (e.g., devtools.DOMAIN) with its own auth.
func (c *Client) BuildImage(tarballPath string) (string, error) {
	buildURL := c.buildServiceURL
	buildToken := c.buildToken
	if buildURL == "" {
		return "", fmt.Errorf("no Build Service configured. Set build service URL with --build-service or use CLOUDRON_BUILD_SERVICE_URL env var")
	}
	if buildToken == "" {
		buildToken = c.token // fallback to Cloudron API token
	}

	// Open the tarball
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("cannot open tarball: %w", err)
	}
	defer f.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(tarballPath))
	if err != nil {
		return "", fmt.Errorf("multipart error: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", fmt.Errorf("file copy error: %w", err)
	}
	writer.Close()

	// POST to Build Service /api/v1/builds
	req, err := http.NewRequest("POST", buildURL+"/api/v1/builds", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+buildToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		return "", fmt.Errorf("build conflict: another build may be in progress (HTTP 409)")
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		return "", fmt.Errorf("build failed (HTTP %d)", resp.StatusCode)
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

	// If we got a build ID, we may need to poll for completion
	// and then push the image to the registry
	imageTag := result.Image
	if imageTag == "" && result.Tag != "" {
		imageTag = result.Tag
	}

	// If build returned an ID but no image, poll for completion
	if imageTag == "" && result.ID != "" {
		imageTag, err = c.waitForBuild(buildURL, buildToken, result.ID)
		if err != nil {
			return "", err
		}
	}

	if imageTag == "" {
		return "", fmt.Errorf("build succeeded but no image tag returned")
	}
	return imageTag, nil
}

// waitForBuild polls the Build Service until a build completes.
func (c *Client) waitForBuild(buildURL, token, buildID string) (string, error) {
	for i := 0; i < 120; i++ { // max 10 minutes (5s * 120)
		time.Sleep(5 * time.Second)

		req, err := http.NewRequest("GET", buildURL+"/api/v1/builds/"+buildID, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue // retry on network error
		}

		var status struct {
			Status string `json:"status"`
			Image  string `json:"image"`
			Tag    string `json:"tag"`
			Error  string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&status)
		resp.Body.Close()

		switch status.Status {
		case "success", "completed", "pushed":
			tag := status.Image
			if tag == "" {
				tag = status.Tag
			}
			if tag == "" {
				// Push the build to the registry
				pushReq, _ := http.NewRequest("POST", buildURL+"/api/v1/builds/"+buildID+"/push", nil)
				pushReq.Header.Set("Authorization", "Bearer "+token)
				pushResp, err := c.httpClient.Do(pushReq)
				if err != nil {
					return "", fmt.Errorf("push failed: %w", err)
				}
				var pushResult struct {
					Image string `json:"image"`
					Tag   string `json:"tag"`
				}
				json.NewDecoder(pushResp.Body).Decode(&pushResult)
				pushResp.Body.Close()
				tag = pushResult.Image
				if tag == "" {
					tag = pushResult.Tag
				}
			}
			if tag != "" {
				return tag, nil
			}
			return "", fmt.Errorf("build completed but no image tag available")
		case "error", "failed":
			return "", fmt.Errorf("build failed: %s", status.Error)
		}
		// "building", "pending" → keep polling
	}
	return "", fmt.Errorf("build timed out after 10 minutes")
}

// InstallApp installs an app on Cloudron using the built image.
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

	// Build install payload
	payload := map[string]any{
		"appStoreId":  "",
		"manifest":    manifest,
		"location":    subdomain,
		"accessRestriction": nil,
		"image":       imageTag,
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
		return "", fmt.Errorf("install failed (HTTP %d)", resp.StatusCode)
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
