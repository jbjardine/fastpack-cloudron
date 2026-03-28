package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestFullDeploymentFlow simulates the complete CLI → Cloudron API workflow.
func TestFullDeploymentFlow(t *testing.T) {
	var cloudronStep, buildStep int

	// Mock Cloudron API (profile, status, install)
	cloudronSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/profile":
			cloudronStep++
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token-xyz" {
				w.WriteHeader(401)
				return
			}
			json.NewEncoder(w).Encode(map[string]string{"username": "admin"})

		case r.Method == "GET" && r.URL.Path == "/api/v1/cloudron/status":
			cloudronStep++
			json.NewEncoder(w).Encode(CloudronInfo{Version: "9.1.5", DisplayName: "Test Cloudron"})

		case r.Method == "POST" && r.URL.Path == "/api/v1/apps":
			cloudronStep++
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["location"] != "testapp" {
				t.Fatalf("wrong subdomain: %v", payload["location"])
			}
			if payload["image"] != "registry.test/app:build-42" {
				t.Fatalf("wrong image: %v", payload["image"])
			}
			json.NewEncoder(w).Encode(map[string]string{"id": "app-999", "fqdn": "testapp.cloud.example.com"})

		default:
			t.Fatalf("unexpected Cloudron request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer cloudronSrv.Close()

	// Mock Build Service
	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/api/v1/builds" {
			buildStep++
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("expected multipart: %v", err)
			}
			json.NewEncoder(w).Encode(map[string]string{"image": "registry.test/app:build-42"})
		} else {
			t.Fatalf("unexpected Build Service request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer buildSrv.Close()

	// Create mock package directory
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	os.WriteFile(manifestPath, []byte(`{"id":"io.test.myapp","title":"Test App","version":"1.0.0"}`), 0644)

	tarball := filepath.Join(dir, "package.tar.gz")
	os.WriteFile(tarball, []byte("fake-tarball"), 0644)

	// === Run the full flow ===
	client := NewClient(cloudronSrv.URL, "test-token-xyz", false)
	client.SetBuildService(buildSrv.URL, "build-token")

	// Step 1: Verify connection
	info, err := client.GetCloudronInfo()
	if err != nil {
		t.Fatalf("GetCloudronInfo failed: %v", err)
	}
	if info.Version != "9.1.5" {
		t.Fatalf("version=%q", info.Version)
	}

	// Step 2: Build image
	imageTag, err := client.BuildImage(tarball)
	if err != nil {
		t.Fatalf("BuildImage failed: %v", err)
	}
	if imageTag != "registry.test/app:build-42" {
		t.Fatalf("imageTag=%q", imageTag)
	}

	// Step 3: Install app
	appURL, err := client.InstallApp(manifestPath, imageTag, "testapp")
	if err != nil {
		t.Fatalf("InstallApp failed: %v", err)
	}
	if appURL != "https://testapp.cloud.example.com" {
		t.Fatalf("appURL=%q", appURL)
	}

	// Verify all steps executed
	if cloudronStep != 3 {
		t.Fatalf("expected 3 Cloudron API calls, got %d", cloudronStep)
	}
	if buildStep != 1 {
		t.Fatalf("expected 1 Build Service call, got %d", buildStep)
	}
}

func TestDeploymentFlow_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad-token", false)
	_, err := client.GetCloudronInfo()
	if err == nil {
		t.Fatal("expected auth failure")
	}
}

func TestDeploymentFlow_BuildFailure(t *testing.T) {
	cloudronSrv := cloudronMock(t)
	defer cloudronSrv.Close()

	buildSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer buildSrv.Close()

	dir := t.TempDir()
	tarball := filepath.Join(dir, "pkg.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	client := NewClient(cloudronSrv.URL, "tok", false)
	client.httpClient = cloudronSrv.Client()
	client.SetBuildService(buildSrv.URL, "tok")

	// Step 1 should succeed
	_, err := client.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}

	// Step 2 should fail
	// Need to use build service's http client
	client.httpClient = buildSrv.Client()
	_, err = client.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected build failure")
	}
}
