# DooD Mode Guide

Run multiple Docker images in a single Cloudron custom app, each in its own isolated container.

## What Is DooD?

DooD (Docker-out-of-Docker) means your Cloudron app launches **separate Docker containers** via Cloudron's Docker addon. Each sub-container runs its own image with its own dependencies.

```
Cloudron App (orchestrator)
│
├── nginx (port 8000, reverse proxy)
│   ├── /n8n/     → n8n container (image: n8nio/n8n)
│   ├── /api/     → FastAPI container (image: tiangolo/uvicorn-gunicorn-fastapi)
│   └── /grafana/ → Grafana container (image: grafana/grafana)
│
├── Docker CLI    → manages sub-containers lifecycle
└── /app/data/    → all sub-container data (backed up by Cloudron)
```

## Why DooD Instead of Multi-Service?

| | Multi-Service (in one container) | DooD (sub-containers) |
|---|---|---|
| Dependencies | Shared. Risk of conflicts | Isolated. Each image has its own |
| Updates | Rebuild everything | `docker pull` one image |
| Complexity | Simpler setup | More moving parts |
| When to use | Same project (frontend + API) | Different projects (n8n + FastAPI) |

## How to Use

### In FastPackCloudron

1. Open the **Sub-containers (DooD)** section
2. Click **Add sub-container**
3. For each sub-container, enter:
   - **Docker image**: e.g., `n8nio/n8n`
   - **Port**: the port the image listens on (e.g., `5678`)
   - **Route**: the URL path (e.g., `/n8n`)
   - **Memory**: memory limit in MB (e.g., `512`)
   - **Volume**: data path inside the container (e.g., `/home/node/.n8n`)
4. Download the ZIP and deploy

### What Gets Generated

**Dockerfile**: Installs Docker CLI (static binary) + nginx + your base image

```dockerfile
FROM python:3-slim

# Docker CLI for DooD sub-containers
ADD https://download.docker.com/linux/static/stable/x86_64/docker-27.5.1.tgz /tmp/docker.tgz
RUN tar -xzf /tmp/docker.tgz ... && mv docker /usr/local/bin/docker

# nginx for reverse proxy
RUN apt-get install -y nginx
```

**start.sh**: Launches sub-containers, discovers IPs, configures nginx

```bash
export DOCKER_HOST="${CLOUDRON_DOCKER_HOST}"

# Cleanup previous containers
docker rm -f fp-n8nio-n8n 2>/dev/null || true

# Launch n8n in its own container
docker run -d --name fp-n8nio-n8n \
  --memory=512m \
  --restart=unless-stopped \
  --label=fastpack.owner=${HOSTNAME} \
  -v /app/data/fp-n8nio-n8n:/home/node/.n8n \
  n8nio/n8n

# Discover IP on cloudron internal network
N8N_IP=$(docker inspect fp-n8nio-n8n \
  --format '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}')

# Inject IP into nginx config
sed "s/FP_N8NIO_N8N_IP/${N8N_IP}/g" nginx.conf.template > /tmp/nginx.conf

# Start nginx
nginx -c /tmp/nginx.conf -g "daemon off;" &
```

**nginx.conf** (template with IP placeholders):

```nginx
location /n8n/ {
    proxy_pass http://FP_N8NIO_N8N_IP:5678/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

**Manifest**: Automatically includes the `docker` addon

```json
{
  "addons": {
    "docker": {},
    "localstorage": {}
  }
}
```

## How It Works (Technical Details)

### Network

Cloudron automatically places sub-containers on the `cloudron` internal Docker network. The orchestrator container is also on this network. They communicate via internal IPs.

```
cloudron network (172.18.0.0/16)
├── orchestrator: 172.18.16.178
├── fp-n8nio-n8n: 172.18.0.2
└── fp-httpbin:   172.18.0.3
```

The orchestrator discovers IPs via `docker inspect` at startup.

### Data Persistence

Sub-container volumes are mounted under `/app/data/`:

```
/app/data/
├── fp-n8nio-n8n/       → n8n data
├── fp-httpbin/         → httpbin data
└── .initialized        → version guard
```

Cloudron backs up `/app/data/` on every snapshot, so **all sub-container data is backed up**.

### Lifecycle

| Event | What Happens |
|-------|-------------|
| App start | `start.sh` launches sub-containers, discovers IPs |
| App stop (SIGTERM) | `trap` handler stops and removes sub-containers |
| App uninstall | Cloudron removes tracked sub-containers |
| App update | Sub-containers are recreated with latest config |

### Resource Limits

Each sub-container has a memory limit (`--memory=256m`). This prevents one container from consuming all host memory.

## Limitations

- **Image pull at startup**: The first start may be slow if images need to be downloaded
- **No DNS resolution**: Sub-containers are reached by IP, not by name
- **Cloudron app limit**: Each DooD app counts as one app toward your Cloudron plan limit
- **No independent SSO**: Sub-containers don't get individual Cloudron SSO. The orchestrator handles auth
- **Resource visibility**: Sub-container CPU/memory usage is not shown in the Cloudron dashboard

## Best Practices

1. **Set memory limits**: Always specify a memory limit for each sub-container
2. **Use /app/data/**: Mount all persistent data under `/app/data/` for Cloudron backup
3. **Name your containers**: Use descriptive names (the tool auto-generates `fp-<image-name>`)
4. **Test locally first**: Build and test with `docker build && cloudron install` before publishing
5. **Consider separate apps**: For truly independent services, separate Cloudron apps are more robust

## Example: n8n + Metabase

```
Sub-container 1:
  Image: n8nio/n8n
  Port: 5678
  Route: /n8n
  Memory: 512 MB
  Volume: /home/node/.n8n

Sub-container 2:
  Image: metabase/metabase
  Port: 3000
  Route: /metabase
  Memory: 1024 MB
  Volume: /metabase-data
```

This creates a single Cloudron app at `myapp.example.com` with:
- `myapp.example.com/n8n/` - n8n workflow automation
- `myapp.example.com/metabase/` - Metabase analytics dashboard

Both share the same Cloudron domain, authentication proxy, and backup schedule.
