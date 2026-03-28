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
// Handles both the builds list (for auto-detect) and the build POST.
func buildServiceMock(t *testing.T, imageTag string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/builds":
			// Return empty builds list (no previous builds for auto-detect)
			json.NewEncoder(w).Encode(map[string]any{"builds": []any{}})
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
		w.Header().Set("Content-Type", "application/json")
		// Handle auto-detect GET (returns empty builds list)
		if r.Method == "GET" && r.URL.Path == "/api/v1/builds" {
			json.NewEncoder(w).Encode(map[string]any{"builds": []any{}})
			return
		}
		if r.Method != "POST" || r.URL.Path != "/api/v1/builds" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		// Verify accessToken query param
		if r.URL.Query().Get("accessToken") != "btok" {
			t.Fatalf("expected accessToken=btok, got %q", r.URL.Query().Get("accessToken"))
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		// Build Service expects "sourceArchive" field, not "file"
		if _, _, err := r.FormFile("sourceArchive"); err != nil {
			t.Fatalf("no sourceArchive field: %v", err)
		}
		// Verify metadata fields
		if r.FormValue("dockerImageRepo") == "" {
			t.Fatal("missing dockerImageRepo field")
		}
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

func TestBuildImage_NoToken(t *testing.T) {
	c := &Client{baseURL: "x", token: "tok", buildServiceURL: "https://build.example.com", httpClient: http.DefaultClient}
	_, err := c.BuildImage("/some/path.tar.gz")
	if err == nil || !strings.Contains(err.Error(), "no Build Service token") {
		t.Fatalf("expected build token error, got %v", err)
	}
}

func TestBuildImage_Auth401(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "bad-token", httpClient: buildSrv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestVerifyBuildService_Success(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build Service uses ?accessToken= query param
		if r.URL.Query().Get("accessToken") != "valid-token" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"username": "admin"})
	}))
	defer buildSrv.Close()

	c := &Client{buildServiceURL: buildSrv.URL, buildToken: "valid-token", httpClient: buildSrv.Client()}
	if err := c.VerifyBuildService(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBuildService_AuthFailure(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer buildSrv.Close()

	c := &Client{buildServiceURL: buildSrv.URL, buildToken: "bad-token", httpClient: buildSrv.Client()}
	err := c.VerifyBuildService()
	if err == nil || !strings.Contains(err.Error(), "auth failed") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestDetectImageRepo_FromPreviousBuilds(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"builds": []map[string]string{
				{"dockerImageRepo": "old-repo/app", "status": "failed"},
				{"dockerImageRepo": "docker.io/myuser/myapp", "status": "success"},
			},
		})
	}))
	defer buildSrv.Close()

	c := &Client{buildServiceURL: buildSrv.URL, buildToken: "tok", httpClient: buildSrv.Client()}
	repo := c.detectImageRepo()
	if repo != "docker.io/myuser/myapp" {
		t.Fatalf("expected auto-detected repo, got %q", repo)
	}
}

func TestDetectImageRepo_FallbackToHostname(t *testing.T) {
	// Server returns empty builds list
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"builds": []any{}})
	}))
	defer buildSrv.Close()

	c := &Client{buildServiceURL: buildSrv.URL, buildToken: "tok", httpClient: buildSrv.Client()}
	repo := c.detectImageRepo()
	if !strings.Contains(repo, "127.0.0.1") || !strings.Contains(repo, "/fastpack-app") {
		t.Fatalf("expected hostname fallback, got %q", repo)
	}
}

func TestDetectImageRepo_EnvVarOverride(t *testing.T) {
	t.Setenv("DOCKER_IMAGE_REPO", "custom-registry.io/my-app")

	c := &Client{buildServiceURL: "https://devtools.example.com", buildToken: "tok"}
	repo := c.detectImageRepo()
	if repo != "custom-registry.io/my-app" {
		t.Fatalf("expected env var override, got %q", repo)
	}
}

func TestVerifyBuildService_UsesAccessTokenQueryParam(t *testing.T) {
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the token is sent as query param, not Bearer header
		if r.Header.Get("Authorization") != "" {
			t.Fatal("Build Service should not use Authorization header")
		}
		if r.URL.Query().Get("accessToken") == "" {
			t.Fatal("Build Service should receive accessToken query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": "admin"})
	}))
	defer buildSrv.Close()

	c := &Client{buildServiceURL: buildSrv.URL, buildToken: "test-token", httpClient: buildSrv.Client()}
	if err := c.VerifyBuildService(); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyBuildService_NoURL(t *testing.T) {
	c := &Client{buildToken: "tok"}
	err := c.VerifyBuildService()
	if err == nil || !strings.Contains(err.Error(), "no Build Service configured") {
		t.Fatalf("expected no-URL error, got %v", err)
	}
}

func TestVerifyBuildService_NoToken(t *testing.T) {
	c := &Client{buildServiceURL: "https://build.example.com"}
	err := c.VerifyBuildService()
	if err == nil || !strings.Contains(err.Error(), "no Build Service token") {
		t.Fatalf("expected no-token error, got %v", err)
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

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "btok", httpClient: buildSrv.Client()}
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

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "btok", httpClient: buildSrv.Client()}
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
		w.Header().Set("Content-Type", "application/json")
		// Handle auto-detect GET
		if r.Method == "GET" && r.URL.Path == "/api/v1/builds" {
			json.NewEncoder(w).Encode(map[string]any{"builds": []any{}})
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Fatalf("expected multipart/form-data, got %q", ct)
		}
		json.NewEncoder(w).Encode(map[string]string{"image": "img:v1"})
	}))
	defer buildSrv.Close()

	tmp := t.TempDir()
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "btok", httpClient: buildSrv.Client()}
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

	c := &Client{baseURL: "x", token: "tok", buildServiceURL: buildSrv.URL, buildToken: "btok", httpClient: buildSrv.Client()}
	_, err := c.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 in error, got %q", err.Error())
	}
}

// === InstallApp tests ===

// === FindAppBySubdomain tests ===

func TestFindAppBySubdomain_Found(t *testing.T) {
	srv := cloudronMock(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == "GET" && r.URL.Path == "/api/v1/apps" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"apps": []map[string]string{
					{"id": "app-1", "subdomain": "other"},
					{"id": "app-2", "subdomain": "myapp", "fqdn": "myapp.example.com"},
				},
			})
			return true
		}
		return false
	})
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	app, err := c.FindAppBySubdomain("myapp")
	if err != nil {
		t.Fatal(err)
	}
	if app == nil {
		t.Fatal("expected to find app")
	}
	if app.ID != "app-2" {
		t.Fatalf("id=%q", app.ID)
	}
}

func TestFindAppBySubdomain_NotFound(t *testing.T) {
	srv := cloudronMock(t, func(w http.ResponseWriter, r *http.Request) bool {
		if r.Method == "GET" && r.URL.Path == "/api/v1/apps" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"apps": []map[string]string{{"id": "app-1", "subdomain": "other"}},
			})
			return true
		}
		return false
	})
	defer srv.Close()

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	app, err := c.FindAppBySubdomain("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if app != nil {
		t.Fatal("expected nil for nonexistent subdomain")
	}
}

// === UpdateApp tests ===

func TestUpdateApp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/apps/app-123/update" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		m, _ := payload["manifest"].(map[string]any)
		if m["dockerImage"] != "registry/app:v2" {
			t.Fatalf("dockerImage=%v", m["dockerImage"])
		}
		if payload["force"] != true {
			t.Fatal("expected force=true")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]string{"taskId": "42"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test.app","title":"Test","version":"2.0.0"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.UpdateApp("app-123", manifest, "registry/app:v2")
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdateApp_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifest := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifest, []byte(`{"id":"io.test"}`), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.UpdateApp("app-123", manifest, "img:v1")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
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
		// Cloudron v9: subdomain + domain instead of location
		if payload["subdomain"] != "myapp" {
			t.Fatalf("subdomain=%v", payload["subdomain"])
		}
		// Check dockerImage is set inside manifest
		m, _ := payload["manifest"].(map[string]any)
		if m["dockerImage"] != "registry/app:v1" {
			t.Fatalf("dockerImage=%v", m["dockerImage"])
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
		if payload["subdomain"] != "myapp" {
			t.Fatalf("subdomain=%v", payload["subdomain"])
		}
		// Cloudron v9: dockerImage goes inside the manifest
		m, _ := payload["manifest"].(map[string]any)
		if m["dockerImage"] != "registry/app:v1" {
			t.Fatalf("dockerImage=%v", m["dockerImage"])
		}
		if payload["domain"] == nil || payload["domain"] == "" {
			t.Fatal("missing domain field")
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
