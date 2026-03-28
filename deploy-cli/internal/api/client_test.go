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

// === NewClient tests ===

func TestNewClient_Default(t *testing.T) {
	c := NewClient("https://example.com", "tok", false)
	if c.baseURL != "https://example.com" {
		t.Fatalf("baseURL=%q", c.baseURL)
	}
	if c.token != "tok" {
		t.Fatalf("token=%q", c.token)
	}
}

func TestNewClient_EmptyToken(t *testing.T) {
	c := NewClient("https://example.com", "", false)
	if c.token != "" {
		t.Fatalf("expected empty token, got %q", c.token)
	}
}

// === Login tests ===

func TestLogin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/auth/login" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["username"] != "admin" || payload["password"] != "secret" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"message": "Invalid credentials"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok-123"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Login("admin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "tok-123" {
		t.Fatalf("token=%q, want tok-123", c.token)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid credentials"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Login("admin", "wrong")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid username or password") {
		t.Fatalf("err=%q", err.Error())
	}
}

func TestLogin_2FARequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["totpToken"] == "" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"message": "A totpToken must be provided"})
			return
		}
		if payload["totpToken"] != "123456" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"message": "Invalid totpToken"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok-2fa"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}

	// First call: should return Err2FARequired
	err := c.Login("admin", "secret")
	if err != Err2FARequired {
		t.Fatalf("expected Err2FARequired, got %v", err)
	}

	// Second call with valid TOTP
	err = c.LoginWith2FA("admin", "secret", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "tok-2fa" {
		t.Fatalf("token=%q, want tok-2fa", c.token)
	}
}

func TestLogin_2FAInvalidCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid totpToken"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.LoginWith2FA("admin", "secret", "000000")
	if err == nil {
		t.Fatal("expected error for invalid 2FA code")
	}
	// Must NOT return Err2FARequired — that would cause an infinite retry loop
	if err == Err2FARequired {
		t.Fatal("invalid TOTP should return auth error, not Err2FARequired")
	}
	if !strings.Contains(err.Error(), "invalid username or password") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestLogin_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Login("admin", "secret")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}

func TestLogin_TokenFieldFallback(t *testing.T) {
	// Some Cloudron versions use "token" instead of "accessToken"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": "fallback-tok"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Login("admin", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if c.token != "fallback-tok" {
		t.Fatalf("token=%q, want fallback-tok", c.token)
	}
}

func TestLogin_NoTokenReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	err := c.Login("admin", "secret")
	if err == nil || !strings.Contains(err.Error(), "no token returned") {
		t.Fatalf("expected no token error, got %v", err)
	}
}

// === GetCloudronInfo tests ===

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

// === InstallApp tests (multipart sourceArchive) ===

func TestInstallApp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/apps" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		// Verify multipart
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Fatalf("expected multipart/form-data, got %q", ct)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		// Verify fields
		if r.FormValue("subdomain") != "myapp" {
			t.Fatalf("subdomain=%q", r.FormValue("subdomain"))
		}
		if r.FormValue("domain") == "" {
			t.Fatal("missing domain field")
		}
		manifest := r.FormValue("manifest")
		if manifest == "" {
			t.Fatal("missing manifest field")
		}
		// Verify manifest is valid JSON
		var m map[string]any
		if err := json.Unmarshal([]byte(manifest), &m); err != nil {
			t.Fatalf("manifest is not valid JSON: %v", err)
		}
		// Verify sourceArchive file
		if _, _, err := r.FormFile("sourceArchive"); err != nil {
			t.Fatalf("no sourceArchive field: %v", err)
		}
		// Verify accessRestriction
		ar := r.FormValue("accessRestriction")
		if ar == "" {
			t.Fatal("missing accessRestriction field")
		}
		// Verify auth
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok" {
			t.Fatalf("auth=%q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]string{"id": "app-123", "fqdn": "myapp.example.com"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test.app","title":"Test","version":"1.0.0"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("fake tarball content"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	u, err := c.InstallApp(manifestFile, tarball, "myapp")
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
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test.app"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.InstallApp(manifestFile, tarball, "taken")
	if err == nil {
		t.Fatal("expected 409 error")
	}
}

func TestInstallApp_InvalidManifest(t *testing.T) {
	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{not json`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.InstallApp(manifestFile, tarball, "sub")
	if err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestInstallApp_MissingManifest(t *testing.T) {
	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.InstallApp("/nonexistent", "/nonexistent.tar.gz", "sub")
	if err == nil {
		t.Fatal("expected file not found error")
	}
}

func TestInstallApp_MissingTarball(t *testing.T) {
	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test"}`), 0644)

	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.InstallApp(manifestFile, "/nonexistent.tar.gz", "sub")
	if err == nil {
		t.Fatal("expected file not found error")
	}
}

func TestInstallApp_FallbackURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]string{"id": "app-456"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test.app"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	u, err := c.InstallApp(manifestFile, tarball, "myapp")
	if err != nil {
		t.Fatal(err)
	}
	// Should contain the subdomain and app ID
	if !strings.Contains(u, "myapp") || !strings.Contains(u, "app-456") {
		t.Fatalf("url=%q", u)
	}
}

func TestInstallApp_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.InstallApp(manifestFile, tarball, "myapp")
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}

// === UpdateApp tests (multipart sourceArchive) ===

func TestUpdateApp_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/apps/app-123/update" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		// Verify multipart
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Fatalf("expected multipart/form-data, got %q", ct)
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		manifest := r.FormValue("manifest")
		if manifest == "" {
			t.Fatal("missing manifest field")
		}
		if r.FormValue("force") != "true" {
			t.Fatal("expected force=true")
		}
		if _, _, err := r.FormFile("sourceArchive"); err != nil {
			t.Fatalf("no sourceArchive field: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(202)
		json.NewEncoder(w).Encode(map[string]string{"taskId": "42"})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test.app","title":"Test","version":"2.0.0"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("tarball data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.UpdateApp("app-123", manifestFile, tarball)
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
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test"}`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: srv.URL, token: "tok", httpClient: srv.Client()}
	_, err := c.UpdateApp("app-123", manifestFile, tarball)
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected 500 error, got %v", err)
	}
}

func TestUpdateApp_InvalidManifest(t *testing.T) {
	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`not json`), 0644)
	tarball := filepath.Join(tmp, "test.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.UpdateApp("app-123", manifestFile, tarball)
	if err == nil {
		t.Fatal("expected JSON error")
	}
}

func TestUpdateApp_MissingTarball(t *testing.T) {
	tmp := t.TempDir()
	manifestFile := filepath.Join(tmp, "CloudronManifest.json")
	os.WriteFile(manifestFile, []byte(`{"id":"io.test"}`), 0644)

	c := &Client{baseURL: "https://example.com", token: "tok", httpClient: http.DefaultClient}
	_, err := c.UpdateApp("app-123", manifestFile, "/nonexistent.tar.gz")
	if err == nil {
		t.Fatal("expected file not found error")
	}
}

// === extractDomain tests ===

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://my.example.com", "example.com"},
		{"https://cloudron.io", "cloudron.io"},
		{"https://my.192.168.1.50.nip.io", "192.168.1.50.nip.io"},
		{"https://example.com:8443", "example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractDomain(tt.input)
			if got != tt.want {
				t.Fatalf("extractDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
