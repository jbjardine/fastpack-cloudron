package wizard

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestRunWithIO_FullFlow(t *testing.T) {
	in := strings.NewReader("my.example.com\nsecret-token\nmyapp\n")
	out := new(bytes.Buffer)

	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloudronURL != "https://my.example.com" {
		t.Fatalf("url=%q, want https://my.example.com", cfg.CloudronURL)
	}
	if cfg.Token != "secret-token" {
		t.Fatalf("token=%q, want secret-token", cfg.Token)
	}
	if cfg.Subdomain != "myapp" {
		t.Fatalf("subdomain=%q, want myapp", cfg.Subdomain)
	}
	if cfg.AllowSelfSigned {
		t.Fatal("AllowSelfSigned should be false for example.com")
	}
}

func TestRunWithIO_URLNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "https://example.com"},
		{"https://example.com", "https://example.com"},
		{"http://example.com", "http://example.com"},
		{"example.com/", "https://example.com"},
		{"https://my.cloud.com///", "https://my.cloud.com"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			in := strings.NewReader(tt.input + "\ntoken\nsub\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.CloudronURL != tt.want {
				t.Fatalf("url=%q, want %q", cfg.CloudronURL, tt.want)
			}
		})
	}
}

func TestRunWithIO_EmptyURL(t *testing.T) {
	in := strings.NewReader("\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "URL is required") {
		t.Fatalf("expected URL required error, got %v", err)
	}
}

func TestRunWithIO_EmptyToken(t *testing.T) {
	in := strings.NewReader("example.com\n\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "token is required") {
		t.Fatalf("expected token required error, got %v", err)
	}
}

func TestRunWithIO_InvalidURL(t *testing.T) {
	in := strings.NewReader("://bad\ntoken\nsub\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "invalid URL") {
		t.Fatalf("expected invalid URL error, got %v", err)
	}
}

func TestRunWithIO_SelfSignedDetection(t *testing.T) {
	devURLs := []string{
		"my.192.168.1.50.nip.io",
		"localhost",
		"192.168.1.1",
		"10.0.0.5",
	}
	for _, u := range devURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\ntoken\nsub\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if !cfg.AllowSelfSigned {
				t.Fatalf("AllowSelfSigned should be true for %s", u)
			}
			if !strings.Contains(out.String(), "WARNING") {
				t.Fatal("expected explicit TLS warning in output")
			}
		})
	}
}

func TestRunWithIO_InvalidSubdomain(t *testing.T) {
	invalidSubs := []string{"My-App", "-bad", "a b", "has_underscore", "CAPS"}
	for _, sub := range invalidSubs {
		t.Run(sub, func(t *testing.T) {
			in := strings.NewReader("example.com\ntoken\n" + sub + "\n")
			out := new(bytes.Buffer)
			_, err := RunWithIO(in, out)
			if err == nil || !strings.Contains(err.Error(), "invalid subdomain") {
				t.Fatalf("expected invalid subdomain error for %q, got %v", sub, err)
			}
		})
	}
}

func TestRunWithIO_ValidSubdomains(t *testing.T) {
	validSubs := []string{"myapp", "my-app", "app123", "a", "a1b2c3"}
	for _, sub := range validSubs {
		t.Run(sub, func(t *testing.T) {
			in := strings.NewReader("example.com\ntoken\n" + sub + "\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatalf("unexpected error for valid subdomain %q: %v", sub, err)
			}
			if cfg.Subdomain != sub {
				t.Fatalf("subdomain=%q, want %q", cfg.Subdomain, sub)
			}
		})
	}
}

func TestRunWithIO_EmptySubdomain(t *testing.T) {
	in := strings.NewReader("example.com\ntoken\n\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("empty subdomain should be allowed: %v", err)
	}
	if cfg.Subdomain != "" {
		t.Fatalf("subdomain=%q, want empty", cfg.Subdomain)
	}
}

func TestRunWithIO_CloudronTokenEnv(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "env-token-123")

	// Only need URL and subdomain when CLOUDRON_TOKEN is set
	in := strings.NewReader("example.com\nmyapp\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "env-token-123" {
		t.Fatalf("token=%q, want env-token-123", cfg.Token)
	}
	if !strings.Contains(out.String(), "CLOUDRON_TOKEN") {
		t.Fatal("expected message about env var in output")
	}
}

func TestAskSubdomainWithIO(t *testing.T) {
	in := strings.NewReader("myapp\n")
	out := new(bytes.Buffer)
	sub, err := AskSubdomainWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if sub != "myapp" {
		t.Fatalf("subdomain=%q, want myapp", sub)
	}
}

func TestAskSubdomainWithIO_Empty(t *testing.T) {
	in := strings.NewReader("\n")
	out := new(bytes.Buffer)
	_, err := AskSubdomainWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestAskSubdomainWithIO_Invalid(t *testing.T) {
	in := strings.NewReader("BAD!\n")
	out := new(bytes.Buffer)
	_, err := AskSubdomainWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "invalid subdomain") {
		t.Fatalf("expected invalid subdomain error, got %v", err)
	}
}

func TestRunWithIO_PipedInputNoTrailingNewline(t *testing.T) {
	// Simulate piped input: printf "url\ntoken\nsub" | ./fastpack-deploy
	in := strings.NewReader("example.com\nmy-token\nmyapp")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("piped input without trailing newline should work: %v", err)
	}
	if cfg.Token != "my-token" {
		t.Fatalf("token=%q", cfg.Token)
	}
	if cfg.Subdomain != "myapp" {
		t.Fatalf("subdomain=%q", cfg.Subdomain)
	}
}

func TestRunWithIO_SelfSignedFalsePositive(t *testing.T) {
	// Production URLs containing "10." should NOT trigger TLS bypass
	safeURLs := []string{
		"api.v10.example.com",
		"my.example10.com",
		"10cloud.example.com",
		"something.10x.io",
	}
	for _, u := range safeURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\ntoken\nsub\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AllowSelfSigned {
				t.Fatalf("AllowSelfSigned should be false for production URL %s", u)
			}
		})
	}
}

// === MUTATION-KILLING TESTS ===
// These tests are designed to catch specific mutations that would otherwise survive.

func TestRunWithIO_NipIoFalsePositive(t *testing.T) {
	// Mutation target: strings.Contains(host, ".nip.io") should NOT match
	// domains that have ".nip.io" in the middle (e.g., evil.nip.io.attacker.com)
	attackerURLs := []string{
		"evil.nip.io.attacker.com",
		"nip.io.evil.com",
		"my.nip.io.co.uk",
	}
	for _, u := range attackerURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\ntoken\nsub\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AllowSelfSigned {
				t.Fatalf("AllowSelfSigned should be false for attacker URL %s — nip.io in middle is NOT a dev instance", u)
			}
		})
	}
}

func TestRunWithIO_NipIoTruePositive(t *testing.T) {
	// Verify legitimate nip.io subdomains still trigger
	legit := []string{"192.168.1.50.nip.io", "10.0.0.1.nip.io", "my.app.nip.io"}
	for _, u := range legit {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\ntoken\nsub\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if !cfg.AllowSelfSigned {
				t.Fatalf("AllowSelfSigned should be true for legit nip.io URL %s", u)
			}
		})
	}
}

func TestRunWithIO_WhitespaceTrimmingURL(t *testing.T) {
	// Mutation target: remove TrimSpace(rawURL) → whitespace preserved in URL
	in := strings.NewReader("  example.com  \ntoken\nsub\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloudronURL != "https://example.com" {
		t.Fatalf("URL should be trimmed, got %q", cfg.CloudronURL)
	}
}

func TestRunWithIO_WhitespaceTrimmingToken(t *testing.T) {
	// Mutation target: remove TrimSpace(token) → whitespace in token
	in := strings.NewReader("example.com\n  my-token  \nsub\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "my-token" {
		t.Fatalf("token should be trimmed, got %q", cfg.Token)
	}
}

func TestRunWithIO_WhitespaceTrimmingSubdomain(t *testing.T) {
	// Mutation target: remove TrimSpace(subdomain) → whitespace in subdomain
	in := strings.NewReader("example.com\ntoken\n  myapp  \n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Subdomain != "myapp" {
		t.Fatalf("subdomain should be trimmed, got %q", cfg.Subdomain)
	}
}

func TestRunWithIO_EOFImmediateEmpty(t *testing.T) {
	// Mutation target: readLine EOF with len(line) > 0 changed to >= 0
	// Empty EOF (Ctrl+D immediately) should return error, not empty string
	in := strings.NewReader("")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil {
		t.Fatal("empty EOF should return error, not proceed with empty input")
	}
}

func TestRunWithIO_TokenWhitespaceOnly(t *testing.T) {
	// Mutation target: remove TrimSpace → whitespace-only token passes validation
	in := strings.NewReader("example.com\n   \n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "token is required") {
		t.Fatalf("whitespace-only token should be rejected, got %v", err)
	}
}

func TestRunWithIO_SubdomainMaxLength(t *testing.T) {
	// DNS subdomain label max length is 63 chars. Our regex enforces {0,61} + 2 chars = 63.
	validMax := "a" + strings.Repeat("b", 61) + "c" // 63 chars
	in := strings.NewReader("example.com\ntoken\n" + validMax + "\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("63-char subdomain should be valid: %v", err)
	}
	if cfg.Subdomain != validMax {
		t.Fatalf("subdomain=%q", cfg.Subdomain)
	}

	// 64 chars should be rejected
	tooLong := validMax + "d"
	in2 := strings.NewReader("example.com\ntoken\n" + tooLong + "\n")
	out2 := new(bytes.Buffer)
	_, err = RunWithIO(in2, out2)
	if err == nil {
		t.Fatal("64-char subdomain should be rejected")
	}
}

func TestRunWithIO_CloudronTokenEnvWhitespace(t *testing.T) {
	// Mutation target: CLOUDRON_TOKEN with whitespace should be used as-is
	// (env vars are not trimmed — the user controls their environment)
	t.Setenv("CLOUDRON_TOKEN", "  spaced-token  ")

	in := strings.NewReader("example.com\nmyapp\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	// Env var tokens are used exactly as provided (spaces and all)
	if cfg.Token != "  spaced-token  " {
		t.Fatalf("env token should be used as-is, got %q", cfg.Token)
	}
}

func TestRunWithIO_CloudronTokenEnvEmpty(t *testing.T) {
	// Empty CLOUDRON_TOKEN should fall through to prompt
	t.Setenv("CLOUDRON_TOKEN", "")

	in := strings.NewReader("example.com\nmy-token\nmyapp\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "my-token" {
		t.Fatalf("empty env should fall through to prompt, got token=%q", cfg.Token)
	}
}

func TestRunWithIO_LocalhostWithPort(t *testing.T) {
	// Verify localhost:PORT is correctly detected as dev instance
	in := strings.NewReader("localhost:3000\ntoken\nsub\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowSelfSigned {
		t.Fatal("localhost:3000 should trigger AllowSelfSigned")
	}
}

func TestRunWithIO_URLWhitespaceOnly(t *testing.T) {
	in := strings.NewReader("   \n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "URL is required") {
		t.Fatalf("whitespace-only URL should be rejected, got %v", err)
	}
}

func TestValidSubdomainRegex(t *testing.T) {
	valid := []string{"a", "ab", "a1", "my-app", "app123", "a-b-c"}
	invalid := []string{"", "-a", "a-", "A", "a_b", "a b", ".a", "a.b"}

	for _, s := range valid {
		if !ValidSubdomain.MatchString(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	for _, s := range invalid {
		if ValidSubdomain.MatchString(s) {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestRunWithIO_BuildServiceWithToken(t *testing.T) {
	// Full flow including Build Service URL + token
	in := strings.NewReader("example.com\ntoken\nmyapp\ndevtools.example.com\nmy-build-token\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BuildServiceURL != "https://devtools.example.com" {
		t.Fatalf("buildServiceURL=%q", cfg.BuildServiceURL)
	}
	if cfg.BuildToken != "my-build-token" {
		t.Fatalf("buildToken=%q", cfg.BuildToken)
	}
	output := out.String()
	if !strings.Contains(output, "Step 1/4") {
		t.Fatal("expected Step 1/4 in output")
	}
	if !strings.Contains(output, "Step 4/4") {
		t.Fatal("expected Step 4/4 in output")
	}
}

func TestRunWithIO_BuildServiceEmptyToken(t *testing.T) {
	// Providing a Build Service URL but empty token should now error
	in := strings.NewReader("example.com\ntoken\nmyapp\ndevtools.example.com\n\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "Build Service token is required") {
		t.Fatalf("expected build token required error, got %v", err)
	}
}

func TestRunWithIO_DynamicStepNumbers(t *testing.T) {
	// When CLOUDRON_TOKEN is set, steps should be 1/3, 2/3, 3/3 (not 1/4)
	t.Setenv("CLOUDRON_TOKEN", "env-tok")
	in := strings.NewReader("example.com\nmyapp\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if !strings.Contains(output, "Step 1/3") {
		t.Fatalf("expected Step 1/3 when token env is set, got:\n%s", output)
	}
	if strings.Contains(output, "Step 1/4") {
		t.Fatal("should not show Step 1/4 when token env is set")
	}
}

func TestRunWithIO_BuildServiceEnvVars(t *testing.T) {
	t.Setenv("CLOUDRON_BUILD_SERVICE_URL", "https://devtools.example.com")
	t.Setenv("CLOUDRON_BUILD_TOKEN", "env-build-tok")

	in := strings.NewReader("example.com\ntoken\nmyapp\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BuildServiceURL != "https://devtools.example.com" {
		t.Fatalf("buildServiceURL=%q", cfg.BuildServiceURL)
	}
	if cfg.BuildToken != "env-build-tok" {
		t.Fatalf("buildToken=%q", cfg.BuildToken)
	}
	output := out.String()
	// With both env vars set, only 3 steps (URL, Token, Subdomain)
	if !strings.Contains(output, "Step 1/3") {
		t.Fatalf("expected Step 1/3 with build env set, got:\n%s", output)
	}
}

func TestRunWithIO_SubdomainHintIncludesDomain(t *testing.T) {
	in := strings.NewReader("my.cloud.example.com\ntoken\nmyapp\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if !strings.Contains(output, "myapp.cloud.example.com") {
		t.Fatalf("expected domain hint in subdomain prompt, got:\n%s", output)
	}
}

func TestMain(m *testing.M) {
	// Clear env vars for tests that don't set them explicitly
	os.Unsetenv("CLOUDRON_TOKEN")
	os.Unsetenv("CLOUDRON_BUILD_SERVICE_URL")
	os.Unsetenv("CLOUDRON_BUILD_TOKEN")
	os.Exit(m.Run())
}
