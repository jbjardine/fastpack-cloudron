// Package archive creates tarballs from FastPackCloudron package directories.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Files to include in the tarball (the Cloudron package files).
var packageFiles = []string{
	"CloudronManifest.json",
	"Dockerfile",
	"Dockerfile.cloudron",
	"start.sh",
	".dockerignore",
	"nginx.conf",
	"icon.png",
}

// CreateTarball creates a .tar.gz of the package files in dir.
// Returns the path to the temporary tarball file.
func CreateTarball(dir string) (string, error) {
	tmpFile, err := os.CreateTemp("", "fastpack-deploy-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	gw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gw)

	filesAdded := 0
	for _, name := range packageFiles {
		fullPath := filepath.Join(dir, name)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			continue // Optional files (nginx.conf, icon.png, Dockerfile.cloudron)
		}
		if err != nil {
			tw.Close()
			gw.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("cannot stat %s: %w", name, err)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			tw.Close()
			gw.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", fmt.Errorf("tar header error for %s: %w", name, err)
		}
		// Use just the filename, not the full path
		header.Name = name

		if err := tw.WriteHeader(header); err != nil {
			tw.Close()
			gw.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", err
		}

		f, err := os.Open(fullPath)
		if err != nil {
			tw.Close()
			gw.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", err
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			tw.Close()
			gw.Close()
			tmpFile.Close()
			os.Remove(tmpPath)
			return "", err
		}
		f.Close()
		filesAdded++
	}

	// Also include any other files that might be needed (DESCRIPTION.md, etc.)
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip already-added files, the deploy binary itself, and non-package files
		if contains(packageFiles, name) {
			continue
		}
		if strings.HasSuffix(name, ".exe") || name == "deploy" || name == "deploy.cmd" || name == "deploy.js" {
			continue
		}
		if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json") {
			// Include markdown and JSON files (DESCRIPTION.md, CHANGELOG.md, etc.)
			fullPath := filepath.Join(dir, name)
			info, _ := entry.Info()
			header, _ := tar.FileInfoHeader(info, "")
			header.Name = name
			tw.WriteHeader(header)
			f, _ := os.Open(fullPath)
			io.Copy(tw, f)
			f.Close()
			filesAdded++
		}
	}

	tw.Close()
	gw.Close()
	tmpFile.Close()

	if filesAdded == 0 {
		os.Remove(tmpPath)
		return "", fmt.Errorf("no package files found in %s", dir)
	}

	return tmpPath, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
