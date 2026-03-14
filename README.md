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

## Development

No build step required. Serve the directory with any static file server:

```bash
cd fastpack-cloudron
python3 -m http.server 8080
```

Open `http://localhost:8080` in your browser.

Run tests: open `http://localhost:8080/test.html` (47 unit tests).

## License

MIT
