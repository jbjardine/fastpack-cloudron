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

// === MUTATION-KILLING TESTS ===

func TestCreateTarball_SizeLimitBoundary(t *testing.T) {
	// Mutation target: MaxTarballSize changed to math.MaxInt64
	// Verify the constant is actually 100MB
	if MaxTarballSize != 100*1024*1024 {
		t.Fatalf("MaxTarballSize=%d, want %d (100MB)", MaxTarballSize, 100*1024*1024)
	}
}

func TestCreateTarball_FileContentPreserved(t *testing.T) {
	// Mutation target: io.Copy replaced with no-op → files have zero content
	dir := t.TempDir()
	manifest := `{"id":"io.test.app","title":"Test","version":"1.0.0"}`
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(manifest), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM nginx:latest"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	if files["CloudronManifest.json"] != manifest {
		t.Fatalf("manifest content not preserved: got %q", files["CloudronManifest.json"])
	}
	if files["Dockerfile"] != "FROM nginx:latest" {
		t.Fatalf("Dockerfile content not preserved: got %q", files["Dockerfile"])
	}
}

func TestCreateTarball_FileNamePreserved(t *testing.T) {
	// Mutation target: header.Name changed to wrong value
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM a"), 0644)
	os.WriteFile(filepath.Join(dir, "start.sh"), []byte("#!/bin/sh"), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	expectedNames := []string{"CloudronManifest.json", "Dockerfile", "start.sh"}
	for _, name := range expectedNames {
		if _, ok := files[name]; !ok {
			t.Fatalf("expected file %q in tarball, got files: %v", name, keys(files))
		}
	}
}

func TestCreateTarball_ExactAllowListEnforcement(t *testing.T) {
	// Mutation target: allow-list expanded or made into wildcard
	// Create every common dangerous file type and verify NONE are included
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM x"), 0644)

	dangerous := []string{
		".env", ".env.local", ".env.production",
		".gitignore", ".git", "id_rsa", "id_rsa.pub",
		"server.key", "server.pem", "server.crt",
		"package.json", "package-lock.json",
		"node_modules", "secrets.json", "config.json",
		"NOTES.md", "TODO.md", "SECURITY.md",
		"deploy.js", "deploy.cmd", "deploy.exe",
		"random.txt", "data.csv",
	}
	for _, name := range dangerous {
		os.WriteFile(filepath.Join(dir, name), []byte("sensitive"), 0644)
	}

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	for name := range files {
		if name != "CloudronManifest.json" && name != "Dockerfile" {
			t.Fatalf("unexpected file %q in tarball — only allowed files should be included", name)
		}
	}
}

func TestCreateTarball_NonZeroTarball(t *testing.T) {
	// Mutation target: gzip/tar writers not properly flushed
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	info, err := os.Stat(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("tarball file should not be empty")
	}
	if info.Size() < 20 {
		t.Fatalf("tarball suspiciously small: %d bytes", info.Size())
	}
}

func TestCreateTarball_CloudronVersionsCase(t *testing.T) {
	// Bug found by Codex: frontend generates "CloudronVersions.json" (PascalCase)
	// The allow-list must match exactly.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CloudronManifest.json"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM x"), 0644)
	os.WriteFile(filepath.Join(dir, "CloudronVersions.json"), []byte(`{"1.0.0":{}}`), 0644)
	// Wrong case should NOT be included
	os.WriteFile(filepath.Join(dir, "cloudron-versions.json"), []byte(`{"wrong":true}`), 0644)

	tarPath, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tarPath)

	files := untar(t, tarPath)
	if _, ok := files["CloudronVersions.json"]; !ok {
		t.Fatal("CloudronVersions.json (PascalCase) should be in tarball")
	}
	if _, ok := files["cloudron-versions.json"]; ok {
		t.Fatal("cloudron-versions.json (kebab-case) should NOT be in tarball")
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
	// Verify CloudronVersions.json uses correct case (matches frontend)
	found := false
	for _, f := range files {
		if f == "CloudronVersions.json" {
			found = true
		}
		if f == "cloudron-versions.json" {
			t.Fatal("allow-list should use CloudronVersions.json, not cloudron-versions.json")
		}
	}
	if !found {
		t.Fatal("CloudronVersions.json not found in allow-list")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
