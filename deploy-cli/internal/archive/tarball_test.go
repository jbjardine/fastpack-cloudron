package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// untar reads a .tar.gz and returns a map of filename → content.
func untar(t *testing.T, tarPath string) map[string]string {
	t.Helper()
	f, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	out := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[hdr.Name] = string(b)
	}
}

func TestCreateTarball_AllFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{"id":"test"}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM nginx"), 0644)
	os.WriteFile(filepath.Join(dir, "start.sh"), []byte("#!/bin/sh"), 0644)
	os.WriteFile(filepath.Join(dir, "nginx.conf"), []byte("server {}"), 0644)
	os.WriteFile(filepath.Join(dir, "DESCRIPTION.md"), []byte("A test app"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	if _, ok := files["CloudronManifest.json"]; !ok {
		t.Fatal("manifest missing from tarball")
	}
	if _, ok := files["Dockerfile"]; !ok {
		t.Fatal("Dockerfile missing from tarball")
	}
	if _, ok := files["start.sh"]; !ok {
		t.Fatal("start.sh missing from tarball")
	}
	if _, ok := files["nginx.conf"]; !ok {
		t.Fatal("nginx.conf missing from tarball")
	}
	if files["DESCRIPTION.md"] != "A test app" {
		t.Fatalf("DESCRIPTION.md content=%q", files["DESCRIPTION.md"])
	}
}

func TestCreateTarball_OptionalFilesMissing(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), keys(files))
	}
}

func TestCreateTarball_NoPackageFiles(t *testing.T) {
	dir := t.TempDir()
	// Only create files not in the allow-list
	os.WriteFile(filepath.Join(dir, "random.txt"), []byte("nothing"), 0644)

	_, err := CreateTarball(dir)
	if err == nil {
		t.Fatal("expected error for empty package")
	}
	if !strings.Contains(err.Error(), "no package files found") {
		t.Fatalf("err=%q", err.Error())
	}
}

func TestCreateTarball_ExcludesSecrets(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM nginx"), 0644)

	// These should NOT be included (not in allow-list)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=bad"), 0644)
	os.WriteFile(filepath.Join(dir, ".env.production"), []byte("SECRET=prod"), 0644)
	os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "private.key"), []byte("key"), 0644)
	os.WriteFile(filepath.Join(dir, "deploy.exe"), []byte("binary"), 0644)
	os.WriteFile(filepath.Join(dir, "random.md"), []byte("notes"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	for name := range files {
		if name == ".env" || name == ".env.production" || name == "credentials.json" ||
			name == "private.key" || name == "deploy.exe" || name == "random.md" {
			t.Fatalf("secret/excluded file %q should not be in tarball", name)
		}
	}
}

func TestCreateTarball_OnlyAllowListIncluded(t *testing.T) {
	dir := t.TempDir()
	// Create all allowed files
	for _, name := range AllowedFiles() {
		os.WriteFile(filepath.Join(dir, name), []byte("content-"+name), 0644)
	}
	// Create non-allowed files
	os.WriteFile(filepath.Join(dir, "secrets.json"), []byte("bad"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0644)
	os.WriteFile(filepath.Join(dir, "NOTES.md"), []byte("not allowed"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	allowed := make(map[string]bool)
	for _, name := range AllowedFiles() {
		allowed[name] = true
	}
	for name := range files {
		if !allowed[name] {
			t.Fatalf("file %q is not in the allow-list but was included", name)
		}
	}
}

func TestCreateTarball_SizeLimit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)

	// Create a Dockerfile that exceeds the limit
	bigContent := strings.Repeat("x", int(MaxTarballSize)+1)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(bigContent), 0644)

	_, err := CreateTarball(dir)
	if err == nil {
		t.Fatal("expected size limit error")
	}
	if !strings.Contains(err.Error(), "size limit") {
		t.Fatalf("err=%q", err.Error())
	}
}

func TestCreateTarball_Cleanup(t *testing.T) {
	dir := t.TempDir()
	// No files → error → temp file should be cleaned up
	_, err := CreateTarball(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	// Verify no orphan temp files (best effort check)
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "fastpack-deploy-*.tar.gz"))
	for _, m := range matches {
		info, _ := os.Stat(m)
		if info != nil && info.Size() == 0 {
			t.Logf("warning: orphan temp file %s", m)
		}
	}
}

func TestAllowedFiles(t *testing.T) {
	files := AllowedFiles()
	if len(files) == 0 {
		t.Fatal("AllowedFiles should return non-empty list")
	}
	// Verify CloudronManifest.json is first (required)
	if files[0] != "CloudronManifest.json" {
		t.Fatalf("first file should be CloudronManifest.json, got %q", files[0])
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
