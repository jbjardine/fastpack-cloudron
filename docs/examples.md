# Configuration Examples

Real-world examples from simple to advanced. Each shows what to select in the UI and what gets generated.

---

## 1. Static Website (Simplest)

**Use case**: Serve a static site with nginx.

| Field | Value |
|-------|-------|
| Docker Image | `nginx:alpine` |
| Database | None |
| SSO | None |
| Web Interface | Yes |
| Stack | Generic |

**What you get**:
```dockerfile
FROM nginx:alpine
# ... cloudron user, su-exec (Alpine), start.sh
EXPOSE 8000
CMD ["/app/code/start.sh"]
```

**start.sh**: `exec /usr/local/bin/gosu cloudron:cloudron YOUR_APP_COMMAND`

**You replace** `YOUR_APP_COMMAND` with: `nginx -g 'daemon off;'`

---

## 2. Node.js App with PostgreSQL

**Use case**: A Next.js or Express app with a database.

| Field | Value |
|-------|-------|
| Docker Image | `node:20-slim` |
| Database | PostgreSQL |
| SSO | None |
| Stack | Node.js |
| Addons | localstorage (checked) |

**What you get**:

```json
// CloudronManifest.json
{
  "addons": {
    "postgresql": {},
    "localstorage": {}
  }
}
```

**start.sh** (auto-filled by Node.js stack template):
```bash
#!/bin/sh
set -eu

if [ ! -f /app/data/.initialized ]; then
    echo "Initializing data directory..."
    echo "1.0.0" > /app/data/.initialized
fi
chown -R cloudron:cloudron /app/data

# Adjust path to your Node.js entry point
exec /usr/local/bin/gosu cloudron:cloudron node /app/code/server.js
```

**Environment variable** available at runtime: `CLOUDRON_POSTGRESQL_URL`

---

## 3. Python FastAPI with OIDC SSO

**Use case**: A Python API with Cloudron single sign-on.

| Field | Value |
|-------|-------|
| Docker Image | `python:3-slim` |
| Database | None |
| SSO | OIDC |
| Stack | Python |
| OIDC Redirect URI | `/auth/callback` |
| Addons | localstorage |

**What you get**:

```json
{
  "addons": {
    "oidc": {
      "loginRedirectUri": "/auth/callback",
      "logoutRedirectUri": "/"
    },
    "localstorage": {}
  }
}
```

**Environment variables**: `CLOUDRON_OIDC_IDENTIFIER`, `CLOUDRON_OIDC_SECRET`, `CLOUDRON_OIDC_ISSUER`

**start.sh**: `exec /usr/local/bin/gosu cloudron:cloudron python3 /app/code/app.py`

---

## 4. Java Spring Boot

**Use case**: A Java application with memory management.

| Field | Value |
|-------|-------|
| Docker Image | `eclipse-temurin:21-jre` |
| Database | MySQL |
| SSO | None |
| Stack | Java |
| Memory Limit | 512 MB |
| Addons | localstorage |

**start.sh** (Java template with MaxRAMPercentage):
```bash
# MaxRAMPercentage respects Cloudron memory limits
exec /usr/local/bin/gosu cloudron:cloudron java -XX:MaxRAMPercentage=75.0 -jar /app/code/app.jar
```

**Key**: The `-XX:MaxRAMPercentage=75.0` flag ensures the JVM respects the container memory limit set in the manifest.

---

## 5. Go Binary

**Use case**: A compiled Go web server.

| Field | Value |
|-------|-------|
| Docker Image | `golang:1.22-alpine` |
| Database | Redis |
| SSO | ProxyAuth |
| Stack | Go |
| ProxyAuth Path | `/api` (only protect /api) |

**start.sh**: `exec /usr/local/bin/gosu cloudron:cloudron /app/code/server`

**Manifest addons**:
```json
{
  "redis": {},
  "proxyAuth": { "path": "/api" },
  "localstorage": {}
}
```

ProxyAuth with a path means only `/api/*` requires authentication. The rest is public.

---

## 6. MQTT Broker (TCP-only, no web UI)

**Use case**: Eclipse Mosquitto MQTT broker.

| Field | Value |
|-------|-------|
| Docker Image | `eclipse-mosquitto:2.0` |
| Web Interface | **No** |
| TCP Ports | Name: `MQTT`, Port: 1883 |
| Addons | localstorage |

**What changes**: No web UI means:
- `python3` is installed for a minimal health-check server
- start.sh runs the broker in background + a Python HTTP server for Cloudron healthcheck
- The HTTP port returns "OK" so Cloudron knows the app is running

```bash
# start.sh for TCP-only mode
/usr/local/bin/gosu cloudron:cloudron YOUR_SERVICE_COMMAND &

# Minimal health check (returns 200 OK)
python3 -c "
import http.server
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'OK')
http.server.HTTPServer(('0.0.0.0', 8000), H).serve_forever()
" &

wait
```

---

## 7. WordPress + Redis (Multi-service)

**Use case**: WordPress with a Redis object cache sidecar.

| Field | Value |
|-------|-------|
| Docker Image | `wordpress:php8.2-apache` |
| Database | MySQL |
| Stack | Apache + PHP |
| **Services** | |
| Service 1 | Name: `wordpress`, Command: `apache2-foreground`, Port: 8080, Route: `/` |
| Service 2 | Name: `redis`, Command: `redis-server --port 6379`, Port: 6379, Route: (none) |

**What happens**: FastPackCloudron generates:
- **nginx.conf**: Routes `/` to WordPress on port 8080
- **start.sh**: Launches both services in background with signal forwarding
- **Dockerfile**: Installs nginx for the reverse proxy

```bash
# start.sh multi-service mode
trap 'kill $(jobs -p) 2>/dev/null; wait' TERM INT

echo "Starting wordpress..."
/usr/local/bin/gosu cloudron:cloudron apache2-foreground &

echo "Starting redis..."
/usr/local/bin/gosu cloudron:cloudron redis-server --port 6379 &

nginx -c /app/code/nginx.conf -g "daemon off;" &

wait
```

---

## 8. Multi-Stage Build (COPY --from=)

**Use case**: Your app needs a binary from another Docker image.

| Field | Value |
|-------|-------|
| Docker Image | `python:3-slim` |
| **Copy from image** | |
| Source Image | `node:20-slim` |
| Source Path | `/usr/local/bin/node` |
| Dest Path | `/usr/local/bin/node` |

**Generated Dockerfile**:
```dockerfile
FROM python:3-slim

# Copy files from external images
COPY --from=node:20-slim /usr/local/bin/node /usr/local/bin/node

# ... rest of Cloudron setup
```

**Warning**: If you copy from an Alpine image into a Debian image (or vice versa), compiled binaries won't work due to musl/glibc incompatibility. FastPackCloudron warns you about this.

---

## 9. DooD Mode (Sub-containers)

**Use case**: Run n8n and httpbin as separate Docker containers inside one Cloudron app.

| Field | Value |
|-------|-------|
| Docker Image | `python:3-slim` (base image for the orchestrator) |
| **Sub-containers** | |
| Sub 1 | Image: `kennethreitz/httpbin`, Port: 80, Route: `/api`, Memory: 256 MB |
| Sub 2 | Image: `n8nio/n8n`, Port: 5678, Route: `/n8n`, Memory: 512 MB |

**How DooD works**:

1. The main container acts as an **orchestrator** with nginx + Docker CLI
2. `start.sh` runs `docker run` for each sub-container image
3. Cloudron auto-places sub-containers on the `cloudron` internal network
4. The orchestrator discovers sub-container IPs via `docker inspect`
5. nginx proxies to each sub-container by IP

```
Cloudron App (orchestrator)
├── nginx (port 8000)
│   ├── /api/  → httpbin container (172.18.0.2:80)
│   └── /n8n/  → n8n container (172.18.0.3:5678)
└── Docker CLI → manages sub-containers
```

**Data**: Each sub-container mounts a volume in `/app/data/` so Cloudron backs up everything.

**Requirements**: The `docker` addon is automatically added to the manifest.

See the full [DooD Guide](dood-guide.md) for details.

---

## 10. Publishing-Ready App

**Use case**: Prepare an app for the Cloudron App Store.

| Field | Value |
|-------|-------|
| Docker Image | `ghost:5-alpine` |
| Stack | Node.js |
| **Metadata** | |
| Author | Your Name |
| Tagline | Modern publishing platform |
| Description | (markdown description) |
| Website | https://ghost.org |
| Icon | file://icon.png |
| Post-Install Checklist | Change admin password, Configure email |
| **Publishing** | |
| Docker Hub Username | yourusername |
| Packager Name | Your Name |
| Documentation URL | https://ghost.org/docs |

**Extra files generated**:
- `POSTINSTALL.md` with checklist items
- `CHANGELOG.md` stub
- `CloudronVersions.json` with Docker Hub image URL
- `DESCRIPTION.md` from your description

**Checklist in manifest** (Cloudron shows this after install):
```json
{
  "checklist": {
    "change_admin_password": { "message": "Change admin password" },
    "configure_email": { "message": "Configure email" }
  }
}
```

---

## 11. Full-Domain App

**Use case**: An app that needs the bare domain (no subdomain).

| Field | Value |
|-------|-------|
| Docker Image | `wordpress:latest` |
| **Advanced** | |
| Full domain | Checked |
| Secondary Subdomains | `www, api` |

**Manifest**:
```json
{
  "fullDomain": true,
  "secondarySubdomains": ["www", "api"]
}
```

The app is deployed at `example.com` instead of `app.example.com`.

---

## 12. App with Custom Backup

**Use case**: An app that needs custom backup/restore commands.

| Field | Value |
|-------|-------|
| Docker Image | `mysql:8` |
| **Advanced** | |
| Backup Command | `/app/code/backup.sh` |
| Restore Command | `/app/code/restore.sh` |
| Persistent Dirs | `data, config` |
| Runtime Dirs | `tmp, cache` |

**Manifest**:
```json
{
  "backupCommand": "/app/code/backup.sh",
  "restoreCommand": "/app/code/restore.sh",
  "persistentDirs": ["data", "config"],
  "runtimeDirs": ["tmp", "cache"]
}
```

---

## 13. Fedora-Based Image

**Use case**: An app based on Fedora/RHEL.

| Field | Value |
|-------|-------|
| Docker Image | `fedora:39` |

**What changes**: FastPackCloudron detects the Fedora base and:
- Uses `dnf` instead of `apt-get`/`apk`
- Installs `util-linux` for `setpriv` (gosu equivalent)
- Creates a shell wrapper at `/usr/local/bin/gosu`
- Shows a warning: "This image uses dnf. gosu will be installed via util-linux/setpriv."

---

## 14. App with Extra HTTP Ports

**Use case**: MinIO with separate API and Console ports.

| Field | Value |
|-------|-------|
| Docker Image | `minio/minio` |
| **Extra HTTP Ports** | |
| Port 1 | Name: `API`, Port: 9000 |

**Manifest**:
```json
{
  "httpPort": 8000,
  "httpPorts": {
    "API": {
      "title": "API Port",
      "description": "API Port port",
      "containerPort": 9000,
      "defaultValue": 9000
    }
  }
}
```

Cloudron assigns a separate subdomain for each extra HTTP port.

---

## 15. Dockerfile.cloudron Naming

**Use case**: Your project already has a `Dockerfile`.

| Field | Value |
|-------|-------|
| **Advanced** | |
| Dockerfile.cloudron | Checked |

The generated file is named `Dockerfile.cloudron` instead of `Dockerfile`, so it doesn't conflict with your existing Dockerfile. Use `cloudron build --file Dockerfile.cloudron` to build.

---

## Tips

- **Start simple**: Get a basic config working first, then add addons
- **Check the preview**: All generated files are shown live before you download
- **LocalStorage**: Almost always enable this. Without it, your app loses data on updates
- **Dark mode**: The UI respects your system preference
- **Form persistence**: Your configuration is saved in localStorage and restored on reload
