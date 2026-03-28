// Package wizard provides interactive terminal prompts for deployment configuration.
// Uses only stdlib — no external dependencies.
package wizard

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// ValidSubdomain matches valid DNS subdomain labels (RFC 1123).
var ValidSubdomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// Config holds the deployment configuration collected from the user.
type Config struct {
	CloudronURL     string
	Token           string
	Subdomain       string
	AllowSelfSigned bool
}

// Run executes the interactive wizard using stdin/stdout.
func Run() (*Config, error) {
	return RunWithIO(os.Stdin, os.Stdout)
}

// RunWithIO executes the interactive wizard with injectable IO for testing.
func RunWithIO(r io.Reader, w io.Writer) (*Config, error) {
	reader := bufio.NewReader(r)
	config := &Config{}

	// Check CLOUDRON_TOKEN env var first
	envToken := os.Getenv("CLOUDRON_TOKEN")

	// Cloudron URL
	fmt.Fprintln(w, "Step 1/3: Enter your Cloudron URL")
	fmt.Fprintln(w, "   Example: https://my.example.com")
	fmt.Fprint(w, "   URL: ")
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
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid URL: %s", rawURL)
	}
	// Reject non-HTTP schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL: %s (only http/https supported)", rawURL)
	}
	config.CloudronURL = u.Scheme + "://" + u.Host

	// API Token — use env var if set, otherwise prompt
	if envToken != "" {
		fmt.Fprintln(w, "\n   Using API token from CLOUDRON_TOKEN environment variable.")
		config.Token = envToken
	} else {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Step 2/3: Enter your API token")
		fmt.Fprintln(w, "   Get it from: Cloudron Dashboard > Profile > API Access > Create Token")
		fmt.Fprint(w, "   Token: ")
		token, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		token = strings.TrimSpace(token)
		if token == "" {
			return nil, fmt.Errorf("API token is required")
		}
		config.Token = token
	}

	// Subdomain
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Step 3/3: Choose a subdomain for your app")
	fmt.Fprintln(w, "   Example: myapp  (your app will be at myapp.example.com)")
	fmt.Fprint(w, "   Subdomain: ")
	subdomain, err := readLine(reader)
	if err != nil {
		return nil, err
	}
	subdomain = strings.TrimSpace(subdomain)
	if subdomain != "" && !ValidSubdomain.MatchString(subdomain) {
		return nil, fmt.Errorf("invalid subdomain %q: must contain only lowercase letters, digits, and hyphens", subdomain)
	}
	config.Subdomain = subdomain

	// Self-signed cert detection — warn explicitly
	if isDevInstance(rawURL) {
		config.AllowSelfSigned = true
		fmt.Fprintln(w, "\n   ⚠️  Dev instance detected — self-signed certificates will be accepted.")
		fmt.Fprintln(w, "   WARNING: TLS verification is disabled. Do not use this in production.")
	}

	fmt.Fprintln(w)
	return config, nil
}

// AskSubdomain prompts for a subdomain if not already provided.
func AskSubdomain() (string, error) {
	return AskSubdomainWithIO(os.Stdin, os.Stdout)
}

// AskSubdomainWithIO prompts for a subdomain with injectable IO.
func AskSubdomainWithIO(r io.Reader, w io.Writer) (string, error) {
	reader := bufio.NewReader(r)
	fmt.Fprint(w, "   Subdomain for your app: ")
	sub, err := readLine(reader)
	if err != nil {
		return "", err
	}
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return "", fmt.Errorf("subdomain is required")
	}
	if !ValidSubdomain.MatchString(sub) {
		return "", fmt.Errorf("invalid subdomain %q: must contain only lowercase letters, digits, and hyphens", sub)
	}
	return sub, nil
}

// isDevInstance returns true if the URL looks like a development Cloudron instance.
// Checks are performed on the parsed hostname to avoid false positives
// (e.g., "api.v10.example.com" should NOT trigger TLS bypass).
func isDevInstance(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return strings.HasSuffix(host, ".nip.io") ||
		host == "nip.io" ||
		host == "localhost" ||
		strings.HasPrefix(host, "192.168.") ||
		strings.HasPrefix(host, "10.")
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	// Accept partial data on EOF (piped input without trailing newline)
	if err == io.EOF && len(line) > 0 {
		return strings.TrimRight(line, "\r\n"), nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
