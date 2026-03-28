# Tutorial: Deploy Any Docker Image to Cloudron (From Scratch)

A step-by-step guide for complete beginners. No command line experience required.

## What You Will Build

By the end of this tutorial, you will have:
- A custom app running on your Cloudron server
- Accessible at `https://myapp.your-domain.com`
- Built from any public Docker image

**Time required**: 10-15 minutes

## What You Need

| Requirement | How to Get It |
|------------|---------------|
| A Cloudron server | [cloudron.io](https://cloudron.io) - free trial available |
| The Docker Builder app | Install from Cloudron App Store (search "Docker Builder") |
| A web browser | Chrome, Firefox, Edge, or Safari |

## Step 1: Install the Docker Builder on Your Cloudron

The Docker Builder is a Cloudron app that builds Docker images on your server. You need it to deploy custom apps.

1. Open your Cloudron dashboard: `https://my.your-domain.com`
2. Go to **App Store** (left sidebar)
3. Search for **"Docker Builder"** (or "Build Service")
4. Click **Install**
5. Choose a subdomain like `devtools` (it will be at `devtools.your-domain.com`)
6. Wait for installation to complete

## Step 2: Get Your Tokens

You need two tokens: one for the Cloudron API, and one for the Docker Builder.

### Cloudron API Token

1. Go to your Cloudron dashboard: `https://my.your-domain.com`
2. Click your **profile icon** (top-right)
3. Go to **API Access**
4. Click **Create Token**
5. Copy the token and save it somewhere safe

### Build Service Token

1. Open the Docker Builder app: `https://devtools.your-domain.com`
2. Log in with your Cloudron account (SSO)
3. On the **Setup** page, you will see a token
4. Copy the token and save it somewhere safe

## Step 3: Create Your App Package with FastPackCloudron

1. Go to [FastPackCloudron](https://jbjardine.github.io/fastpack-cloudron/)
2. In the **Docker Image** field, type: `nginx:latest`
   - This is the simplest example — a web server
   - You can use any public Docker image here
3. Leave all other options as defaults
4. Click **Download ZIP**
5. A deploy wizard will appear — keep it open

## Step 4: Extract the ZIP

### On Windows
1. Find the downloaded ZIP file (usually in your Downloads folder)
2. Right-click the ZIP file
3. Select **Extract All...**
4. Click **Extract**
5. Open the extracted folder

### On macOS
1. Double-click the downloaded ZIP file
2. It extracts automatically
3. Open the extracted folder

### On Linux
```bash
unzip my-app-cloudron.zip -d my-app
cd my-app
```

## Step 5: Download the Deploy Binary

The deploy wizard (from Step 3) shows which binary to download for your operating system.

Download the correct binary from the [latest release](https://github.com/jbjardine/fastpack-cloudron/releases):

| Your OS | Binary to Download |
|---------|-------------------|
| Windows | `fastpack-deploy-windows-amd64.exe` |
| macOS (Apple Silicon) | `fastpack-deploy-darwin-arm64` |
| macOS (Intel) | `fastpack-deploy-darwin-amd64` |
| Linux | `fastpack-deploy-linux-amd64` |

**Place the binary inside the extracted folder** (next to `CloudronManifest.json`).

## Step 6: Run the Deploy Binary

### On Windows

1. Open a **Command Prompt** (search "cmd" in Start menu)
2. Navigate to the extracted folder:
   ```
   cd C:\Users\YOU\Downloads\my-app
   ```
3. Run the binary:
   ```
   .\fastpack-deploy-windows-amd64.exe
   ```

### On macOS / Linux

1. Open a **Terminal**
2. Navigate to the extracted folder:
   ```bash
   cd ~/Downloads/my-app
   ```
3. Make the binary executable (macOS/Linux only):
   ```bash
   chmod +x ./fastpack-deploy-*
   ```
4. Run it:
   ```bash
   ./fastpack-deploy-darwin-arm64    # macOS Apple Silicon
   ./fastpack-deploy-darwin-amd64    # macOS Intel
   ./fastpack-deploy-linux-amd64     # Linux
   ```

## Step 7: Follow the Wizard

The deploy binary will ask you 4 questions:

```
Step 1/4: Enter your Cloudron URL
   URL: my.your-domain.com

Step 2/4: Enter your API token
   Token: (paste the Cloudron API token from Step 2)

Step 3/4: Choose a subdomain for your app
   Subdomain: myapp

Step 4/4: Build Service
   Build Service URL: devtools.your-domain.com
   Build Service Token: (paste the Build Service token from Step 2)
```

Then sit back and watch:

```
Connecting to Cloudron... OK (My Cloudron v9.1.5)
Verifying Build Service... OK
Packaging files... OK
Building on Cloudron (this may take a few minutes)... ....OK
Installing app at myapp.your-domain.com... OK

  App deployed!
  https://myapp.your-domain.com
```

## Step 8: Visit Your App

Open `https://myapp.your-domain.com` in your browser. You should see the Nginx welcome page!

## Troubleshooting

### "CloudronManifest.json not found"

You need to run the binary from inside the extracted ZIP folder. Make sure you see `CloudronManifest.json` when you list files in the current directory.

### "Cannot connect: invalid API token"

Your Cloudron API token is incorrect or expired.
1. Go to your Cloudron dashboard > Profile > API Access
2. Create a new token
3. Try again

### "Build Service auth failed (HTTP 401)"

Your Build Service token is incorrect or expired.
1. Open the Docker Builder app (devtools.your-domain.com)
2. Log in and go to the Setup page
3. Copy the new token
4. Try again

### "build timed out after 10 minutes"

The Docker image is large and takes time to build. This is normal for the first build. Try again — subsequent builds use caching and are faster.

### "subdomain already in use (HTTP 409)"

That subdomain is already taken on your Cloudron. Choose a different one (e.g., `myapp2` instead of `myapp`).

## What's Next?

- Try deploying other Docker images: `ghost:5`, `gitea/gitea`, `n8nio/n8n`
- Add a database: check the PostgreSQL/MySQL options in FastPackCloudron
- Add SSO: enable OIDC to use Cloudron user authentication
- See [Custom App Example](custom-app-example.md) for a more advanced setup
- See [Configuration Examples](examples.md) for 15+ real-world configurations
