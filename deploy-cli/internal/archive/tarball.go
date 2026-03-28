// Package archive creates tarballs from FastPackCloudron package directories.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxTarballSize is the maximum uncompressed tarball size (100 MB).
const MaxTarballSize int64 = 100 * 1024 * 1024

// allowedFiles is the strict allow-list of files to include in the tarball.
// No wildcard inclusion — only explicitly listed files are packaged.
var allowedFiles = []string{
	// Required
	"CloudronManifest.json",
	"Dockerfile",
	// Optional Cloudron files
	"Dockerfile.cloudron",
	"start.sh",
	".dockerignore",
	"nginx.conf",
	"icon.png",
	// Documentation (explicit, no wildcards)
	"DESCRIPTION.md",
	"CHANGELOG.md",
	"README.md",
	// Extra config
	"cloudron-versions.json",
}

// CreateTarball creates a .tar.gz of the allowed package files in dir.
// Returns the path to the temporary tarball file.
func CreateTarball(dir string) (string, error) {
	tmpFile, err := os.CreateTemp("", "fastpack-deploy-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// cleanup closes writers and removes the temp file on error.
	cleanup := func(e error) (string, error) {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", e
	}

	gw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gw)

	var totalSize int64
	filesAdded := 0

	for _, name := range allowedFiles {
		fullPath := filepath.Join(dir, name)
		info, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			continue // Optional files are silently skipped
		}
		if err != nil {
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("cannot stat %s: %w", name, err))
		}

		// Enforce size limit
		totalSize += info.Size()
		if totalSize > MaxTarballSize {
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("package exceeds %d MB size limit", MaxTarballSize/(1024*1024)))
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("tar header error for %s: %w", name, err))
		}
		header.Name = name

		if err := tw.WriteHeader(header); err != nil {
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("write header %s: %w", name, err))
		}

		f, err := os.Open(fullPath)
		if err != nil {
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("open %s: %w", name, err))
		}
		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			tw.Close()
			gw.Close()
			return cleanup(fmt.Errorf("copy %s: %w", name, err))
		}
		f.Close()
		filesAdded++
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		return cleanup(fmt.Errorf("tar close: %w", err))
	}
	if err := gw.Close(); err != nil {
		return cleanup(fmt.Errorf("gzip close: %w", err))
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("file close: %w", err)
	}

	if filesAdded == 0 {
		os.Remove(tmpPath)
		return "", fmt.Errorf("no package files found in %s", dir)
	}

	return tmpPath, nil
}

// AllowedFiles returns the list of files that will be included in the tarball.
func AllowedFiles() []string {
	out := make([]string, len(allowedFiles))
	copy(out, allowedFiles)
	return out
}
