package wizard

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// === v2.0 flow tests (URL → Username → Password) ===

func TestRunWithIO_V2Flow(t *testing.T) {
	in := strings.NewReader("my.example.com\nadmin\nsecret\n")
	out := new(bytes.Buffer)

	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloudronURL != "https://my.example.com" {
		t.Fatalf("url=%q", cfg.CloudronURL)
	}
	if cfg.Username != "admin" {
		t.Fatalf("username=%q", cfg.Username)
	}
	if cfg.Password != "secret" {
		t.Fatalf("password=%q", cfg.Password)
	}
	if cfg.Token != "" {
		t.Fatalf("token should be empty in v2 flow, got %q", cfg.Token)
	}
	output := out.String()
	if !strings.Contains(output, "Step 1/3") {
		t.Fatal("expected Step 1/3")
	}
	if !strings.Contains(output, "Step 2/3") {
		t.Fatal("expected Step 2/3")
	}
	if !strings.Contains(output, "Step 3/3") {
		t.Fatal("expected Step 3/3")
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
			in := strings.NewReader(tt.input + "\nuser\npass\n")
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

func TestRunWithIO_EmptyUsername(t *testing.T) {
	in := strings.NewReader("example.com\n\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "username is required") {
		t.Fatalf("expected username required error, got %v", err)
	}
}

func TestRunWithIO_EmptyPassword(t *testing.T) {
	in := strings.NewReader("example.com\nadmin\n\n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "password is required") {
		t.Fatalf("expected password required error, got %v", err)
	}
}

func TestRunWithIO_InvalidURL(t *testing.T) {
	in := strings.NewReader("://bad\nuser\npass\n")
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
		"172.17.0.1",
		"172.16.0.1",
		"172.31.255.1",
	}
	for _, u := range devURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\nuser\npass\n")
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

func TestRunWithIO_SelfSignedFalsePositive(t *testing.T) {
	safeURLs := []string{
		"api.v10.example.com",
		"my.example10.com",
		"10cloud.example.com",
		"something.10x.io",
		"172.32.0.1",
		"172.15.0.1",
		"172cloud.example.com",
	}
	for _, u := range safeURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\nuser\npass\n")
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

func TestRunWithIO_WhitespaceTrimmingURL(t *testing.T) {
	in := strings.NewReader("  example.com  \nuser\npass\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloudronURL != "https://example.com" {
		t.Fatalf("URL should be trimmed, got %q", cfg.CloudronURL)
	}
}

func TestRunWithIO_WhitespaceTrimmingUsername(t *testing.T) {
	in := strings.NewReader("example.com\n  admin  \npass\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "admin" {
		t.Fatalf("username should be trimmed, got %q", cfg.Username)
	}
}

func TestRunWithIO_PasswordPreservesSpaces(t *testing.T) {
	// Passwords with leading/trailing spaces must be preserved (not trimmed)
	in := strings.NewReader("example.com\nadmin\n  secret  \n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != "  secret  " {
		t.Fatalf("password spaces should be preserved, got %q", cfg.Password)
	}
}

func TestRunWithIO_EOFImmediateEmpty(t *testing.T) {
	in := strings.NewReader("")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil {
		t.Fatal("empty EOF should return error")
	}
}

func TestRunWithIO_PipedInputNoTrailingNewline(t *testing.T) {
	in := strings.NewReader("example.com\nadmin\nsecret")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("piped input without trailing newline should work: %v", err)
	}
	if cfg.Username != "admin" {
		t.Fatalf("username=%q", cfg.Username)
	}
	if cfg.Password != "secret" {
		t.Fatalf("password=%q", cfg.Password)
	}
}

func TestRunWithIO_UsernameWhitespaceOnly(t *testing.T) {
	in := strings.NewReader("example.com\n   \n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "username is required") {
		t.Fatalf("whitespace-only username should be rejected, got %v", err)
	}
}

func TestRunWithIO_PasswordWhitespaceOnly(t *testing.T) {
	in := strings.NewReader("example.com\nadmin\n   \n")
	out := new(bytes.Buffer)
	_, err := RunWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "password is required") {
		t.Fatalf("whitespace-only password should be rejected, got %v", err)
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

func TestRunWithIO_LocalhostWithPort(t *testing.T) {
	in := strings.NewReader("localhost:3000\nuser\npass\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowSelfSigned {
		t.Fatal("localhost:3000 should trigger AllowSelfSigned")
	}
}

// === Legacy flow tests (CLOUDRON_TOKEN env var) ===

func TestRunWithIO_LegacyTokenFlow(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "env-token-123")

	in := strings.NewReader("example.com\nmyapp\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "env-token-123" {
		t.Fatalf("token=%q, want env-token-123", cfg.Token)
	}
	if cfg.Username != "" {
		t.Fatalf("username should be empty in legacy flow, got %q", cfg.Username)
	}
	output := out.String()
	if !strings.Contains(output, "CLOUDRON_TOKEN") {
		t.Fatal("expected message about CLOUDRON_TOKEN")
	}
	if !strings.Contains(output, "Step 1/2") {
		t.Fatal("expected Step 1/2 in legacy flow")
	}
}

func TestRunWithIO_LegacyTokenEmptyEnv(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "")

	// Should fall through to v2 flow
	in := strings.NewReader("example.com\nadmin\nsecret\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Username != "admin" {
		t.Fatalf("empty env should use v2 flow, got username=%q", cfg.Username)
	}
}

func TestRunWithIO_LegacyInvalidSubdomain(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "tok")

	invalidSubs := []string{"My-App", "-bad", "a b", "has_underscore", "CAPS"}
	for _, sub := range invalidSubs {
		t.Run(sub, func(t *testing.T) {
			in := strings.NewReader("example.com\n" + sub + "\n")
			out := new(bytes.Buffer)
			_, err := RunWithIO(in, out)
			if err == nil || !strings.Contains(err.Error(), "invalid subdomain") {
				t.Fatalf("expected invalid subdomain error for %q, got %v", sub, err)
			}
		})
	}
}

func TestRunWithIO_LegacyValidSubdomains(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "tok")

	validSubs := []string{"myapp", "my-app", "app123", "a", "a1b2c3"}
	for _, sub := range validSubs {
		t.Run(sub, func(t *testing.T) {
			in := strings.NewReader("example.com\n" + sub + "\n")
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

func TestRunWithIO_LegacyEmptySubdomain(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "tok")

	in := strings.NewReader("example.com\n\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("empty subdomain should be allowed: %v", err)
	}
	if cfg.Subdomain != "" {
		t.Fatalf("subdomain=%q, want empty", cfg.Subdomain)
	}
}

// === NIP.IO edge cases ===

func TestRunWithIO_NipIoFalsePositive(t *testing.T) {
	attackerURLs := []string{
		"evil.nip.io.attacker.com",
		"nip.io.evil.com",
		"my.nip.io.co.uk",
	}
	for _, u := range attackerURLs {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\nuser\npass\n")
			out := new(bytes.Buffer)
			cfg, err := RunWithIO(in, out)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AllowSelfSigned {
				t.Fatalf("AllowSelfSigned should be false for attacker URL %s", u)
			}
		})
	}
}

func TestRunWithIO_NipIoTruePositive(t *testing.T) {
	legit := []string{"192.168.1.50.nip.io", "10.0.0.1.nip.io", "my.app.nip.io"}
	for _, u := range legit {
		t.Run(u, func(t *testing.T) {
			in := strings.NewReader(u + "\nuser\npass\n")
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

// === Ask2FA tests ===

func TestAsk2FAWithIO(t *testing.T) {
	in := strings.NewReader("123456\n")
	out := new(bytes.Buffer)
	code, err := Ask2FAWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if code != "123456" {
		t.Fatalf("code=%q, want 123456", code)
	}
	if !strings.Contains(out.String(), "Two-factor") {
		t.Fatal("expected 2FA prompt in output")
	}
}

func TestAsk2FAWithIO_Empty(t *testing.T) {
	in := strings.NewReader("\n")
	out := new(bytes.Buffer)
	_, err := Ask2FAWithIO(in, out)
	if err == nil || !strings.Contains(err.Error(), "2FA code is required") {
		t.Fatalf("expected 2FA code required error, got %v", err)
	}
}

func TestAsk2FAWithIO_Whitespace(t *testing.T) {
	in := strings.NewReader("  654321  \n")
	out := new(bytes.Buffer)
	code, err := Ask2FAWithIO(in, out)
	if err != nil {
		t.Fatal(err)
	}
	if code != "654321" {
		t.Fatalf("code should be trimmed, got %q", code)
	}
}

// === AskSubdomain tests ===

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

// === Regex tests ===

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

func TestRunWithIO_SubdomainMaxLength(t *testing.T) {
	t.Setenv("CLOUDRON_TOKEN", "tok")

	validMax := "a" + strings.Repeat("b", 61) + "c" // 63 chars
	in := strings.NewReader("example.com\n" + validMax + "\n")
	out := new(bytes.Buffer)
	cfg, err := RunWithIO(in, out)
	if err != nil {
		t.Fatalf("63-char subdomain should be valid: %v", err)
	}
	if cfg.Subdomain != validMax {
		t.Fatalf("subdomain=%q", cfg.Subdomain)
	}

	tooLong := validMax + "d"
	in2 := strings.NewReader("example.com\n" + tooLong + "\n")
	out2 := new(bytes.Buffer)
	_, err = RunWithIO(in2, out2)
	if err == nil {
		t.Fatal("64-char subdomain should be rejected")
	}
}

func TestMain(m *testing.M) {
	os.Unsetenv("CLOUDRON_TOKEN")
	// Override password reader to use stdin (not terminal) in tests
	readPasswordFn = func() (string, error) {
		return "", fmt.Errorf("not a terminal")
	}
	os.Exit(m.Run())
}

func TestRunWithIO_FileConfigFullV2Flow(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{
		"cloudronUrl": "my.example.com",
		"username": "admin",
		"password": "secret",
		"subdomain": "myapp"
	}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloudronURL != "https://my.example.com" {
		t.Fatalf("url=%q", cfg.CloudronURL)
	}
	if cfg.Username != "admin" || cfg.Password != "secret" || cfg.Subdomain != "myapp" {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
	if !strings.Contains(out.String(), "Using deploy config") {
		t.Fatal("expected deploy config message")
	}
	if strings.Contains(out.String(), "Step 1/3") || strings.Contains(out.String(), "Step 2/3") || strings.Contains(out.String(), "Step 3/3") {
		t.Fatal("did not expect prompts for configured fields")
	}
}

func TestRunWithIO_FileConfigPromptsForMissingPassword(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{"cloudronUrl":"example.com","username":"admin"}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader("secret\n"), out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Password != "secret" {
		t.Fatalf("password=%q", cfg.Password)
	}
	if !strings.Contains(out.String(), "Step 3/3") {
		t.Fatal("expected password prompt")
	}
}

func TestRunWithIO_FileConfigHonorsExplicitAllowSelfSignedFalse(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{
		"cloudronUrl": "localhost",
		"username": "admin",
		"password": "secret",
		"allowSelfSigned": false
	}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowSelfSigned {
		t.Fatal("explicit allowSelfSigned=false should keep TLS verification enabled for dev-looking URLs")
	}
	if strings.Contains(out.String(), "WARNING") {
		t.Fatal("did not expect TLS warning when self-signed certificates are explicitly disabled")
	}
}

func TestRunWithIO_FileConfigHonorsExplicitAllowSelfSignedFalseWithToken(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{
		"cloudronUrl": "localhost",
		"token": "tok",
		"subdomain": "myapp",
		"allowSelfSigned": false
	}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowSelfSigned {
		t.Fatal("explicit allowSelfSigned=false should keep TLS verification enabled in token flow")
	}
	if strings.Contains(out.String(), "WARNING") {
		t.Fatal("did not expect TLS warning when self-signed certificates are explicitly disabled")
	}
}

func TestRunWithIO_FileConfigWarnsForExplicitAllowSelfSignedTrue(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{
		"cloudronUrl": "example.com",
		"username": "admin",
		"password": "secret",
		"allowSelfSigned": true
	}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowSelfSigned {
		t.Fatal("explicit allowSelfSigned=true should disable TLS verification")
	}
	if !strings.Contains(out.String(), "WARNING: TLS verification is disabled") {
		t.Fatal("expected TLS warning when self-signed certificates are explicitly enabled")
	}
}

func TestRunWithIO_FileConfigWarnsForExplicitAllowSelfSignedTrueWithToken(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	configJSON := `{
		"cloudronUrl": "localhost",
		"token": "tok",
		"subdomain": "myapp",
		"allowSelfSigned": true
	}`
	if err := os.WriteFile("fastpack-deploy.json", []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowSelfSigned {
		t.Fatal("explicit allowSelfSigned=true should disable TLS verification in token flow")
	}
	if !strings.Contains(out.String(), "WARNING: TLS verification is disabled") {
		t.Fatal("expected TLS warning when self-signed certificates are explicitly enabled in token flow")
	}
}

func TestRunWithIO_CustomFileConfigPath(t *testing.T) {
	dir := t.TempDir()
	path := dir + string(os.PathSeparator) + "deploy.json"
	configJSON := `{"cloudronUrl":"example.com","token":"tok","subdomain":"myapp"}`
	if err := os.WriteFile(path, []byte(configJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FASTPACK_DEPLOY_CONFIG", path)

	out := new(bytes.Buffer)
	cfg, err := RunWithIO(strings.NewReader(""), out)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "tok" || cfg.Subdomain != "myapp" {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
}
