package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestFullDeploymentFlow simulates the complete CLI → Cloudron API workflow
// by running all 3 API calls in sequence against a single mock server.
// This is the integration-level test in the test pyramid.
func TestFullDeploymentFlow(t *testing.T) {
	var step int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Step 1: Verify connectivity
		case r.Method == "GET" && r.URL.Path == "/api/v1/config":
			step++
			if step != 1 {
				t.Fatalf("GetCloudronInfo called at step %d, expected 1", step)
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token-xyz" {
				w.WriteHeader(401)
				return
			}
			json.NewEncoder(w).Encode(CloudronInfo{
				Version:     "8.2.0",
				DisplayName: "Test Cloudron",
				Domain:      "cloud.example.com",
			})

		// Step 2: Build image
		case r.Method == "POST" && r.URL.Path == "/api/v1/developer/build":
			step++
			if step != 2 {
				t.Fatalf("BuildImage called at step %d, expected 2", step)
			}
			// Verify multipart upload
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("expected multipart form: %v", err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("missing file in upload: %v", err)
			}
			defer file.Close()
			if header.Size == 0 {
				t.Fatal("uploaded file is empty")
			}

			json.NewEncoder(w).Encode(map[string]string{
				"image": "registry.cloud.example.com/app-abc123:build-42",
			})

		// Step 3: Install app
		case r.Method == "POST" && r.URL.Path == "/api/v1/apps":
			step++
			if step != 3 {
				t.Fatalf("InstallApp called at step %d, expected 3", step)
			}
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)

			if payload["location"] != "testapp" {
				t.Fatalf("wrong subdomain: %v", payload["location"])
			}
			if payload["image"] != "registry.cloud.example.com/app-abc123:build-42" {
				t.Fatalf("wrong image: %v", payload["image"])
			}
			manifest, ok := payload["manifest"].(map[string]any)
			if !ok || manifest["id"] != "io.test.myapp" {
				t.Fatalf("wrong manifest: %v", payload["manifest"])
			}

			json.NewEncoder(w).Encode(map[string]string{
				"id":   "app-999",
				"fqdn": "testapp.cloud.example.com",
			})

		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	// Create mock package directory with manifest and tarball-ready files
	dir := t.TempDir()
	manifestContent := `{"id":"io.test.myapp","title":"Test App","version":"1.0.0","manifestVersion":2}`
	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	os.WriteFile(manifestPath, []byte(manifestContent), 0644)

	// Create a fake tarball for upload
	tarball := filepath.Join(dir, "package.tar.gz")
	os.WriteFile(tarball, []byte("fake-tarball-content-for-test"), 0644)

	// === Run the full flow ===
	client := NewClient(srv.URL, "test-token-xyz", false)

	// Step 1: Verify connection
	info, err := client.GetCloudronInfo()
	if err != nil {
		t.Fatalf("Step 1 (GetCloudronInfo) failed: %v", err)
	}
	if info.Domain != "cloud.example.com" {
		t.Fatalf("domain=%q", info.Domain)
	}

	// Step 2: Build image
	imageTag, err := client.BuildImage(tarball)
	if err != nil {
		t.Fatalf("Step 2 (BuildImage) failed: %v", err)
	}
	if imageTag != "registry.cloud.example.com/app-abc123:build-42" {
		t.Fatalf("imageTag=%q", imageTag)
	}

	// Step 3: Install app
	appURL, err := client.InstallApp(manifestPath, imageTag, "testapp")
	if err != nil {
		t.Fatalf("Step 3 (InstallApp) failed: %v", err)
	}
	if appURL != "https://testapp.cloud.example.com" {
		t.Fatalf("appURL=%q", appURL)
	}

	// Verify all 3 steps were executed in order
	if step != 3 {
		t.Fatalf("expected 3 API calls, got %d", step)
	}
}

// TestDeploymentFlow_AuthFailure verifies the flow stops at step 1 on bad token.
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
	// Flow should stop here — BuildImage and InstallApp should NOT be called
}

// TestDeploymentFlow_BuildFailure verifies correct behavior when build fails.
func TestDeploymentFlow_BuildFailure(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/config":
			json.NewEncoder(w).Encode(CloudronInfo{Version: "8.0", DisplayName: "T", Domain: "t.com"})
		case "/api/v1/developer/build":
			w.WriteHeader(500) // Build failure
		default:
			t.Fatalf("unexpected call to %s after build failure", r.URL.Path)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	tarball := filepath.Join(dir, "pkg.tar.gz")
	os.WriteFile(tarball, []byte("data"), 0644)

	client := NewClient(srv.URL, "tok", false)

	// Step 1 should succeed
	_, err := client.GetCloudronInfo()
	if err != nil {
		t.Fatal(err)
	}

	// Step 2 should fail
	_, err = client.BuildImage(tarball)
	if err == nil {
		t.Fatal("expected build failure")
	}

	// Verify exactly 2 calls were made (no step 3)
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}
