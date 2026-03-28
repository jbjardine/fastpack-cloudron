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

// TestFullDeploymentFlow_V2 simulates the complete v2.0 flow:
// Login → GetInfo → Install (multipart with sourceArchive).
func TestFullDeploymentFlow_V2(t *testing.T) {
	var loginCalls, infoCalls, installCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/auth/login":
			loginCalls++
			var payload map[string]string
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["username"] != "admin" || payload["password"] != "secret" {
				w.WriteHeader(401)
				json.NewEncoder(w).Encode(map[string]string{"message": "Invalid credentials"})
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"accessToken": "session-tok-abc"})

		case r.Method == "GET" && r.URL.Path == "/api/v1/profile":
			infoCalls++
			auth := r.Header.Get("Authorization")
			if auth != "Bearer session-tok-abc" {
				w.WriteHeader(401)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})

		case r.Method == "GET" && r.URL.Path == "/api/v1/cloudron/status":
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.1.5", DisplayName: "Test Cloudron", Domain: "cloud.example.com"})

		case r.Method == "POST" && r.URL.Path == "/api/v1/apps":
			installCalls++
			// Verify multipart
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Fatal("expected multipart/form-data")
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart: %v", err)
			}
			if r.FormValue("subdomain") != "testapp" {
				t.Fatalf("subdomain=%q", r.FormValue("subdomain"))
			}
			manifest := r.FormValue("manifest")
			if manifest == "" {
				t.Fatal("missing manifest")
			}
			if _, _, err := r.FormFile("sourceArchive"); err != nil {
				t.Fatalf("missing sourceArchive: %v", err)
			}
			// Verify auth token from login
			auth := r.Header.Get("Authorization")
			if auth != "Bearer session-tok-abc" {
				t.Fatalf("auth=%q, expected Bearer session-tok-abc", auth)
			}
			w.WriteHeader(202)
			json.NewEncoder(w).Encode(map[string]string{"id": "app-999", "fqdn": "testapp.cloud.example.com"})

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	// Create mock package files
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	os.WriteFile(manifestPath, []byte(`{"id":"io.test.myapp","title":"Test App","version":"1.0.0"}`), 0644)
	tarball := filepath.Join(dir, "package.tar.gz")
	os.WriteFile(tarball, []byte("fake-tarball-content"), 0644)

	// === Run the full v2.0 flow ===
	client := NewClient(srv.URL, "", false)
	client.httpClient = srv.Client()

	// Step 1: Login
	err := client.Login("admin", "secret")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Step 2: Verify connection
	info, err := client.GetCloudronInfo()
	if err != nil {
		t.Fatalf("GetCloudronInfo failed: %v", err)
	}
	if info.Version != "9.1.5" {
		t.Fatalf("version=%q", info.Version)
	}

	// Step 3: Install app with sourceArchive
	appURL, err := client.InstallApp(manifestPath, tarball, "testapp")
	if err != nil {
		t.Fatalf("InstallApp failed: %v", err)
	}
	if appURL != "https://testapp.cloud.example.com" {
		t.Fatalf("appURL=%q", appURL)
	}

	// Verify call counts
	if loginCalls != 1 {
		t.Fatalf("expected 1 login call, got %d", loginCalls)
	}
	if infoCalls != 1 {
		t.Fatalf("expected 1 info call, got %d", infoCalls)
	}
	if installCalls != 1 {
		t.Fatalf("expected 1 install call, got %d", installCalls)
	}
}

// TestFullDeploymentFlow_V2_With2FA tests the login → 2FA → install flow.
func TestFullDeploymentFlow_V2_With2FA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/auth/login":
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
			json.NewEncoder(w).Encode(map[string]string{"accessToken": "2fa-session-tok"})

		case r.Method == "GET" && r.URL.Path == "/api/v1/profile":
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})

		case r.Method == "GET" && r.URL.Path == "/api/v1/cloudron/status":
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.1.5", DisplayName: "Test"})

		case r.Method == "POST" && r.URL.Path == "/api/v1/apps":
			r.ParseMultipartForm(10 << 20)
			w.WriteHeader(202)
			json.NewEncoder(w).Encode(map[string]string{"id": "app-2fa", "fqdn": "app.example.com"})

		default:
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	os.WriteFile(manifestPath, []byte(`{"id":"io.test"}`), 0644)
	tarball := filepath.Join(dir, "pkg.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	client := NewClient(srv.URL, "", false)
	client.httpClient = srv.Client()

	// First login attempt → 2FA required
	err := client.Login("admin", "secret")
	if err != Err2FARequired {
		t.Fatalf("expected Err2FARequired, got %v", err)
	}

	// Retry with TOTP
	err = client.LoginWith2FA("admin", "secret", "123456")
	if err != nil {
		t.Fatal(err)
	}

	// Continue with install
	_, err = client.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.InstallApp(manifestPath, tarball, "app")
	if err != nil {
		t.Fatal(err)
	}
}

// TestFullDeploymentFlow_Update tests the update path.
func TestFullDeploymentFlow_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/auth/login":
			json.NewEncoder(w).Encode(map[string]string{"accessToken": "tok"})
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile":
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})
		case r.Method == "GET" && r.URL.Path == "/api/v1/cloudron/status":
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.1.5", DisplayName: "Test"})
		case r.Method == "GET" && r.URL.Path == "/api/v1/apps":
			json.NewEncoder(w).Encode(map[string]any{
				"apps": []map[string]string{
					{"id": "existing-app", "subdomain": "myapp"},
				},
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/apps/existing-app/update":
			if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				t.Fatal("expected multipart")
			}
			r.ParseMultipartForm(10 << 20)
			if _, _, err := r.FormFile("sourceArchive"); err != nil {
				t.Fatalf("missing sourceArchive: %v", err)
			}
			w.WriteHeader(202)
			json.NewEncoder(w).Encode(map[string]string{"taskId": "task-42"})
		default:
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	os.WriteFile(manifestPath, []byte(`{"id":"io.test","version":"2.0.0"}`), 0644)
	tarball := filepath.Join(dir, "pkg.tar.gz")
	os.WriteFile(tarball, []byte("updated tarball"), 0644)

	client := NewClient(srv.URL, "", false)
	client.httpClient = srv.Client()

	client.Login("admin", "secret")
	client.GetCloudronInfo()

	app, _ := client.FindAppBySubdomain("myapp")
	if app == nil {
		t.Fatal("expected to find existing app")
	}

	_, err := client.UpdateApp(app.ID, manifestPath, tarball)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeploymentFlow_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid credentials"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", false)
	client.httpClient = srv.Client()
	err := client.Login("admin", "wrong")
	if err == nil {
		t.Fatal("expected auth failure")
	}
}
