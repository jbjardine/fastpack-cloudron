# FastPack Deploy CLI

Zero-dependency Cloudron deployer. Uploads your app directly to Cloudron via the `sourceArchive` API — no Build Service, no Docker Registry, no local Docker needed.

**Version**: 2.0.0 | **Platforms**: Windows, macOS (Intel + ARM), Linux | **Binary size**: ~6 MB

## Quick Start

```bash
# 1. Download the binary for your platform
#    (or copy it from the ZIP after generating a package on FastPackCloudron)

# 2. Run it from the extracted package directory
cd my-app-package/
./fastpack-deploy-linux-amd64     # Linux
./fastpack-deploy-darwin-arm64    # macOS Apple Silicon
./fastpack-deploy-darwin-amd64    # macOS Intel
.\fastpack-deploy-windows-amd64.exe  # Windows
```

The wizard will ask for:
1. Your Cloudron URL (e.g., `https://my.example.com`)
2. Your Cloudron username
3. Your Cloudron password (+ 2FA code if enabled)

Then it packages, uploads, and deploys your app automatically.

## How It Works

```
┌──────────────────┐     ┌───────────────────────┐
│  FastPack CLI     │     │  Your Cloudron         │
│                   │     │                        │
│  1. Login (API)   │────▶│  /api/v1/auth/login    │
│  2. Package files │     │                        │
│  3. Upload tar.gz │────▶│  /api/v1/apps           │
│                   │     │  (sourceArchive field)  │
│                   │     │  4. Docker build        │
│                   │     │  5. App running!        │
└──────────────────┘     └───────────────────────┘
```

No Build Service app needed. Cloudron builds the Docker image internally.

## Authentication

### Interactive (default)
The CLI prompts for username and password. If 2FA is enabled, it will ask for your TOTP code.

### Environment variable (CI/scripts)
```bash
export CLOUDRON_TOKEN="your-api-token"
./fastpack-deploy-linux-amd64
```

When `CLOUDRON_TOKEN` is set, the CLI uses it directly as a Bearer token (legacy flow). The wizard only asks for the Cloudron URL and subdomain.

## What Gets Deployed

Only these files are packaged (strict allow-list):

| File | Required |
|------|----------|
| `CloudronManifest.json` | Yes |
| `Dockerfile` | Yes (or `Dockerfile.cloudron`) |
| `start.sh` | No |
| `.dockerignore` | No |
| `nginx.conf` | No |
| `icon.png` | No |
| `DESCRIPTION.md`, `CHANGELOG.md`, `README.md` | No |
| `CloudronVersions.json` | No |

All other files are excluded — no `.env`, no `node_modules`, no secrets.

## Auto-Detection

- **Existing app**: If an app is already installed at the chosen subdomain, the CLI offers to update it instead.
- **Self-signed certs**: Dev instances (localhost, `*.nip.io`, private IPs) automatically accept self-signed certificates with a warning.

## Building from Source

```bash
cd deploy-cli/
go build -o fastpack-deploy .
```

Cross-compile for all platforms:
```bash
make all   # Builds linux, darwin (arm64+amd64), windows
```

## Architecture

```
deploy-cli/
├── main.go                      # CLI entry point, orchestrates the flow
├── internal/
│   ├── api/
│   │   ├── client.go            # Cloudron API: Login, Install, Update, FindApp
│   │   ├── client_test.go       # 30 unit tests with httptest mocks
│   │   └── integration_test.go  # Full flow E2E tests (login → install → update)
│   ├── archive/
│   │   ├── tarball.go           # Allow-list tar.gz creation
│   │   └── tarball_test.go      # 14 tests including security checks
│   └── wizard/
│       ├── wizard.go            # Interactive 3-step wizard + 2FA + legacy flow
│       └── wizard_test.go       # 33 tests including mutation-killing tests
├── Makefile                     # Cross-compilation targets
└── go.mod                       # Go 1.22, zero external dependencies
```

## Changelog

### v2.0.0 (Breaking)
- **Login with username/password** instead of manually copying API tokens
- **Direct sourceArchive upload** — no Build Service or Docker Registry needed
- **2FA (TOTP) support** — automatic detection and prompt
- **3-step wizard** (URL, Username, Password) instead of 4 steps
- **`CLOUDRON_TOKEN`** env var still supported as legacy fallback
- Removed: `CLOUDRON_BUILD_SERVICE_URL`, `CLOUDRON_BUILD_TOKEN`

### v1.1.0
- Auto-detect existing app and offer update
- Auto-detect Docker image registry from Build Service history
- Keep terminal open after deploy on Windows
