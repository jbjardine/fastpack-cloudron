# Getting Started

Generate your first Cloudron custom app package in 3 minutes.

## What You Need

- A Docker image you want to deploy (e.g., `nginx:latest`, `ghost:5`, `n8nio/n8n`)
- A Cloudron server ([cloudron.io](https://cloudron.io))
- The Cloudron CLI (`npm install -g cloudron`)

## Step 1: Open FastPackCloudron

Go to [jbjardine.github.io/fastpack-cloudron](https://jbjardine.github.io/fastpack-cloudron/)

## Step 2: Enter Your Docker Image

Type the image name in the "Docker Image" field. FastPackCloudron auto-generates:
- **App ID**: `io.fastpack.nginx` (from the image name)
- **Title**: `Nginx` (humanized image name)
- **Version**: `1.0.0`

## Step 3: Pick Your Options

| Option | What It Does |
|--------|-------------|
| **Database** | Adds PostgreSQL, MySQL, MongoDB, or Redis as a Cloudron addon |
| **SSO** | Adds ProxyAuth, OIDC, LDAP, OAuth, or SimpleAuth |
| **Stack** | Pre-fills the start.sh with the right command (Node.js, PHP, Python, Java, Go) |
| **Web Interface** | Yes = single-process app. No = background service with healthcheck |

## Step 4: Download ZIP

Click "Download ZIP". You get a ready-to-deploy package:

```
my-app-cloudron.zip
├── CloudronManifest.json    # App metadata, addons, ports
├── Dockerfile               # Container definition
├── start.sh                 # Init script with gosu, signals
├── .dockerignore
├── README.md                # Deployment instructions
├── POSTINSTALL.md           # Post-install checklist
├── CHANGELOG.md             # Version history stub
├── CloudronVersions.json    # For App Store publishing
├── deploy.js                # Cross-platform deploy script
└── deploy.cmd               # Windows launcher
```

## Step 5: Deploy

```bash
# Extract the ZIP
unzip my-app-cloudron.zip -d my-app
cd my-app
```

### Option A: FastPack Deploy CLI (recommended — no dependencies)

Download the [Go binary](../deploy-cli/README.md) for your platform, place it in the extracted folder, and run it:

```bash
./fastpack-deploy-linux-amd64     # Linux
./fastpack-deploy-darwin-arm64    # macOS Apple Silicon
.\fastpack-deploy-windows-amd64.exe  # Windows
```

The wizard guides you through 4 steps: Cloudron URL, API token, subdomain, and Build Service. See the [full tutorial](tutorial-deploy-from-scratch.md) for detailed instructions.

### Option B: Cloudron CLI

```bash
npm install -g cloudron
cloudron login my.cloudron.com
cloudron build
cloudron install
```

### Option C: Deploy script

```bash
node deploy.js
```

## What Happens Behind the Scenes

1. **Dockerfile** wraps your image with Cloudron conventions:
   - Creates a `cloudron` user (uid 808) for non-root execution
   - Installs `gosu` (or `su-exec` on Alpine) for privilege dropping
   - Sets up signal handling for graceful shutdowns

2. **start.sh** initializes the app:
   - Creates `/app/data/.initialized` guard file (prevents re-init on updates)
   - Runs `chown` on localstorage directory
   - Executes your app via `gosu cloudron:cloudron`

3. **CloudronManifest.json** tells Cloudron:
   - Which port to expose
   - Which addons to provision (database, SSO, etc.)
   - Health check path
   - Memory limits

## Next Steps

- See [Configuration Examples](examples.md) for real-world setups
- Read about [DooD Mode](dood-guide.md) for multi-image apps
- Check the [Architecture Guide](architecture.md) for deep technical details
