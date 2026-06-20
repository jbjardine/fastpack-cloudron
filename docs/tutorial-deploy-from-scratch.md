# Tutorial: Deploy a Docker Image to Cloudron

This tutorial deploys `nginx:latest` as a Cloudron custom app using FastPackCloudron and the FastPack Deploy CLI.

## What You Need

- A Cloudron server
- A Cloudron admin account
- The FastPack Deploy CLI binary for your platform
- A browser

## 1. Create the Package

1. Open [FastPackCloudron](https://jbjardine.github.io/fastpack-cloudron/).
2. Enter `nginx:latest` in **Docker Image**.
3. Leave the default options.
4. Click **Download ZIP**.

## 2. Prepare the Folder

Extract the ZIP and place the deploy binary next to `CloudronManifest.json`.

```bash
unzip nginx-cloudron.zip -d nginx-cloudron
cd nginx-cloudron
```

Download binaries from the [latest release](https://github.com/jbjardine/fastpack-cloudron/releases/latest):

| Platform | Binary |
|----------|--------|
| Windows | `fastpack-deploy-windows-amd64.exe` |
| macOS Apple Silicon | `fastpack-deploy-darwin-arm64` |
| macOS Intel | `fastpack-deploy-darwin-amd64` |
| Linux | `fastpack-deploy-linux-amd64` |

On macOS or Linux:

```bash
chmod +x ./fastpack-deploy-*
```

## 3. Deploy

Run the binary from the extracted folder:

```bash
./fastpack-deploy-linux-amd64
```

The wizard asks for your Cloudron URL, username, password, and app subdomain. If your account uses 2FA, it also asks for the current TOTP code.

When deployment finishes, open the URL printed by the CLI.

## Troubleshooting

### `CloudronManifest.json not found`

Run the binary from inside the extracted package folder.

### Login failed

Check the Cloudron URL, username, password, and 2FA code.

### TLS warning on a development server

Self-signed certificates are only for development. Use `allowSelfSigned` deliberately in `fastpack-deploy.json`; keep TLS verification enabled in production.

### App starts but does not respond

Review `start.sh`. Some Docker images require a custom start command or a different HTTP port.

## More

- [Getting Started](getting-started.md)
- [Configuration Examples](examples.md)
- [Custom App Example](custom-app-example.md)
