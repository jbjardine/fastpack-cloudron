// FastPack Deploy CLI — zero-dependency Cloudron deployer
// Deploys apps directly to Cloudron via sourceArchive upload. No Build Service needed.
// Cross-platform: Windows, macOS, Linux. No Node.js, no cloudron CLI, no Docker needed.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/api"
	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/archive"
	"github.com/jbjardine/fastpack-cloudron/deploy-cli/internal/wizard"
)

var version = "2.1.3"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		fmt.Println("\nPress Enter to exit...")
		fmt.Scanln()
		os.Exit(1)
	}
	fmt.Println("\nPress Enter to exit...")
	wizard.StdinReader.ReadString('\n')
}

func run() error {
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║   FastPack Deploy — Cloudron Deployer ║")
	fmt.Printf("║   v%s                              ║\n", version)
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Detect package files in current directory
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine current directory: %w", err)
	}

	manifestPath := filepath.Join(dir, "CloudronManifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("CloudronManifest.json not found in %s\nMake sure you run this from the extracted ZIP folder", dir)
	}

	fmt.Printf("📦 Package found in %s\n\n", dir)

	// Step 2: Interactive wizard — ask for Cloudron URL + credentials
	config, err := wizard.Run()
	if err != nil {
		return fmt.Errorf("setup cancelled: %w", err)
	}

	// Step 3: Authenticate
	client := api.NewClient(config.CloudronURL, config.Token, config.AllowSelfSigned)

	if config.Token == "" {
		// v2.0 flow: login with username/password
		fmt.Print("🔑 Logging in... ")
		err = client.Login(config.Username, config.Password)
		if errors.Is(err, api.Err2FARequired) {
			fmt.Println("2FA required")
			code, err2 := wizard.Ask2FA()
			if err2 != nil {
				return fmt.Errorf("setup cancelled: %w", err2)
			}
			fmt.Print("🔑 Verifying 2FA... ")
			err = client.LoginWith2FA(config.Username, config.Password, code)
		}
		if err != nil {
			return fmt.Errorf("❌ Login failed: %w\nCheck your username and password", err)
		}
		fmt.Println("OK")
	}

	// Step 4: Verify connection to Cloudron
	fmt.Print("🔗 Connecting to Cloudron... ")
	info, err := client.GetCloudronInfo()
	if err != nil {
		return fmt.Errorf("❌ Cannot connect: %w", err)
	}
	fmt.Printf("OK (%s v%s)\n", info.DisplayName, info.Version)

	// Step 5: Create tarball from package files
	fmt.Print("📦 Packaging files... ")
	tarball, err := archive.CreateTarball(dir)
	if err != nil {
		return fmt.Errorf("❌ Packaging failed: %w", err)
	}
	defer os.Remove(tarball)
	fmt.Println("OK")

	// Step 6: Choose subdomain
	subdomain := config.Subdomain
	if subdomain == "" {
		subdomain, err = wizard.AskSubdomain()
		if err != nil {
			return fmt.Errorf("setup cancelled: %w", err)
		}
	}

	// Step 7: Check if app already exists
	existing, _ := client.FindAppBySubdomain(subdomain)

	if existing != nil {
		// App exists — ask to update
		fmt.Printf("\n⚠️  App already installed at %s.%s\n", subdomain, info.Domain)
		fmt.Print("   Update existing app? (y/n): ")
		answer, _ := wizard.StdinReader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			return fmt.Errorf("cancelled — choose a different subdomain or update manually")
		}

		fmt.Printf("🔄 Updating app %s.%s... ", subdomain, info.Domain)
		if _, err = client.UpdateApp(existing.ID, manifestPath, tarball); err != nil {
			return fmt.Errorf("❌ Update failed: %w", err)
		}
		fmt.Println("OK")
	} else {
		// New install
		fmt.Printf("🚀 Installing app at %s.%s (this may take a few minutes)... ", subdomain, info.Domain)
		if _, err = client.InstallApp(manifestPath, tarball, subdomain); err != nil {
			return fmt.Errorf("❌ Install failed: %w", err)
		}
		fmt.Println("OK")
	}

	// Step 8: Success!
	action := "deployed"
	if existing != nil {
		action = "updated"
	}
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Printf("║          ✅ App %s!            ║\n", action)
	fmt.Printf("║  https://%s.%s\n", subdomain, info.Domain)
	fmt.Println("╚══════════════════════════════════════╝")
	return nil
}
