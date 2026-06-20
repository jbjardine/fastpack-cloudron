# FastPackCloudron Spec Coverage Expansion Roadmap

## Current State

| Area                | Coverage | After Roadmap |
|---------------------|----------|---------------|
| Manifest fields     | ~85%     | ~98%          |
| Addon types         | ~87%     | ~100%         |
| Dockerfile patterns | ~80%     | ~95%          |
| start.sh patterns   | ~75%     | ~92%          |
| Publishing workflow | ~60%     | ~90%          |

## Prioritized Phases

Items are ordered by: (1) community demand, (2) publishing requirements, (3) spec completeness.

---

## Phase 1 -- Critical for Publishing and High Community Demand

### 1.1 httpPorts -- Extra HTTP services with separate subdomains

**Importance:** Critical for publishing / High community demand
**Complexity:** L (Large)

Apps like MinIO (API port + Console port), Gitea (web + SSH), or Matrix Synapse need multiple HTTP endpoints each with their own subdomain. This is the single most requested missing feature based on the Cloudron forum.

**Manifest spec:**
```json
{
  "httpPorts": {
    "CONSOLE_PORT": {
      "title": "Console",
      "defaultValue": 9001,
      "containerPort": 9001
    }
  }
}
```

**UI changes (index.html):**
- Add a new `<details>` section "HTTP Ports (extra subdomains)" between "Customize ports" and the existing TCP/UDP ports section.
- Dynamic rows similar to the existing TCP port pattern: fields for name, title, containerPort, defaultValue.
- Add help text: "Each HTTP port gets its own subdomain (e.g., console.myapp.example.com)."

**Generator changes (generators.js):**
- `generateManifest()`: Add `httpPorts` object to manifest when `config.httpPorts` has entries. Format matches `tcpPorts` structure.
- `generateDockerfile()`: Add httpPort container ports to the EXPOSE line.
- `generateStartSh()`: No change needed (app handles its own ports).

**Validation rules (app.js):**
- Validate httpPort names follow `[A-Za-z_][A-Za-z0-9_]*` pattern.
- Validate containerPort is 1-65535 and does not conflict with the primary `httpPort` or any TCP/UDP ports.
- Warning if httpPorts is used without a web UI.

**Tests to add:**
- test.html: Manifest includes `httpPorts` when configured; omits when empty; port conflict detection.
- test-build.mjs: Docker build with httpPorts EXPOSE includes extra ports.
- test-cloudron.mjs: MinIO-style two-port config (requires subdomain routing -- may be manual verification).

---

### 1.2 checklist -- Post-install security checklist

**Importance:** Critical for publishing (required for App Store submission)
**Complexity:** S (Small)

The Cloudron App Store requires a `checklist` field for security-sensitive apps. This is a simple array of markdown strings shown to the admin after install.

**Manifest spec:**
```json
{
  "checklist": [
    "Change the default admin password at `/admin`",
    "Configure OAuth credentials under Settings > Authentication"
  ]
}
```

**UI changes (index.html):**
- Add a `<textarea>` in the "Publishing" `<details>` section, labeled "Post-Install Checklist (one item per line)".
- Placeholder: "Change default admin password\nReview security settings".

**Generator changes (generators.js):**
- `generateManifest()`: Parse newline-separated text into an array. Only emit `checklist` when at least one non-empty item exists.

**Validation rules (app.js):**
- Warning when publishing fields (packagerName, etc.) are set but checklist is empty: "App Store submissions typically require a security checklist."

**Tests to add:**
- test.html: Manifest includes `checklist` array; omits when empty; preserves multi-line items.

---

### 1.3 oauth addon -- Internal SSO via OAuth

**Importance:** High community demand
**Complexity:** M (Medium)

Many apps (Grafana, Gitea, Nextcloud) implement OAuth for login. The `oauth` addon provides client credentials from Cloudron's identity provider, distinct from `oidc` which is a newer spec.

**Manifest spec:**
```json
{
  "addons": {
    "oauth": {}
  }
}
```
Environment variables exposed: `CLOUDRON_OAUTH_CLIENT_ID`, `CLOUDRON_OAUTH_CLIENT_SECRET`, `CLOUDRON_OAUTH_ORIGIN`, `CLOUDRON_OAUTH_API_ORIGIN`.

**UI changes (index.html):**
- Add `oauth` to the SSO `<select>` dropdown alongside ProxyAuth, OIDC, LDAP.
- No sub-options needed (oauth addon has no config keys).

**Generator changes (generators.js):**
- `generateManifest()`: When `config.sso === "oauth"`, add `addons.oauth = {}` and omit `optionalSso`.
- `generateStartSh()`: Add comment block listing the `CLOUDRON_OAUTH_*` environment variables available.

**Validation rules (app.js):**
- No special validation beyond existing SSO logic.

**Tests to add:**
- test.html: Manifest includes `oauth` addon when selected; no `optionalSso` set; environment variable comments in start.sh.

---

### 1.4 simpleauth addon -- Single-user auth via HTTP request

**Importance:** High community demand
**Complexity:** M (Medium)

For apps that cannot integrate OIDC/OAuth but need more than ProxyAuth headers. The app calls `http://127.0.0.1:5050/api/v1/user?access_token=TOKEN` to verify the logged-in user.

**Manifest spec:**
```json
{
  "addons": {
    "simpleauth": {}
  }
}
```

**UI changes (index.html):**
- Add `simpleauth` to the SSO `<select>` dropdown.
- Add help text below: "App verifies users via HTTP call to Cloudron's simpleauth API."

**Generator changes (generators.js):**
- `generateManifest()`: When `config.sso === "simpleauth"`, add `addons.simpleauth = {}` and omit `optionalSso`.
- `generateStartSh()`: Add comment explaining the simpleauth API endpoint and usage pattern.

**Validation rules (app.js):**
- No special validation.

**Tests to add:**
- test.html: Manifest includes `simpleauth` addon; no `optionalSso`.

---

### 1.5 logPaths -- Custom log file paths

**Importance:** High community demand
**Complexity:** S (Small)

Apache, Java apps, and legacy software write logs to files instead of stdout. Cloudron can tail these if declared.

**Manifest spec:**
```json
{
  "logPaths": ["/var/log/apache2/error.log", "/app/data/logs/app.log"]
}
```

**UI changes (index.html):**
- Add a text input in the "Advanced" `<details>` section, labeled "Log Paths (comma-separated)".
- Placeholder: "/var/log/apache2/error.log, /app/data/logs/app.log".

**Generator changes (generators.js):**
- `generateManifest()`: Parse comma-separated paths into array. Only emit when non-empty.

**Validation rules (app.js):**
- Each path must start with `/`.
- Warning: "Logs to stdout/stderr are captured automatically. Only add logPaths for files that cannot be redirected."

**Tests to add:**
- test.html: Manifest includes `logPaths` array; omits when empty; validates paths start with `/`.

---

### 1.6 Stack-specific start.sh templates

**Importance:** High community demand
**Complexity:** L (Large)

The current start.sh is generic. Most Cloudron packaging questions on the forum involve Apache/PHP, Java, Go, or Node.js startup patterns. Providing stack-aware templates would dramatically reduce user friction.

**UI changes (index.html):**
- Add a `<select>` dropdown "Application Stack" after the Docker Image field.
- Options: `Generic (default)`, `Node.js`, `Apache/PHP`, `Java (JAR)`, `Go Binary`, `Python (gunicorn)`.
- Stack selection auto-adjusts the start.sh template and adds relevant comments.

**Generator changes (generators.js):**
- `generateStartSh()`: Branch on `config.stack` to emit stack-specific patterns:
  - **Node.js**: `exec gosu cloudron:cloudron node /app/code/server.js` with `NODE_ENV=production` and port binding.
  - **Apache/PHP**: `exec gosu cloudron:cloudron apache2-foreground` with config symlinks to `/run`, `DocumentRoot` setup, writable dirs.
  - **Java**: `exec gosu cloudron:cloudron java -jar /app/code/app.jar --server.port=$PORT` with `JAVA_OPTS`.
  - **Go**: `exec gosu cloudron:cloudron /app/code/server` with env-based config.
  - **Python**: `exec gosu cloudron:cloudron gunicorn --bind 0.0.0.0:$PORT app:app`.
- `generateDockerfile()`: Stack-specific additions:
  - Apache/PHP: Install apache2 + php packages, COPY vhost.conf.
  - Java: COPY app.jar.
  - Node.js: COPY package.json, npm install.

**Validation rules (app.js):**
- Warning when image name suggests a stack but a different stack is selected (e.g., image `php:8-apache` but stack is `Generic`).

**Tests to add:**
- test.html: Each stack template contains expected commands and comments.
- test-build.mjs: Build test for Node.js and Apache/PHP templates.

---

## Phase 2 -- Important for Completeness

### 2.1 secondarySubdomains -- Apps needing multiple subdomains

**Importance:** Nice to have (overlaps with httpPorts)
**Complexity:** S (Small)

Some apps need extra subdomains that all route to the same httpPort (unlike httpPorts which map to different container ports). Example: `www` + `api` + `cdn` subdomains.

**Manifest spec:**
```json
{
  "secondarySubdomains": ["www", "api"]
}
```

**UI changes (index.html):**
- Add a text input in the "Advanced" `<details>` section: "Secondary Subdomains (comma-separated)".
- Placeholder: "www, api".

**Generator changes (generators.js):**
- `generateManifest()`: Parse comma-separated values into array. Emit only when non-empty.

**Validation rules (app.js):**
- Each subdomain must be a valid DNS label (lowercase alphanumeric + hyphens, no leading/trailing hyphen).
- Warning when used together with httpPorts: "httpPorts already create separate subdomains."

**Tests to add:**
- test.html: Manifest includes `secondarySubdomains`; omits when empty; validates format.

---

### 2.2 fullDomain -- Bare domain deployment

**Importance:** Nice to have
**Complexity:** S (Small)

Allows the app to be installed on the bare domain (e.g., `example.com` instead of `app.example.com`).

**Manifest spec:**
```json
{
  "fullDomain": true
}
```

**UI changes (index.html):**
- Add a checkbox in the "Advanced" `<details>` section: "Allow bare domain deployment (fullDomain)".

**Generator changes (generators.js):**
- `generateManifest()`: Emit `fullDomain: true` when checked.

**Validation rules (app.js):**
- Warning: "fullDomain apps take over the entire domain. Only one app per domain is allowed."

**Tests to add:**
- test.html: Manifest includes `fullDomain: true` when checked; omits when unchecked.

---

### 2.3 singleUser -- Single-user app mode

**Importance:** Nice to have
**Complexity:** S (Small)

Marks the app as single-user (only one Cloudron user can access it).

**Manifest spec:**
```json
{
  "singleUser": true
}
```

**UI changes (index.html):**
- Add a checkbox in the "Advanced" `<details>` section: "Single-user app (singleUser)".

**Generator changes (generators.js):**
- `generateManifest()`: Emit `singleUser: true` when checked.

**Validation rules (app.js):**
- Warning when singleUser is enabled alongside LDAP: "LDAP implies multi-user. singleUser may conflict."

**Tests to add:**
- test.html: Manifest includes/omits `singleUser`.

---

### 2.4 maxBoxVersion / targetBoxVersion -- Version constraints

**Importance:** Nice to have
**Complexity:** S (Small)

Constrains which Cloudron versions the app can run on. `minBoxVersion` is already implemented; `maxBoxVersion` and `targetBoxVersion` complete the set.

**Manifest spec:**
```json
{
  "minBoxVersion": "7.0.0",
  "maxBoxVersion": "8.0.0",
  "targetBoxVersion": "7.6.0"
}
```

**UI changes (index.html):**
- Add two text inputs in the "Advanced" `<details>` section alongside the existing `minBoxVersion`:
  - "Maximum Cloudron Version" (maxBoxVersion)
  - "Target Cloudron Version" (targetBoxVersion)
- Placeholders: "e.g., 8.0.0" and "e.g., 7.6.0".

**Generator changes (generators.js):**
- `generateManifest()`: Emit `maxBoxVersion` and `targetBoxVersion` when non-empty.

**Validation rules (app.js):**
- Validate semver format (same as existing minBoxVersion check).
- Error when maxBoxVersion < minBoxVersion.
- Warning when targetBoxVersion is outside [minBoxVersion, maxBoxVersion] range.

**Tests to add:**
- test.html: Manifest includes/omits version constraints; validates range logic.

---

### 2.5 localstorage sub-options -- FTP and SQLite paths

**Importance:** Nice to have
**Complexity:** S (Small)

The `localstorage` addon supports `ftp` (with uid/uname) and `sqlite` sub-options for apps that need FTP access to their storage or use SQLite databases.

**Manifest spec:**
```json
{
  "addons": {
    "localstorage": {
      "ftp": { "uid": 808, "uname": "cloudron" },
      "sqlite": ["/app/data/db.sqlite"]
    }
  }
}
```

**UI changes (index.html):**
- When the `localstorage` checkbox is checked, show a sub-options panel (similar to sendmail options):
  - Checkbox: "Enable FTP access"
  - Text input: "SQLite database paths (comma-separated)", placeholder: "/app/data/db.sqlite".

**Generator changes (generators.js):**
- `generateManifest()`: When localstorage has sub-options, emit them nested under the `localstorage` addon key instead of `{}`.

**Validation rules (app.js):**
- SQLite paths must start with `/app/data/`.
- Warning: "FTP access exposes /app/data via SFTP. Ensure sensitive files are not in the data directory."

**Tests to add:**
- test.html: Manifest includes localstorage with `ftp` and `sqlite` sub-options; validates paths.

---

### 2.6 Symlinks to /run for dynamic config

**Importance:** Nice to have (but solves a common packaging pain point)
**Complexity:** S (Small)

Cloudron uses a read-only filesystem. Apps that write config files at startup need those paths symlinked to `/run` (tmpfs). This is a frequent source of "why does my app fail?" questions.

**UI changes (index.html):**
- Add a text input in the "Advanced" `<details>` section: "Symlink to /run (comma-separated paths, e.g., /etc/nginx/conf.d)".
- Placeholder: "/etc/app/config.ini, /var/cache/app".

**Generator changes (generators.js):**
- `generateDockerfile()`: For each path, add `RUN mkdir -p /run/$(basename path) && ln -sf /run/$(basename path) path`.
- `generateStartSh()`: Add comment explaining the read-only FS pattern.

**Validation rules (app.js):**
- Paths must be absolute (start with `/`).
- Warning: "These paths will be empty on each container start. Your start.sh must populate them."

**Tests to add:**
- test.html: Dockerfile includes symlink commands; empty when no paths specified.
- test-build.mjs: Build with symlinks succeeds.

---

## Phase 3 -- Publishing Workflow and Polish

### 3.1 POSTINSTALL.md / CHANGELOG stubs

**Importance:** High (publishing requirement)
**Complexity:** S (Small)

The Cloudron App Store expects `POSTINSTALL.md` for post-install instructions and a structured `CHANGELOG.md`. Currently these are handled via inline fields but not generated as separate files.

**UI changes (index.html):**
- No UI changes needed. The existing "Post-Install Message" textarea and "Changelog" textarea already capture content.

**Generator changes (generators.js):**
- Add `generatePostInstall(config)` -- returns markdown file content when `config.postInstallMessage` is set. Returns `null` otherwise.
- Add `generateChangelog(config)` -- returns structured markdown from `config.changelog`. Returns `null` otherwise.
- Update `generateManifest()`: When postInstallMessage is set, emit `"postInstallMessage": "file://POSTINSTALL.md"` instead of inline text.

**Validation rules (app.js):**
- No new validation.

**Tests to add:**
- test.html: Manifest references `file://POSTINSTALL.md` when content is provided.

**ZIP changes (app.js):**
- `downloadZip()`: Add `POSTINSTALL.md` and `CHANGELOG.md` files to the ZIP when content exists.

---

### 3.2 Full cloudron versions workflow

**Importance:** High (publishing workflow)
**Complexity:** M (Medium)

The current `CloudronVersions.json` generates a single-version stub. For publishing workflow completeness, users need to manage multiple versions.

**UI changes (index.html):**
- Enhance the "versions" preview tab area with an "Add Version" button.
- Each version row: version number, Docker image tag, optional changelog note.
- The first version auto-populates from the manifest version.

**Generator changes (generators.js):**
- `generateCloudronVersions()`: Accept an array of version entries instead of a single version. Generate the full versions object.

**Validation rules (app.js):**
- Version numbers must be valid semver.
- No duplicate version numbers.
- Image tags should include registry prefix.

**Tests to add:**
- test.html: Multi-version output; single-version fallback; validates semver.

---

### 3.3 Icon file handling

**Importance:** High (publishing requirement)
**Complexity:** M (Medium)

The App Store requires a 256x256 PNG icon. Currently only a filename field exists. Users need the ability to upload or provide an actual image.

**UI changes (index.html):**
- Replace the icon filename text input with a file upload `<input type="file" accept="image/png">`.
- Show a preview thumbnail of the uploaded icon.
- Validate dimensions client-side (must be exactly 256x256).
- Keep the text input as fallback for URL-based icons.

**Generator changes (generators.js):**
- No generator changes. The icon is a binary file added to the ZIP.

**Validation rules (app.js):**
- Error if uploaded image is not PNG.
- Error if dimensions are not 256x256.
- Warning if no icon is provided and publishing fields are set.

**Tests to add:**
- test.html: Cannot easily test binary file handling in unit tests. Test validation logic (dimension check, format check) with mock data.

**ZIP changes (app.js):**
- `downloadZip()`: Add the uploaded PNG as `icon.png` in the ZIP root.

---

### 3.4 Environment variable reference per addon

**Importance:** Nice to have (reduces documentation burden)
**Complexity:** S (Small)

Each addon exposes environment variables. Including a reference in the generated README or as a separate file greatly reduces the "what variables do I use?" friction.

**UI changes (index.html):**
- Add a new preview tab "env-ref" that shows all environment variables for selected addons.

**Generator changes (generators.js):**
- Add `generateEnvReference(config)` that maps each addon to its environment variables:
  - `postgresql`: `CLOUDRON_POSTGRESQL_URL`, `CLOUDRON_POSTGRESQL_HOST`, `CLOUDRON_POSTGRESQL_PORT`, `CLOUDRON_POSTGRESQL_USERNAME`, `CLOUDRON_POSTGRESQL_PASSWORD`, `CLOUDRON_POSTGRESQL_DATABASE`
  - `mysql`: `CLOUDRON_MYSQL_URL`, `CLOUDRON_MYSQL_HOST`, `CLOUDRON_MYSQL_PORT`, `CLOUDRON_MYSQL_USERNAME`, `CLOUDRON_MYSQL_PASSWORD`, `CLOUDRON_MYSQL_DATABASE`
  - `mongodb`: `CLOUDRON_MONGODB_URL`, `CLOUDRON_MONGODB_HOST`, `CLOUDRON_MONGODB_PORT`, `CLOUDRON_MONGODB_USERNAME`, `CLOUDRON_MONGODB_PASSWORD`, `CLOUDRON_MONGODB_DATABASE`
  - `redis`: `CLOUDRON_REDIS_URL`, `CLOUDRON_REDIS_HOST`, `CLOUDRON_REDIS_PORT`, `CLOUDRON_REDIS_PASSWORD`
  - `sendmail`: `CLOUDRON_MAIL_SMTP_SERVER`, `CLOUDRON_MAIL_SMTP_PORT`, `CLOUDRON_MAIL_SMTP_USERNAME`, `CLOUDRON_MAIL_SMTP_PASSWORD`, `CLOUDRON_MAIL_FROM`, `CLOUDRON_MAIL_DOMAIN`
  - `oauth`: `CLOUDRON_OAUTH_CLIENT_ID`, `CLOUDRON_OAUTH_CLIENT_SECRET`, `CLOUDRON_OAUTH_ORIGIN`, `CLOUDRON_OAUTH_API_ORIGIN`
  - `oidc`: `CLOUDRON_OIDC_ISSUER`, `CLOUDRON_OIDC_CLIENT_ID`, `CLOUDRON_OIDC_CLIENT_SECRET`
  - `ldap`: `CLOUDRON_LDAP_URL`, `CLOUDRON_LDAP_USERS_BASE_DN`, `CLOUDRON_LDAP_GROUPS_BASE_DN`, `CLOUDRON_LDAP_BIND_DN`, `CLOUDRON_LDAP_BIND_PASSWORD`
  - `localstorage`: `CLOUDRON_APP_DATA_DIR` (always `/app/data`)
  - `turn`: `CLOUDRON_TURN_URL`, `CLOUDRON_TURN_USERNAME`, `CLOUDRON_TURN_PASSWORD`
  - Common: `CLOUDRON_APP_ORIGIN`, `CLOUDRON_APP_DOMAIN`, `CLOUDRON_APP_FQDN`
- Returns a markdown-formatted reference document.

**Validation rules (app.js):**
- No validation needed.

**Tests to add:**
- test.html: Env reference includes expected variables for each addon.

---

### 3.5 Dockerfile.cloudron naming option

**Importance:** Nice to have
**Complexity:** S (Small)

Some users prefer `Dockerfile.cloudron` to avoid conflicts with existing Dockerfiles in their project.

**UI changes (index.html):**
- Add a checkbox in the "Advanced" `<details>` section: "Use Dockerfile.cloudron naming".

**Generator changes (generators.js):**
- No change to content generation.

**Validation rules (app.js):**
- No validation needed.

**ZIP changes (app.js):**
- `downloadZip()`: When checked, name the file `Dockerfile.cloudron` instead of `Dockerfile`.

**Tests to add:**
- test.html: Verify filename in generated output (this is a ZIP-level concern, harder to unit test).

---

## Implementation Summary

### Phase 1 (Critical -- target first)

| Item | Feature                 | Complexity | Files Changed              |
|------|-------------------------|------------|----------------------------|
| 1.1  | httpPorts               | L          | index.html, generators.js, app.js, test.html, test-build.mjs |
| 1.2  | checklist               | S          | index.html, generators.js, app.js, test.html |
| 1.3  | oauth addon             | M          | index.html, generators.js, app.js, test.html |
| 1.4  | simpleauth addon        | M          | index.html, generators.js, app.js, test.html |
| 1.5  | logPaths                | S          | index.html, generators.js, app.js, test.html |
| 1.6  | Stack-specific start.sh | L          | index.html, generators.js, app.js, test.html, test-build.mjs |

### Phase 2 (Important -- follow-on)

| Item | Feature                  | Complexity | Files Changed              |
|------|--------------------------|------------|----------------------------|
| 2.1  | secondarySubdomains      | S          | index.html, generators.js, app.js, test.html |
| 2.2  | fullDomain               | S          | index.html, generators.js, app.js, test.html |
| 2.3  | singleUser               | S          | index.html, generators.js, app.js, test.html |
| 2.4  | maxBoxVersion/target     | S          | index.html, generators.js, app.js, test.html |
| 2.5  | localstorage sub-options | S          | index.html, generators.js, app.js, test.html |
| 2.6  | Symlinks to /run         | S          | generators.js, app.js, test.html, test-build.mjs |

### Phase 3 (Publishing polish)

| Item | Feature              | Complexity | Files Changed              |
|------|----------------------|------------|----------------------------|
| 3.1  | POSTINSTALL/CHANGELOG| S          | generators.js, app.js, test.html |
| 3.2  | Versions workflow    | M          | index.html, generators.js, app.js, test.html |
| 3.3  | Icon file handling   | M          | index.html, app.js         |
| 3.4  | Env var reference    | S          | index.html, generators.js, test.html |
| 3.5  | Dockerfile.cloudron  | S          | index.html, app.js         |

### Total Effort Estimate

- **Phase 1:** 2 Large + 2 Medium + 2 Small = ~5-7 dev sessions
- **Phase 2:** 6 Small = ~2-3 dev sessions
- **Phase 3:** 2 Medium + 3 Small = ~2-3 dev sessions
- **Grand total:** ~9-13 dev sessions

### Projected Coverage After All Phases

| Area                | Before | After  |
|---------------------|--------|--------|
| Manifest fields     | ~85%   | ~98%   |
| Addon types         | ~87%   | ~100%  |
| Dockerfile patterns | ~80%   | ~95%   |
| start.sh patterns   | ~75%   | ~92%   |
| Publishing workflow | ~60%   | ~90%   |

The remaining ~2-10% gaps in each area represent edge cases and rare Cloudron features (e.g., `initScript`, `targetBoxVersion` with complex semver ranges, custom restore hooks with database-specific logic) that are better handled as individual community contributions rather than built into the generator.
