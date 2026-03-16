# Builder + Registry (combined Cloudron app)

A single Cloudron app that runs both the **Docker Build Service** and a **Container Registry**. Uses only 1 app slot instead of 2.

## Architecture

```
nginx (port 8000)
├── /v2/*  → Docker Registry (port 5000)
└── /*     → Build Service (port 3000)
```

## Install

```bash
cloudron build
cloudron install
```

## Setup

1. Open the app in your browser and log in (Cloudron SSO)
2. Copy the **build token** from the Setup page
3. Configure the CLI:

```bash
cloudron build login --build-service-url https://YOUR-APP-DOMAIN --build-token YOUR_TOKEN
```

4. Configure the builder to push to its own registry. Via Cloudron admin, exec into the app and edit `/app/data/docker.json`:

```json
{
  "YOUR-APP-DOMAIN": {
    "username": "",
    "password": ""
  }
}
```

5. If using self-signed certificates, add the app domain as an insecure registry in `/etc/docker/daemon.json` on the Cloudron host:

```json
{
  "insecure-registries": ["YOUR-APP-DOMAIN"]
}
```

Then restart Docker: `sudo systemctl restart docker`

## Usage

From any FastPackCloudron package directory:

```bash
cloudron build --repository YOUR-APP-DOMAIN/my-app
cloudron install
```

Or use the included `deploy.js`:

```bash
node deploy.js          # first install
node deploy.js app.domain.com  # update
```

## Requirements

- Cloudron with the `docker` addon enabled
- At least 1 GiB memory allocated to the app
