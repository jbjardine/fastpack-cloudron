// Package wizard provides interactive terminal prompts for deployment configuration.
// Uses only stdlib — no external dependencies.
package wizard

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Config holds the deployment configuration collected from the user.
type Config struct {
	CloudronURL     string
	Token           string
	Subdomain       string
	AllowSelfSigned bool
}

// Run executes the interactive wizard and returns the deployment config.
func Run() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	config := &Config{}

	// Cloudron URL
	fmt.Println("Step 1/3: Enter your Cloudron URL")
	fmt.Println("   Example: https://my.example.com")
	fmt.Print("   URL: ")
	rawURL, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	// Normalize: add https:// if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	// Remove trailing slash
	rawURL = strings.TrimRight(rawURL, "/")

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid URL: %s", rawURL)
	}
	config.CloudronURL = rawURL

	// API Token
	fmt.Println()
	fmt.Println("Step 2/3: Enter your API token")
	fmt.Println("   Get it from: Cloudron Dashboard > Profile > API Access > Create Token")
	fmt.Print("   Token: ")
	token, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("API token is required")
	}
	config.Token = token

	// Subdomain
	fmt.Println()
	fmt.Println("Step 3/3: Choose a subdomain for your app")
	fmt.Println("   Example: myapp  (your app will be at myapp.example.com)")
	fmt.Print("   Subdomain: ")
	subdomain, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	config.Subdomain = strings.TrimSpace(subdomain)

	// Self-signed cert detection
	if strings.Contains(rawURL, ".nip.io") || strings.Contains(rawURL, "localhost") || strings.Contains(rawURL, "192.168.") || strings.Contains(rawURL, "10.") {
		config.AllowSelfSigned = true
		fmt.Println("\n   ⚠️  Dev instance detected — self-signed certificates will be accepted.")
	}

	fmt.Println()
	return config, nil
}

// AskSubdomain prompts for a subdomain if not already provided.
func AskSubdomain() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("   Subdomain for your app: ")
	sub, err := readLine(reader)
	if err != nil {
		return "", err
	}
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return "", fmt.Errorf("subdomain is required")
	}
	return sub, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
