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
	BuildServiceURL string
	BuildToken      string
}

// StdinReader is the buffered reader from the last Run() call.
// Use it for subsequent interactive prompts to avoid losing piped input.
var StdinReader *bufio.Reader

// Run executes the interactive wizard using stdin/stdout.
func Run() (*Config, error) {
	StdinReader = bufio.NewReader(os.Stdin)
	return runWithReader(StdinReader, os.Stdout)
}

// RunWithIO executes the interactive wizard with injectable IO for testing.
// Step numbering is dynamic — skipped steps (via env vars) don't leave gaps.
func RunWithIO(r io.Reader, w io.Writer) (*Config, error) {
	return runWithReader(bufio.NewReader(r), w)
}

func runWithReader(reader *bufio.Reader, w io.Writer) (*Config, error) {
	config := &Config{}

	// Check env vars to determine which steps to show
	envToken := os.Getenv("CLOUDRON_TOKEN")
	envBuildURL := os.Getenv("CLOUDRON_BUILD_SERVICE_URL")
	envBuildToken := os.Getenv("CLOUDRON_BUILD_TOKEN")

	// Count total interactive steps
	totalSteps := 4 // URL + Token + Subdomain + Build Service
	if envToken != "" {
		totalSteps--
	}
	if envBuildURL != "" {
		totalSteps--
	}
	step := 0

	// --- Cloudron URL ---
	step++
	fmt.Fprintf(w, "Step %d/%d: Enter your Cloudron URL\n", step, totalSteps)
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
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	rawURL = strings.TrimRight(rawURL, "/")

	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid URL: %s", rawURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid URL: %s (only http/https supported)", rawURL)
	}
	config.CloudronURL = u.Scheme + "://" + u.Host

	// --- API Token ---
	if envToken != "" {
		fmt.Fprintln(w, "\n   Using API token from CLOUDRON_TOKEN environment variable.")
		config.Token = envToken
	} else {
		step++
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Step %d/%d: Enter your API token\n", step, totalSteps)
		fmt.Fprintf(w, "   Create one at: %s/#/settings (Profile > API Access)\n", config.CloudronURL)
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

	// --- Subdomain ---
	step++
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Step %d/%d: Choose a subdomain for your app\n", step, totalSteps)
	domain := u.Hostname()
	if strings.HasPrefix(domain, "my.") {
		domain = domain[3:]
	}
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

	// --- Build Service ---
	if envBuildURL != "" {
		if envBuildToken == "" {
			return nil, fmt.Errorf("CLOUDRON_BUILD_SERVICE_URL is set but CLOUDRON_BUILD_TOKEN is missing.\n   Set CLOUDRON_BUILD_TOKEN to your Build Service token")
		}
		config.BuildServiceURL = envBuildURL
		config.BuildToken = envBuildToken
		fmt.Fprintln(w, "\n   Using Build Service from environment variables.")
	} else {
		step++
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Step %d/%d: Build Service (builds Docker images on your Cloudron)\n", step, totalSteps)
		fmt.Fprintln(w, "   This is the Cloudron app that builds Docker images server-side.")
		fmt.Fprintln(w, "   You need the Docker Builder app installed on your Cloudron.")
		fmt.Fprint(w, "   Build Service URL (e.g., devtools.example.com): ")
		buildURL, err := readLine(reader)
		if err != nil && err != io.EOF {
			return nil, err
		}
		buildURL = strings.TrimSpace(buildURL)
		if buildURL != "" {
			if !strings.HasPrefix(buildURL, "http://") && !strings.HasPrefix(buildURL, "https://") {
				buildURL = "https://" + buildURL
			}
			buildURL = strings.TrimRight(buildURL, "/")
			config.BuildServiceURL = buildURL

			fmt.Fprintln(w)
			fmt.Fprintf(w, "   To get your Build Service token:\n")
			fmt.Fprintf(w, "   1. Open %s in your browser\n", buildURL)
			fmt.Fprintf(w, "   2. Log in with your Cloudron account\n")
			fmt.Fprintf(w, "   3. Copy the token from the Setup page\n")
			fmt.Fprint(w, "   Build Service Token: ")
			bt, _ := readLine(reader)
			bt = strings.TrimSpace(bt)
			if bt == "" {
				return nil, fmt.Errorf("Build Service token is required.\n   Get it from: %s (log in → Setup page)", buildURL)
			}
			config.BuildToken = bt
		}
	}

	// Self-signed cert detection
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
