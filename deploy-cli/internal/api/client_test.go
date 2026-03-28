package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewClient_Default(t *testing.T) {
	c := NewClient("https://example.com", "tok", false)
	if c.baseURL != "https://example.com" {
		t.Fatalf("baseURL=%q", c.baseURL)
	}
	if c.token != "tok" {
		t.Fatalf("token=%q", c.token)
	}
}

func TestGetCloudronInfo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer testtoken" {
			t.Fatalf("auth=%q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CloudronInfo{
			Version:     "8.0.0",
			DisplayName: "Dev",
			Domain:      "example.com",
		})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "testtoken", httpClient: srv.Client()}
	info, err := c.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Domain != "example.com" {
		t.Fatalf("domain=%q", info.Domain)
	}
	if info.Version != "8.0.0" {
		t.Fatalf("version=%q", info.Version)
	}
}

func TestGetCloudronInfo_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "bad", httpClient: srv.Client()}
	_, err := c.GetCloudronInfo()
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "invalid API token (HTTP 401)" {
		t.Fatalf("err=%q", got)
	}
}

func TestGetCloudronInfo_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.GetCloudronInfo()
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestGetCloudronInfo_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{not json}`)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.GetCloudronInfo()
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
}

func TestBuildImage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("method=%s", r.Method)
		}
		if r.URL.Path != "/api/v1/developer/build" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		// Verify multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Fatalf("no file field: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"image": "registry.example.com/app:v1.0.0"})
	}))
	defer srv.Close()

	// Create a temp tarball
	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("fake tarball content"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	tag, err := c.BuildImage(tarball)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "registry.example.com/app:v1.0.0" {
		t.Fatalf("tag=%q", tag)
	}
}

func TestBuildImage_FileNotFound(t *testing.T) {
	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.BuildImage("/nonexistent/path.tar.gz")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBuildImage_Conflict409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected 409 error")
	}
	if got := err.Error(); got != "build conflict: another build may be in progress (HTTP 409)" {
		t.Fatalf("err=%q", got)
	}
}

func TestBuildImage_NoImageTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected error for missing image tag")
	}
}

func TestBuildImage_TagFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag": "fallback:latest"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	tag, err := c.BuildImage(tarball)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "fallback:latest" {
		t.Fatalf("tag=%q, want fallback:latest", tag)
	}
}

func TestInstallApp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/apps" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["location"] != "myapp" {
			t.Fatalf("location=%v", payload["location"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"id":   "app-123",
			"fqdn": "myapp.example.com",
		})
	}))
	defer srv.Close()

	// Create a manifest
	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test.app","title":"Test","version":"1.0.0"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	url, err := c.InstallApp(manifest, "registry/app:v1", "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://myapp.example.com" {
		t.Fatalf("url=%q", url)
	}
}

func TestInstallApp_SubdomainConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test.app"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.InstallApp(manifest, "img:v1", "taken")
	if err == nil {
		t.Fatal("expected 409 error")
	}
	if got := err.Error(); got != "subdomain already in use (HTTP 409)" {
		t.Fatalf("err=%q", got)
	}
}

func TestInstallApp_InvalidManifest(t *testing.T) {
	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{not json`), 0644)

	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.InstallApp(manifest, "img:v1", "sub")
	if err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestInstallApp_MissingManifest(t *testing.T) {
	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.InstallApp("/nonexistent", "img:v1", "sub")
	if err == nil {
		t.Fatal("expected file not found error")
	}
}

func TestInstallApp_FallbackURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "app-456"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test.app"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	url, err := c.InstallApp(manifest, "img:v1", "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if url != "https://myapp (app ID: app-456)" {
		t.Fatalf("url=%q", url)
	}
}
