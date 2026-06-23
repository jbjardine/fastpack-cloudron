# FastPack Deploy CLI

Zero-dependency Cloudron deployer for users. Uploads a source archive to Cloudron's custom-app install/update API — no Build Service, Docker Registry, or local Docker needed.

**Version**: 2.1.3 | **Platforms**: Windows, macOS (Intel + ARM), Linux | **Binary size**: ~6 MB

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
│                   │     │  (source archive)       │
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

### Configuration file (repeat deploys)

To avoid retyping the Cloudron server and credentials for each field visit, place a `fastpack-deploy.json` file next to `CloudronManifest.json` in the extracted package folder:

```json
{
  "cloudronUrl": "https://my.example.com",
  "username": "admin",
  "password": "your-password"
}
```

You can also use an API token instead of username/password and optionally pin the app subdomain:

```json
{
  "cloudronUrl": "https://my.example.com",
  "token": "your-api-token",
  "subdomain": "myapp",
  "allowSelfSigned": false
}
```

The CLI only prompts for missing values, so with the file above the technician can enter just the app name/subdomain when needed. When a config file supplies `cloudronUrl`, the CLI asks you to confirm the exact destination before sending credentials or tokens. To keep credentials outside the package folder, set `FASTPACK_DEPLOY_CONFIG=/path/to/deploy.json`. `CLOUDRON_TOKEN` still overrides any token from the config file for existing scripts. Use `allowSelfSigned` to explicitly enable or disable self-signed certificate support for dev-looking Cloudron URLs.

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

All other files are excluded — no `.env`, no `node_modules`, no secrets. Allowed entries must be regular files; symlinks are rejected instead of being followed.

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
│   │   └── tarball_test.go      # 15 tests including security checks
│   └── wizard/
│       ├── wizard.go            # Interactive 3-step wizard + 2FA + legacy flow
│       └── wizard_test.go       # 34 tests including mutation-killing tests
├── Makefile                     # Cross-compilation targets
└── go.mod                       # Go 1.26.4, no runtime dependency for users
```

## Changelog

### v2.1.3
- **Deploy config safety**: config-supplied Cloudron URLs must be confirmed before credentials or tokens are sent
- **Archive hardening**: source archives reject symlinks and non-regular files instead of following them
- **DooD integrity**: generated Dockerfiles verify the Docker CLI tarball checksum before installing it

### v2.1.2
- **Public-readiness hardening**: rebuilds target Go 1.26.4 and updated `golang.org/x/sys`/`x/term`
- **Release assets**: tag builds now publish binaries and SHA256 checksums to GitHub Releases

### v2.1.1
- **TLS warning fix**: explicit `allowSelfSigned: true` in deploy config now always prints the TLS verification warning

### v2.1.0
- **Deploy config file support** via `fastpack-deploy.json` or `FASTPACK_DEPLOY_CONFIG`
- **Reusable credentials and subdomain**: configure URL, username/password, token, subdomain, and TLS behavior
- **Partial config prompting**: the wizard asks only for missing values
- **Explicit TLS control**: `allowSelfSigned: false` is honored even for localhost, private IPs, and `*.nip.io`

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
