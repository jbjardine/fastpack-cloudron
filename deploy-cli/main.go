// FastPack Deploy CLI — zero-dependency Cloudron deployer
// Builds Docker images on the user's Cloudron Build Service and installs via API.
// Cross-platform: Windows, macOS, Linux. No Node.js, no cloudron CLI, no Docker needed.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/api"
	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/archive"
	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/wizard"
)

const version = "1.0.0"

func main() {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   FastPack Deploy — Cloudron Deployer ║")
	fmt.Printf("║   v%s                              ║\n", version)
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Detect package files in current directory
	dir, err := os.Getwd()
	if err != nil {
		fatal("Cannot determine current directory: %v", err)
	}

	manifest := filepath.Join(dir, "CloudronManifest.json")
	if _, err := os.Stat(manifest); os.IsNotExist(err) {
		fatal("CloudronManifest.json not found in %s\nMake sure you run this from the extracted ZIP folder.", dir)
	}

	fmt.Printf("📦 Package found in %s\n\n", dir)

	// Step 2: Interactive wizard — ask for Cloudron URL + API token
	config, err := wizard.Run()
	if err != nil {
		fatal("Setup cancelled: %v", err)
	}

	// Step 3: Verify connection to Cloudron
	fmt.Print("🔗 Connecting to Cloudron... ")
	client := api.NewClient(config.CloudronURL, config.Token, config.AllowSelfSigned)
	if config.BuildServiceURL != "" {
		client.SetBuildService(config.BuildServiceURL, config.BuildToken)
	}
	info, err := client.GetCloudronInfo()
	if err != nil {
		fatal("\n❌ Cannot connect: %v\nCheck your URL and API token.", err)
	}
	fmt.Printf("OK (%s v%s)\n", info.DisplayName, info.Version)

	// Step 4: Create tarball from package files
	fmt.Print("📦 Packaging files... ")
	tarball, err := archive.CreateTarball(dir)
	if err != nil {
		fatal("\n❌ Packaging failed: %v", err)
	}
	defer os.Remove(tarball)
	fmt.Println("OK")

	// Step 5: Upload to Cloudron Build Service and build
	fmt.Print("🔨 Building on Cloudron (this may take a few minutes)... ")
	imageTag, err := client.BuildImage(tarball)
	if err != nil {
		fatal("\n❌ Build failed: %v", err)
	}
	fmt.Printf("OK (image: %s)\n", imageTag)

	// Step 6: Install the app
	subdomain := config.Subdomain
	if subdomain == "" {
		subdomain, err = wizard.AskSubdomain()
		if err != nil {
			fatal("Setup cancelled: %v", err)
		}
	}

	fmt.Printf("🚀 Installing app at %s.%s... ", subdomain, info.Domain)
	appURL, err := client.InstallApp(manifest, imageTag, subdomain)
	if err != nil {
		fatal("\n❌ Install failed: %v", err)
	}
	fmt.Println("OK")

	// Step 7: Success!
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║          ✅ App deployed!             ║")
	fmt.Printf("║  %s\n", appURL)
	fmt.Println("╚══════════════════════════════════════╝")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\n"+format+"\n", args...)
	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
	os.Exit(1)
}
