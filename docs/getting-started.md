# Getting Started

Generate and deploy your first Cloudron custom app package.

## What You Need

- A Docker image to package, such as `nginx:latest`, `ghost:5`, or `n8nio/n8n`
- A Cloudron server
- The FastPack Deploy CLI binary for your platform, or the official Cloudron CLI

## 1. Generate the Package

1. Open [FastPackCloudron](https://jbjardine.github.io/fastpack-cloudron/).
2. Enter a Docker image.
3. Pick only the options you need: database, SSO, ports, storage, or stack template.
4. Click **Download ZIP**.

The ZIP contains:

```text
CloudronManifest.json
Dockerfile
start.sh
.dockerignore
README.md
POSTINSTALL.md
CHANGELOG.md
CloudronVersions.json
```

Review `start.sh` before deploying. Some images need a custom start command.

## 2. Deploy With FastPack Deploy CLI

Extract the ZIP, place the FastPack Deploy CLI binary in that folder, then run it:

```bash
unzip my-app-cloudron.zip -d my-app
cd my-app
./fastpack-deploy-linux-amd64
```

The wizard asks for:

1. Cloudron URL
2. Cloudron username
3. Cloudron password, plus 2FA code if enabled
4. App subdomain

The CLI uploads a source archive to Cloudron. No local Docker, Docker Registry, or Build Service is required.

## 3. Alternative: Cloudron CLI

```bash
npm install -g cloudron
cloudron login my.example.com
cloudron build
cloudron install
```

Use this path when you want full manual control over the Cloudron build/install flow.

## Development Cloudrons

For self-signed development instances, set `allowSelfSigned` in `fastpack-deploy.json`. Keep TLS verification enabled for production.

## Next Steps

- [Configuration Examples](examples.md)
- [Custom App Example](custom-app-example.md)
- [Architecture](architecture.md)
