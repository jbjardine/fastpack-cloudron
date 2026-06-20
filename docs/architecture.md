# Architecture

FastPackCloudron is a client-side Cloudron package generator. The app runs in the browser, stores form state locally, and downloads a ZIP with the generated package files.

## Main Flow

```text
Form input
  -> buildConfigFromStore()
  -> validate(config)
  -> generate*(config)
  -> preview tabs
  -> ZIP download
```

There is no backend and no user data is sent to FastPackCloudron.

## Repository Layout

```text
index.html              UI, styles, and deploy wizard markup
app.js                  Alpine controller, validation, ZIP assembly
generators.js           Pure package file generators
test.html               Browser unit tests
test-ci.mjs             Playwright runner for test.html
test-build.mjs          Docker build validation for generated packages
deploy-cli/             Go deploy CLI
docs/                   Public documentation
vendor/                 Vendored browser libraries
```

## Generated Files

| Function | Output |
|----------|--------|
| `generateManifest(config)` | `CloudronManifest.json` |
| `generateDockerfile(config)` | `Dockerfile` |
| `generateStartSh(config)` | `start.sh` |
| `generateDockerignore()` | `.dockerignore` |
| `generateReadme(config)` | `README.md` |
| `generateDescription(config)` | `DESCRIPTION.md` |
| `generateCloudronVersions(config)` | `CloudronVersions.json` |
| `generatePostInstall(config)` | `POSTINSTALL.md` |
| `generateChangelog(config)` | `CHANGELOG.md` |
| `generateNginxConf(config)` | `nginx.conf` for multi-service and DooD modes |

## Security Model

Input safety has two layers:

1. `validate(config)` returns user-facing errors before download.
2. `generators.js` has fail-closed assertions before interpolating values into Dockerfiles, shell scripts, or nginx config.

Important safe patterns:

| Pattern | Used For |
|---------|----------|
| `SAFE_DOCKER_REF` | Docker images and `COPY --from` images |
| `SAFE_PATH` | File and volume paths |
| `SAFE_IDENTIFIER` | Service names |
| `SAFE_VERSION` | Version strings |
| `SAFE_ROUTE_PATH` | nginx route paths |

Service commands and scheduler commands are intentionally free-form because they are commands the package author chooses to run in their own app.

## Runtime Modes

`generateStartSh(config)` has four main modes:

| Mode | Trigger |
|------|---------|
| Single web process | `hasWebUI` with no services or subcontainers |
| TCP/background service | `hasWebUI` disabled |
| Multi-service | `services.length > 0` |
| DooD subcontainers | `subcontainers.length > 0` |

Generated packages follow Cloudron conventions: non-root `cloudron` user, `/app/data` for persistence, stdout/stderr logs, health checks, and signal handling.

## Deploy CLI

`deploy-cli/` builds a standalone Go binary. It:

1. Reads deploy config or prompts for Cloudron URL and credentials.
2. Logs in through Cloudron.
3. Creates a tar.gz from a strict allow-list of package files.
4. Installs or updates the app through Cloudron's custom-app API.

The CLI intentionally does not upload arbitrary source trees. For custom apps with source files, use the official Cloudron CLI from the source directory.

## Tests

Use the narrowest test set for the area touched:

```bash
npm test
npm run test:build
npm run test:go
```

Live Cloudron E2E scripts require `FASTPACK_E2E_*` environment variables and skip when that private test environment is not configured.
