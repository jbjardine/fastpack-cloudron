# Architecture

How FastPackCloudron works under the hood.

## Overview

FastPackCloudron is a **client-side code generator**. It runs entirely in the browser with zero backend. User input goes through a pipeline of pure functions that produce deployment-ready files.

```
User Input (form)
    ↓
buildConfig()         → Reads 80+ form elements into a config object
    ↓
validate(config)      → Returns errors[] and warnings[]
    ↓
generate*(config)     → Pure functions that produce file content strings
    ↓
Preview + ZIP Download
```

## File Structure

```
fastpack-cloudron/
├── index.html              # UI: form, CSS, ARIA tabs, progressive disclosure
├── app.js                  # Controller: form handling, validation, events, localStorage
├── generators.js           # Model: pure functions that generate file content
├── test.html               # 207 browser-based unit tests
├── test-ci.mjs             # Headless test runner (Playwright)
├── test-build.mjs          # Docker build integration tests
├── test-cloudron.mjs       # E2E tests on real Cloudron
├── test-e2e-full.mjs       # Exhaustive E2E (10 configs)
├── test-go-deploy-e2e.mjs  # Full user flow E2E (UI → ZIP → Go CLI → Deploy)
├── deploy-cli/             # Go Deploy CLI (zero-dependency Cloudron deployer)
│   ├── main.go             # Entry point: 7-step deploy flow
│   ├── Makefile            # Cross-compilation for all platforms
│   └── internal/
│       ├── wizard/         # Interactive terminal prompts
│       ├── api/            # Cloudron API + Build Service client
│       └── archive/        # Strict allow-list tarball creator
└── extras/
    └── builder-registry/   # Combined Docker Builder + Registry Cloudron app
```

## Data Flow

### 1. Form → Config Object

`buildConfig()` in `app.js` reads every form element and constructs a normalized config object:

```javascript
{
  image: "node:20-alpine",
  id: "io.fastpack.node",
  title: "Node",
  version: "1.0.0",
  httpPort: 8000,
  hasWebUI: true,
  stack: "nodejs",
  database: "postgresql",
  sso: "oidc",
  addons: ["localstorage"],
  services: [],
  subcontainers: [],
  // ... 40+ more fields
}
```

### 2. Config → Validation

`validate(config)` checks the config and returns structured errors/warnings:

```javascript
{
  errors: [
    { field: "docker-image", message: "Invalid image name..." }
  ],
  warnings: [
    { message: "Without localstorage, your app cannot persist data." }
  ]
}
```

Validation includes:
- **Input safety**: Regex checks against injection (SAFE_DOCKER_REF, SAFE_IDENTIFIER, etc.)
- **Cloudron conventions**: Port ranges, ID format, healthcheck path
- **Smart warnings**: Distroless/busybox detection, Alpine/Debian mismatch for COPY --from=

### 3. Config → Generated Files

Each `generate*()` function is **pure** — config in, string out, no side effects:

| Function | Output | Notes |
|----------|--------|-------|
| `generateManifest(config)` | CloudronManifest.json | Uses `setIfPresent()`/`setIfArray()` helpers |
| `generateDockerfile(config)` | Dockerfile | Multi-distro (apt/apk/dnf), gosu, Docker CLI for DooD |
| `generateStartSh(config)` | start.sh | 4 modes: single, multi-service, TCP-only, DooD |
| `generateNginxConf(config)` | nginx.conf | For multi-service and DooD (with IP templates) |
| `generateDescription(config)` | DESCRIPTION.md | Markdown description |
| `generateDockerignore()` | .dockerignore | Static list |
| `generateReadme(config)` | README.md | With env var reference per addon |
| `generateDeploySh()` | deploy.js | Cross-platform deploy script |
| `generateDeployCmd()` | deploy.cmd | Windows launcher |
| `generateCloudronVersions(config)` | CloudronVersions.json | For App Store publishing |
| `generatePostInstall(config)` | POSTINSTALL.md | Checklist stub |
| `generateChangelog(config)` | CHANGELOG.md | Version history stub |

### 4. Preview + Download

The preview shows all generated files in ARIA-compliant tabs with keyboard navigation. The download button creates a ZIP using JSZip (client-side).

## Security Model

### Defense in Depth

Two layers of protection against injection in generated files:

1. **`validate()` in app.js** — User-friendly error messages with shared regex patterns
2. **`assert*()` in generators.js** — Fail-closed guards that throw if input is unsafe

```
User Input → validate() → Error shown in UI
                 ↓ (if valid)
          → assert*() → Throws if somehow bypassed
                 ↓ (if safe)
          → String interpolation into Dockerfile/shell
```

### Regex Patterns

| Pattern | Used For | Allows |
|---------|----------|--------|
| `SAFE_DOCKER_REF` | Image names, COPY --from | `[a-zA-Z0-9._/-]`, tag, SHA256 digest |
| `SAFE_PATH` | File paths | `[a-zA-Z0-9._/-]` |
| `SAFE_IDENTIFIER` | Service names | `[a-zA-Z][a-zA-Z0-9_-]*` |
| `SAFE_VERSION` | Version strings | `[a-zA-Z0-9][a-zA-Z0-9._-]*` |
| `SAFE_ROUTE_PATH` | Nginx routes | `/[a-zA-Z0-9._/-]*` |

### Intentionally Unvalidated

Service commands (`svc.command`) and scheduler commands are **intentionally** free-form — they are shell commands the user writes for execution on their own server.

## Start.sh Modes

FastPackCloudron generates different start.sh patterns based on the configuration:

### Mode 1: DooD (Sub-containers)

When `config.subcontainers.length > 0`:
- Exports `DOCKER_HOST`
- Runs `docker run` for each sub-container with resource limits
- Discovers IPs via `docker inspect`
- Renders nginx config from template
- Trap cleanup on SIGTERM

### Mode 2: Multi-Service

When `config.services.length > 0`:
- Background-launches each service via `gosu`
- Starts nginx reverse proxy
- Trap forwards SIGTERM to all children

### Mode 3: Single Web UI

When `config.hasWebUI && !services && !subcontainers`:
- `exec gosu cloudron:cloudron <command>`
- PID 1 is the app process (SIGTERM goes directly to it)

### Mode 4: TCP-only Background Service

When `!config.hasWebUI`:
- Background-launches the service
- Starts a Python minimal HTTP server for healthcheck
- Trap forwards SIGTERM

## Cloudron Conventions

FastPackCloudron generates files that follow these Cloudron patterns:

| Convention | Implementation |
|-----------|---------------|
| Non-root execution | `cloudron` user (uid 808) + gosu |
| Read-only filesystem | Writes only to `/app/data/` (localstorage) |
| Signal handling | `exec` for single-process, `trap` for multi-process |
| Health check | Manifest `healthCheckPath` + Python server for TCP-only |
| Localstorage init | `.initialized` guard file in `/app/data/` |
| Multi-distro support | Auto-detects apt-get, apk, dnf, and fallback |

## Go Deploy CLI

The Go Deploy CLI (`deploy-cli/`) is a zero-dependency binary that replaces the `cloudron` CLI for deploying FastPackCloudron-generated packages. See [deploy-cli/README.md](../deploy-cli/README.md) for full documentation.

### End-to-End Flow

```
Browser (index.html)
    ↓ downloadZip()
ZIP file (CloudronManifest.json + Dockerfile + start.sh + ...)
    ↓ User extracts
Package folder + Go binary
    ↓ ./fastpack-deploy
Wizard (4 steps) → Cloudron API (verify) → Build Service (upload + build) → Install
    ↓
App running at https://myapp.your-domain.com
```

### Build Service Protocol

The Cloudron Build Service uses a non-standard auth mechanism:
- **Auth**: `?accessToken=<token>` query parameter (not Bearer headers)
- **Upload**: `POST /api/v1/builds` with multipart form (`sourceArchive` + metadata fields)
- **Install**: `POST /api/v1/apps` with `subdomain` + `domain` + `dockerImage` inside manifest (Cloudron v9)

### builder-registry (Test Infrastructure)

`extras/builder-registry/` is a custom Cloudron app that combines the Docker Builder and Docker Registry into a single app slot. Used for development and testing.

```
nginx (port 8000)
├── /v2/*  → Docker Registry (port 5000)
└── /*     → Build Service (port 3000, from cloudron/io.cloudron.buildservice)
```

## Testing Pyramid

```
Unit Tests (207)           ← test.html (generators, manifests, security)
    ↓
Go Unit Tests (75)         ← deploy-cli go test ./... (API, wizard, archive)
    ↓
Build Tests (11)           ← test-build.mjs (Docker builds succeed)
    ↓
E2E Tests (20)             ← test-go-deploy-e2e.mjs (UI → ZIP → Go CLI → Deploy → HTTP 200)
    ↓
Full E2E (10)              ← test-e2e-full.mjs (10 configs on live Cloudron)
    ↓
Manual E2E                 ← test-cloudron.mjs (9 configs on live Cloudron)
```

## Accessibility

- `:focus-visible` on all interactive elements
- ARIA tabs with keyboard navigation (arrows, Home, End)
- `aria-live` regions for dynamic errors/warnings
- Dark mode via `prefers-color-scheme`
- Reduced motion support
- `<fieldset>/<legend>` for grouped controls

## LocalStorage Persistence

Form state is automatically saved to `localStorage` on every change and restored on page load. The key is `fastpack-cloudron-config` and includes all form fields, checkboxes, addon selections, and tag choices.
