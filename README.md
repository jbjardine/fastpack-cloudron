# FastPackCloudron

Generate ready-to-deploy [Cloudron](https://cloudron.io) custom app packages in 3 clicks.

**[Try it live](https://jbjardine.github.io/fastpack-cloudron/)**

## What is this?

Packaging a Docker image as a Cloudron custom app requires writing several files that follow specific conventions (read-only filesystem, healthcheck, signal handling, addons...). FastPackCloudron generates all of them from a simple form:

- `CloudronManifest.json` — app metadata, addons, ports
- `Dockerfile` — container definition wrapping your source image
- `start.sh` — init script with proper patterns (localstorage, healthcheck, signals)
- `.dockerignore`
- `README.md` — deployment instructions

## Usage

1. Open [FastPackCloudron](https://jbjardine.github.io/fastpack-cloudron/)
2. Paste a Docker image name (e.g., `eclipse-mosquitto:2.0`)
3. Pick your options (database, SSO, ports)
4. Download the ZIP
5. Extract and deploy:

### Option A: FastPack Deploy CLI (recommended)

Download the [Go binary](deploy-cli/README.md) for your platform (Windows/macOS/Linux) and run it from the extracted folder:

```bash
./fastpack-deploy-linux-amd64
# Wizard guides you: Cloudron URL → Token → Subdomain → Build Service → Done!
```

No Node.js, no npm, no Docker on your machine — just a single binary.

### Option B: Cloudron CLI

```bash
cloudron login my.example.com
cloudron build
cloudron install
```

## Features

- **Smart defaults** — app ID, title, and version auto-generated from image name
- **Database addons** — PostgreSQL, MySQL, MongoDB, Redis
- **SSO** — ProxyAuth, OIDC, LDAP, or no SSO
- **System addons** — localstorage, sendmail, recvmail, scheduler, TLS, TURN, Docker
- **TCP/UDP ports** — dynamic port configuration
- **Web vs TCP mode** — automatic Python healthcheck for non-web services
- **Live preview** — see generated files before downloading
- **Progressive disclosure** — simple 3-click path with collapsible advanced sections
- **Zero backend** — everything runs client-side, no data sent anywhere

## Documentation

- [Getting Started](docs/getting-started.md) — First package in 3 minutes
- [Tutorial: Deploy From Scratch](docs/tutorial-deploy-from-scratch.md) — Complete beginner guide
- [Custom App Example](docs/custom-app-example.md) — Build and deploy a Flask app
- [Configuration Examples](docs/examples.md) — 15+ real-world setups
- [Architecture](docs/architecture.md) — Technical deep-dive
- [Go Deploy CLI](deploy-cli/README.md) — Zero-dependency deployer binary

## Development

No build step required. Serve the directory with any static file server:

```bash
cd fastpack-cloudron
python3 -m http.server 8080
```

Open `http://localhost:8080` in your browser.

### Tests

```bash
# Frontend unit tests (207 assertions)
node test-ci.mjs

# Go CLI unit tests (75 tests)
cd deploy-cli && go test ./...

# Docker build tests (11 configs)
node test-build.mjs

# Full E2E (requires Cloudron VM + Build Service token)
CLOUDRON_BUILD_TOKEN=<token> node test-go-deploy-e2e.mjs
```

## Support

[![Fuel my projects](https://img.shields.io/badge/%E2%9A%A1%20Fuel%20my%20projects%20%E2%98%95-111827?style=for-the-badge&labelColor=FFDD00&color=111827)](https://www.buymeacoffee.com/jbjardine)

## License

MIT
