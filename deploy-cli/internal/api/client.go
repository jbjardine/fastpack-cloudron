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
	"os"
	"path/filepath"
	"time"
)

// Client wraps HTTP calls to the Cloudron API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// CloudronInfo holds basic server info from GET /api/v1/config.
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
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}
}

// GetCloudronInfo fetches server info to verify connectivity + auth.
func (c *Client) GetCloudronInfo() (*CloudronInfo, error) {
	resp, err := c.get("/api/v1/config")
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid API token (HTTP 401)")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var info CloudronInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return &info, nil
}

// BuildImage uploads a tarball to the Cloudron Build Service and returns the image tag.
// The Build Service builds the Docker image server-side — no local Docker needed.
func (c *Client) BuildImage(tarballPath string) (string, error) {
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

	// POST to build endpoint
	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/developer/build", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("build failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response for image tag
	var result struct {
		Image string `json:"image"`
		Tag   string `json:"tag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("invalid build response: %w", err)
	}

	imageTag := result.Image
	if imageTag == "" && result.Tag != "" {
		imageTag = result.Tag
	}
	if imageTag == "" {
		return "", fmt.Errorf("build succeeded but no image tag returned")
	}
	return imageTag, nil
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

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("install failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response for app URL
	var result struct {
		ID       string `json:"id"`
		Location string `json:"location"`
		FQDN     string `json:"fqdn"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

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
