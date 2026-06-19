// Package wizard provides interactive terminal prompts for deployment configuration.
package wizard

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// ValidSubdomain matches valid DNS subdomain labels (RFC 1123).
var ValidSubdomain = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// Config holds the deployment configuration collected from the user.
type Config struct {
	CloudronURL     string
	Username        string
	Password        string
	Subdomain       string
	AllowSelfSigned bool
	// Token is set externally after login, or via CLOUDRON_TOKEN env var (legacy).
	Token string
}

// FileConfig is the JSON format accepted by fastpack-deploy.json.
type FileConfig struct {
	CloudronURL     string `json:"cloudronUrl"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Token           string `json:"token"`
	Subdomain       string `json:"subdomain"`
	AllowSelfSigned *bool  `json:"allowSelfSigned"`
}

// StdinReader is the buffered reader from the last Run() call.
// Use it for subsequent interactive prompts to avoid losing piped input.
var StdinReader *bufio.Reader

// readPasswordFn reads a password without echoing. Replaced in tests.
var readPasswordFn = func() (string, error) {
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Run executes the interactive wizard using stdin/stdout.
func Run() (*Config, error) {
	StdinReader = bufio.NewReader(os.Stdin)
	return runWithReader(StdinReader, os.Stdout)
}

// RunWithIO executes the interactive wizard with injectable IO for testing.
func RunWithIO(r io.Reader, w io.Writer) (*Config, error) {
	br := bufio.NewReader(r)
	StdinReader = br
	return runWithReader(br, w)
}

func runWithReader(reader *bufio.Reader, w io.Writer) (*Config, error) {
	config := &Config{}
	loadedFrom, allowSelfSignedSet, err := loadFileConfig(config)
	if err != nil {
		return nil, err
	}
	if loadedFrom != "" {
		fmt.Fprintf(w, "   Using deploy config from %s.\n", loadedFrom)
	}

	// Check for legacy CLOUDRON_TOKEN env var (backward compat for scripts).
	// Environment variables stay highest priority for existing automations.
	envToken := os.Getenv("CLOUDRON_TOKEN")
	if envToken != "" {
		config.Token = envToken
		config.Username = ""
		config.Password = ""
	}

	if config.Token != "" {
		return runTokenFlow(reader, w, config, loadedFrom == "" && envToken != "", allowSelfSignedSet)
	}

	// v2.0 flow: URL → Username → Password (only prompts for missing values)
	totalSteps := 3

	// --- Step 1: Cloudron URL ---
	if config.CloudronURL == "" {
		fmt.Fprintf(w, "Step 1/%d: Enter your Cloudron URL\n", totalSteps)
		fmt.Fprintln(w, "   Example: https://my.example.com")
		fmt.Fprint(w, "   URL: ")
		rawURL, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		if err := parseAndSetURL(config, rawURL); err != nil {
			return nil, err
		}
	}

	// --- Step 2: Username ---
	if config.Username == "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Step 2/%d: Enter your Cloudron username\n", totalSteps)
		fmt.Fprint(w, "   Username: ")
		username, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		username = strings.TrimSpace(username)
		if username == "" {
			return nil, fmt.Errorf("username is required")
		}
		config.Username = username
	}

	// --- Step 3: Password ---
	if config.Password == "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Step 3/%d: Enter your Cloudron password\n", totalSteps)
		fmt.Fprint(w, "   Password: ")
		password, err := readPasswordFn()
		if err != nil {
			// Fallback to plain text if terminal not available (piped input)
			password, err = readLine(reader)
			if err != nil {
				return nil, err
			}
		}
		if strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("password is required")
		}
		config.Password = password
	}

	applySelfSignedPolicy(config, allowSelfSignedSet, w)

	fmt.Fprintln(w)
	return config, nil
}

// runTokenFlow handles API-token deployment from the environment or config file.
func runTokenFlow(reader *bufio.Reader, w io.Writer, config *Config, fromEnv bool, allowSelfSignedSet bool) (*Config, error) {
	if fromEnv {
		fmt.Fprintln(w, "   Using API token from CLOUDRON_TOKEN environment variable.")
	}

	totalSteps := 2

	// --- Step 1: Cloudron URL ---
	if config.CloudronURL == "" {
		fmt.Fprintf(w, "\nStep 1/%d: Enter your Cloudron URL\n", totalSteps)
		fmt.Fprintln(w, "   Example: https://my.example.com")
		fmt.Fprint(w, "   URL: ")
		rawURL, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		if err := parseAndSetURL(config, rawURL); err != nil {
			return nil, err
		}
	}

	// --- Step 2: Subdomain ---
	fmt.Fprintln(w)
	domain := extractDomain(config.CloudronURL)
	if config.Subdomain == "" {
		fmt.Fprintf(w, "Step 2/%d: Choose a subdomain for your app\n", totalSteps)
		fmt.Fprintf(w, "   Example: myapp  (your app will be at myapp.%s)\n", domain)
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
	}

	applySelfSignedPolicy(config, allowSelfSignedSet, w)

	fmt.Fprintln(w)
	return config, nil
}

func applySelfSignedPolicy(config *Config, allowSelfSignedSet bool, w io.Writer) {
	if isDevInstance(config.CloudronURL) && !allowSelfSignedSet {
		config.AllowSelfSigned = true
	}
	if config.AllowSelfSigned {
		fmt.Fprintln(w, "\n   ⚠️  Self-signed certificates will be accepted.")
		fmt.Fprintln(w, "   WARNING: TLS verification is disabled. Do not use this in production.")
	}
}

// AskSubdomain prompts for a subdomain if not already provided.
// Uses StdinReader if available (set by Run) to avoid losing buffered data.
func AskSubdomain() (string, error) {
	if StdinReader != nil {
		return askSubdomainWithReader(StdinReader, os.Stdout)
	}
	return AskSubdomainWithIO(os.Stdin, os.Stdout)
}

// AskSubdomainWithIO prompts for a subdomain with injectable IO.
func AskSubdomainWithIO(r io.Reader, w io.Writer) (string, error) {
	return askSubdomainWithReader(bufio.NewReader(r), w)
}

func askSubdomainWithReader(reader *bufio.Reader, w io.Writer) (string, error) {
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

// Ask2FA prompts for a TOTP code when 2FA is required.
// Uses StdinReader if available (set by Run) to avoid losing buffered data.
func Ask2FA() (string, error) {
	if StdinReader != nil {
		return ask2FAWithReader(StdinReader, os.Stdout)
	}
	return Ask2FAWithIO(os.Stdin, os.Stdout)
}

// Ask2FAWithIO prompts for a TOTP code with injectable IO.
func Ask2FAWithIO(r io.Reader, w io.Writer) (string, error) {
	return ask2FAWithReader(bufio.NewReader(r), w)
}

func ask2FAWithReader(reader *bufio.Reader, w io.Writer) (string, error) {
	fmt.Fprintln(w, "\n   🔐 Two-factor authentication required.")
	fmt.Fprint(w, "   Enter your 6-digit code: ")
	code, err := readLine(reader)
	if err != nil {
		return "", err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return "", fmt.Errorf("2FA code is required")
	}
	return code, nil // TOTP codes are numeric, trimming whitespace is safe
}

func loadFileConfig(config *Config) (string, bool, error) {
	path := strings.TrimSpace(os.Getenv("FASTPACK_DEPLOY_CONFIG"))
	if path == "" {
		path = "fastpack-deploy.json"
	}
	cleanPath := filepath.Clean(path)
	b, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("FASTPACK_DEPLOY_CONFIG") == "" {
			return "", false, nil
		}
		return "", false, fmt.Errorf("cannot read deploy config %s: %w", cleanPath, err)
	}
	var fc FileConfig
	if err := json.Unmarshal(b, &fc); err != nil {
		return "", false, fmt.Errorf("invalid deploy config %s: %w", cleanPath, err)
	}
	if fc.CloudronURL != "" {
		if err := parseAndSetURL(config, fc.CloudronURL); err != nil {
			return "", false, fmt.Errorf("invalid cloudronUrl in %s: %w", cleanPath, err)
		}
	}
	config.Username = strings.TrimSpace(fc.Username)
	config.Password = fc.Password
	config.Token = strings.TrimSpace(fc.Token)
	config.Subdomain = strings.TrimSpace(fc.Subdomain)
	if config.Subdomain != "" && !ValidSubdomain.MatchString(config.Subdomain) {
		return "", false, fmt.Errorf("invalid subdomain %q in %s: must contain only lowercase letters, digits, and hyphens", config.Subdomain, cleanPath)
	}
	if fc.AllowSelfSigned != nil {
		config.AllowSelfSigned = *fc.AllowSelfSigned
	}
	return cleanPath, fc.AllowSelfSigned != nil, nil
}

// parseAndSetURL validates and normalizes the Cloudron URL into config.
func parseAndSetURL(config *Config, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	rawURL = strings.TrimRight(rawURL, "/")

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return fmt.Errorf("invalid URL: %s", rawURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL: %s (only http/https supported)", rawURL)
	}
	config.CloudronURL = u.Scheme + "://" + u.Host
	return nil
}

// extractDomain returns the bare domain from a Cloudron URL (strips "my." prefix).
func extractDomain(cloudronURL string) string {
	u, _ := url.Parse(cloudronURL)
	if u == nil {
		return ""
	}
	host := u.Hostname()
	if strings.HasPrefix(host, "my.") {
		return host[3:]
	}
	return host
}

// isDevInstance returns true if the URL looks like a development Cloudron instance.
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
		strings.HasPrefix(host, "10.") ||
		isRFC1918_172(host)
}

// isRFC1918_172 checks if host is in the 172.16.0.0/12 private range.
func isRFC1918_172(host string) bool {
	if !strings.HasPrefix(host, "172.") {
		return false
	}
	// Extract second octet
	rest := host[4:]
	dot := strings.IndexByte(rest, '.')
	if dot < 1 || dot > 2 {
		return false
	}
	octet := 0
	for _, c := range rest[:dot] {
		if c < '0' || c > '9' {
			return false
		}
		octet = octet*10 + int(c-'0')
	}
	return octet >= 16 && octet <= 31
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err == io.EOF && len(line) > 0 {
		return strings.TrimRight(line, "\r\n"), nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
