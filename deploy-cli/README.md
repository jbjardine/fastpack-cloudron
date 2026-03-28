# FastPack Deploy CLI

Zero-dependency Cloudron deployer. Builds Docker images on your Cloudron's Build Service and installs apps via the REST API. No Node.js, no `cloudron` CLI, no local Docker needed.

**Version**: 1.1.0 | **Platforms**: Windows, macOS (Intel + ARM), Linux | **Binary size**: ~6 MB

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

The wizard guides you through 4 steps:

```
Step 1/4: Enter your Cloudron URL
   Example: https://my.example.com
   URL: my.example.com

Step 2/4: Enter your API token
   Create one at: https://my.example.com/#/settings (Profile > API Access)
   Token: ****

Step 3/4: Choose a subdomain for your app
   Example: myapp  (your app will be at myapp.example.com)
   Subdomain: myapp

Step 4/4: Build Service (builds Docker images on your Cloudron)
   Build Service URL (e.g., devtools.example.com): devtools.example.com
   Build Service Token: ****
```

## How It Works

```
1. Detect CloudronManifest.json in current directory
2. Wizard collects: Cloudron URL, API token, subdomain, Build Service
3. Verify Cloudron connection (/api/v1/profile + /api/v1/cloudron/status)
4. Verify Build Service auth (/api/v1/profile?accessToken=...)
5. Create tarball from package files (strict allow-list)
6. Upload to Build Service (POST /api/v1/builds?accessToken=...)
7. Poll build status with exponential backoff + progress dots
8. Install app (POST /api/v1/apps with subdomain + domain + manifest)
9. Display deployed app URL
```

## Prerequisites

- A **Cloudron** server (v9+) — [cloudron.io](https://cloudron.io)
- The **Docker Builder** app installed on your Cloudron — [App Store](https://www.cloudron.io/store/io.cloudron.buildservice.html)
- A **Cloudron API token** — Dashboard > Profile > API Access > Create Token
- A **Build Service token** — Open the Docker Builder app > Setup page > Copy token

## Environment Variables

For non-interactive use (CI/CD, scripts):

| Variable | Description |
|----------|-------------|
| `CLOUDRON_TOKEN` | Cloudron API token (skips Step 2) |
| `CLOUDRON_BUILD_SERVICE_URL` | Build Service URL (skips Step 4 URL prompt) |
| `CLOUDRON_BUILD_TOKEN` | Build Service token (required with URL) |

```bash
# Non-interactive deployment
CLOUDRON_TOKEN=abc123 \
CLOUDRON_BUILD_SERVICE_URL=https://devtools.example.com \
CLOUDRON_BUILD_TOKEN=xyz789 \
./fastpack-deploy-linux-amd64 <<< "https://my.example.com
myapp"
```

## Architecture

```
deploy-cli/
├── main.go                         # Entry point: 7-step deploy flow
├── Makefile                        # Cross-compilation for all platforms
├── go.mod
├── internal/
│   ├── wizard/
│   │   ├── wizard.go               # Interactive terminal prompts
│   │   └── wizard_test.go          # 33 tests (dynamic steps, env vars, validation)
│   ├── api/
│   │   ├── client.go               # Cloudron API + Build Service client
│   │   ├── client_test.go          # 27 tests (auth, build, install)
│   │   └── integration_test.go     # 3 full deployment flow tests
│   └── archive/
│       ├── tarball.go              # Strict allow-list tarball creator
│       └── tarball_test.go         # 14 tests (security, size limits)
└── dist/                           # Cross-compiled binaries
    ├── fastpack-deploy-windows-amd64.exe
    ├── fastpack-deploy-linux-amd64
    ├── fastpack-deploy-darwin-arm64
    └── fastpack-deploy-darwin-amd64
```

### Design Principles

- **Zero dependencies** — stdlib only (`net/http`, `encoding/json`, `mime/multipart`)
- **Testable** — all I/O is injectable (`RunWithIO(reader, writer)`)
- **Security first** — strict allow-list tarball (12 files max), no token in logs
- **Newbie-friendly** — actionable error messages with links and instructions

## Build Service API

The CLI communicates with two separate APIs:

### Cloudron API (`my.DOMAIN`)

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/api/v1/profile` | GET | `Bearer <token>` | Verify API token |
| `/api/v1/cloudron/status` | GET | `Bearer <token>` | Get version + domain |
| `/api/v1/apps` | POST | `Bearer <token>` | Install app |

### Build Service API (`devtools.DOMAIN`)

| Endpoint | Method | Auth | Purpose |
|----------|--------|------|---------|
| `/api/v1/profile` | GET | `?accessToken=<token>` | Verify Build Service auth |
| `/api/v1/builds` | POST | `?accessToken=<token>` | Upload tarball + start build |
| `/api/v1/builds/:id` | GET | `?accessToken=<token>` | Poll build status |
| `/api/v1/builds/:id/push` | POST | `?accessToken=<token>` | Push image to registry |

> **Important**: The Build Service uses `?accessToken=` query parameters, NOT `Authorization: Bearer` headers. This is a Cloudron-specific pattern for OIDC addon apps.

## Tarball Allow-List

Only these files are included in the upload (security measure):

```
CloudronManifest.json    (required)
Dockerfile               (required)
Dockerfile.cloudron      (alternative)
start.sh                 (required)
.dockerignore            (recommended)
nginx.conf               (optional)
icon.png                 (optional)
DESCRIPTION.md           (optional)
CHANGELOG.md             (optional)
README.md                (optional)
CloudronVersions.json    (optional)
```

Maximum tarball size: **100 MB**. Files like `.env`, `node_modules/`, SSH keys are never included.

## Building from Source

Requires Go 1.22+.

```bash
# Build for current platform
cd deploy-cli
go build -o fastpack-deploy .

# Cross-compile for all platforms
make all

# Run tests
go test ./...

# Static analysis
go vet ./...
```

## Testing

75 Go tests covering:

| Package | Tests | Coverage |
|---------|-------|----------|
| `internal/api` | 27 | Auth, build image, install, full flow, error handling |
| `internal/archive` | 14 | Allow-list, size limits, secret exclusion |
| `internal/wizard` | 34 | URL normalization, subdomain validation, env vars, dynamic steps |

Plus a full E2E test (`test-go-deploy-e2e.mjs`) that validates:
```
UI form fill → ZIP download → Extract → Go CLI binary → Build Service → Docker build → Install → HTTP 200
```

Run E2E:
```bash
CLOUDRON_BUILD_TOKEN=<token> node test-go-deploy-e2e.mjs
```

## Error Messages

The CLI provides actionable error messages:

| Error | What It Means | What To Do |
|-------|--------------|------------|
| `no Build Service configured` | Missing Build Service URL | Enter it in Step 4 or set `CLOUDRON_BUILD_SERVICE_URL` |
| `no Build Service token` | Missing Build Service token | Get from Build Service Setup page |
| `Build Service auth failed (HTTP 401)` | Token invalid/expired | Get new token from Build Service |
| `build conflict (HTTP 409)` | Another build is running | Wait for current build to finish |
| `build timed out after 10 minutes` | Build took too long | Check Build Service resources |
| `subdomain already in use (HTTP 409)` | Subdomain taken | Choose a different subdomain |

## Changelog

### v1.1.0 (2026-03-28)

- **Fixed**: Build Service auth — uses `?accessToken=` query param (Cloudron convention)
- **Fixed**: Cloudron v9 install API — `subdomain` + `domain` fields, `dockerImage` in manifest
- **Fixed**: Multipart upload — `sourceArchive` field with metadata
- **Added**: `VerifyBuildService()` early auth check
- **Added**: Dynamic wizard step numbering
- **Added**: Exponential backoff polling with progress dots
- **Removed**: Silent Cloudron token fallback to Build Service (security fix)
- **Improved**: Actionable error messages with links

### v1.0.0 (2026-03-28)

- Initial release: wizard, Cloudron API v9, tarball creation, build polling, app installation
