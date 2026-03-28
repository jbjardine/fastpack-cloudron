package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cloudronMock returns a mock Cloudron API server that handles the v9 endpoints.
func cloudronMock(t *testing.T, opts ...func(w http.ResponseWriter, r *http.Request) bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let custom handlers override
		for _, h := range opts {
			if h(w, r) {
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile":
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})
		case r.Method == "GET" && r.URL.Path == "/api/v1/cloudron/status":
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.1.5", DisplayName: "Test"})
		default:
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
		}
	}))
}

// buildServiceMock returns a mock Build Service server.
func buildServiceMock(t *testing.T, imageTag string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/builds":
			json.NewEncoder(w).Encode(map[string]string{"image": imageTag})
		default:
			w.WriteHeader(404)
		}
	}))
}

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
	srv := cloudronMock(t)
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "testtoken", httpClient: srv.Client()}
	info, err := c.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "9.1.5" {
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

func TestGetCloudronInfo_VerifiesAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer correct-token" {
			t.Fatalf("expected 'Bearer correct-token', got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/profile" {
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})
		} else {
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.0", DisplayName: "T"})
		}
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "correct-token", httpClient: srv.Client()}
	_, err := c.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}
}

// === BuildImage tests ===

func TestBuildImage_Success(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/builds" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if _, _, err := r.FormFile("file"); err != nil {
			t.Fatalf("no file field: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"image": "registry.example.com/app:v1.0.0"})
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("fake tarball content"), 0644)

	c := &Client{baseURL: "https://unused", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "btok", httpClient: buildSrv.Client()}
	tag, err := c.BuildImage(tarball)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "registry.example.com/app:v1.0.0" {
		t.Fatalf("tag=%q", tag)
	}
}

func TestBuildImage_NoBuildService(t *testing.T) {
	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.BuildImage("/some/path.tar.gz")
	if err == nil || !strings.Contains(err.Error(), "no Build Service configured") {
		t.Fatalf("expected build service error, got %v", err)
	}
}

func TestBuildImage_FileNotFound(t *testing.T) {
	c := &Client{baseURL: "x", token: "tok", buildServiceURL: "https://build.example.com", httpClient: http.DefaultClient}
	_, err := c.BuildImage("/nonexistent/path.tar.gz")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBuildImage_Conflict409(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, httpClient: buildSrv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected 409 error")
	}
	if got := err.Error(); got != "build conflict: another build may be in progress (HTTP 409)" {
		t.Fatalf("err=%q", got)
	}
}

func TestBuildImage_TagFallback(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"tag": "fallback:latest"})
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, httpClient: buildSrv.Client()}
	tag, err := c.BuildImage(tarball)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "fallback:latest" {
		t.Fatalf("tag=%q", tag)
	}
}

func TestBuildImage_VerifiesContentType(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Fatalf("expected multipart/form-data, got %q", ct)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"image": "img:v1"})
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, httpClient: buildSrv.Client()}
	_, err := c.BuildImage(tarball)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildImage_ServerError500(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, httpClient: buildSrv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

// === InstallApp tests ===

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
		json.NewEncoder(w).Encode(map[string]string{"id": "app-123", "fqdn": "myapp.example.com"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test.app","title":"Test","version":"1.0.0"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	u, err := c.InstallApp(manifest, "registry/app:v1", "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://myapp.example.com" {
		t.Fatalf("url=%q", u)
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

func TestInstallApp_VerifiesPayloadStructure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if _, ok := payload["manifest"]; !ok {
			t.Fatal("missing manifest")
		}
		if payload["location"] != "myapp" {
			t.Fatalf("location=%v", payload["location"])
		}
		if payload["image"] != "registry/app:v1" {
			t.Fatalf("image=%v", payload["image"])
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"fqdn": "myapp.example.com"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.InstallApp(manifest, "registry/app:v1", "myapp")
	if err != nil {
		t.Fatal(err)
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
	u, err := c.InstallApp(manifest, "img:v1", "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://myapp (app ID: app-456)" {
		t.Fatalf("url=%q", u)
	}
}
