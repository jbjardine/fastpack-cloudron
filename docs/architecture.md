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
├── index.html         # UI: form, CSS, ARIA tabs, progressive disclosure
├── app.js             # Controller: form handling, validation, events, localStorage
├── generators.js      # Model: pure functions that generate file content
├── test.html          # 313 browser-based unit tests
├── test-ci.mjs        # Headless test runner (Playwright)
├── test-build.mjs     # Docker build integration tests
├── test-cloudron.mjs  # E2E tests on real Cloudron
├── test-e2e-full.mjs  # Exhaustive E2E (10 configs)
└── extras/
    └── builder-registry/  # Combined Builder+Registry Cloudron app
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

## Testing Pyramid

```
Unit Tests (313)           ← test.html (generators, manifests, security)
    ↓
Build Tests (11)           ← test-build.mjs (Docker builds succeed)
    ↓
E2E Tests (10)             ← test-e2e-full.mjs (real Cloudron deploy)
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
