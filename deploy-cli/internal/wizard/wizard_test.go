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

func TestMain(m *testing.M) {
	// Clear CLOUDRON_TOKEN for tests that don't set it explicitly
	os.Unsetenv("CLOUDRON_TOKEN")
	os.Exit(m.Run())
}
