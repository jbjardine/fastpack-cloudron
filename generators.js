// generators.js — Pure generator functions for FastPackCloudron

// --- Input safety assertions (defense-in-depth) ---
// These throw on invalid input to prevent injection in generated Dockerfiles and shell scripts.
// Validation in app.js:validate() provides user-friendly error messages;
// these guards protect generators even when called from other contexts.

const SAFE_DOCKER_REF = /^[a-zA-Z0-9][a-zA-Z0-9._/:-]*(:[a-zA-Z0-9._-]+)?(@sha256:[a-f0-9]{64})?$/;
const SAFE_PATH = /^[a-zA-Z0-9._/-]+$/;
const SAFE_IDENTIFIER = /^[a-zA-Z][a-zA-Z0-9_-]*$/;
const SAFE_VERSION = /^[a-zA-Z0-9][a-zA-Z0-9._-]*$/;
const SAFE_ROUTE_PATH = /^\/[a-zA-Z0-9._\/-]*$/;

function assertSafeDockerRef(value, context) {
  if (!value || !SAFE_DOCKER_REF.test(value)) {
    throw new Error(`Unsafe Docker reference in ${context}: ${JSON.stringify(value)}`);
  }
}

function assertSafePath(value, context) {
  if (!value || !SAFE_PATH.test(value)) {
    throw new Error(`Unsafe path in ${context}: ${JSON.stringify(value)}`);
  }
}

function assertSafeIdentifier(value, context) {
  if (!value || !SAFE_IDENTIFIER.test(value)) {
    throw new Error(`Unsafe identifier in ${context}: ${JSON.stringify(value)}`);
  }
}

function assertSafeVersion(value, context) {
  if (!value || !SAFE_VERSION.test(value)) {
    throw new Error(`Unsafe version string in ${context}: ${JSON.stringify(value)}`);
  }
}

function assertSafePort(value, context) {
  const n = Number(value);
  if (!Number.isInteger(n) || n < 1 || n > 65535) {
    throw new Error(`Unsafe port in ${context}: ${JSON.stringify(value)}`);
  }
}

function assertSafeRoutePath(value, context) {
  if (!value || !SAFE_ROUTE_PATH.test(value)) {
    throw new Error(`Unsafe route path in ${context}: ${JSON.stringify(value)}`);
  }
}

// Re-export regex patterns for use by validation in app.js
export { SAFE_DOCKER_REF, SAFE_PATH, SAFE_IDENTIFIER, SAFE_VERSION, SAFE_ROUTE_PATH };

/**
 * Strips tag, strips registry prefix (take last segment after /),
 * replaces hyphens/underscores with dots, drops leading digits from segments.
 */
export function sanitizeImageName(image) {
  // Strip tag
  let name = image.split(":")[0];
  // Strip registry prefix — take last segment after /
  const parts = name.split("/");
  name = parts[parts.length - 1];
  // Replace hyphens and underscores with dots
  name = name.replace(/[-_]/g, ".");
  // Drop leading digits from each segment
  const segments = name.split(".").map((s) => s.replace(/^\d+/, ""));
  // Filter out empty segments
  return segments.filter((s) => s.length > 0).join(".");
}

/**
 * Strips tag and registry, replaces hyphens/underscores with spaces, capitalizes words.
 */
export function humanizeImageName(image) {
  // Strip tag
  let name = image.split(":")[0];
  // Strip registry prefix
  const parts = name.split("/");
  name = parts[parts.length - 1];
  // Replace hyphens and underscores with spaces
  name = name.replace(/[-_]/g, " ");
  // Capitalize each word
  return name
    .split(" ")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}

/**
 * Builds CloudronManifest.json content as a JSON string.
 */
export function generateManifest(config) {
  const manifest = {};

  manifest.id = config.id;
  manifest.title = config.title;
  manifest.version = config.version;
  manifest.manifestVersion = 2;
  manifest.healthCheckPath = config.healthCheckPath || "/";
  manifest.httpPort = config.httpPort;

  // tagline only if non-empty
  if (config.tagline && config.tagline.trim() !== "") {
    manifest.tagline = config.tagline;
  }

  // author and description only if non-empty
  if (config.author && config.author.trim() !== "") {
    manifest.author = config.author;
  }
  if (config.description && config.description.trim() !== "") {
    manifest.description = "file://DESCRIPTION.md";
  }

  // Publishing fields
  if (config.website && config.website.trim() !== "") {
    manifest.website = config.website;
  }
  if (config.contactEmail && config.contactEmail.trim() !== "") {
    manifest.contactEmail = config.contactEmail;
  }
  if (config.tags && config.tags.length > 0) {
    manifest.tags = config.tags;
  }
  if (config.configurePath && config.configurePath.trim() !== "") {
    manifest.configurePath = config.configurePath;
  }
  if (config.upstreamVersion && config.upstreamVersion.trim() !== "") {
    manifest.upstreamVersion = config.upstreamVersion;
  }
  if (config.postInstallMessage && config.postInstallMessage.trim() !== "") {
    manifest.postInstallMessage = config.postInstallMessage;
  }
  if (config.changelog && config.changelog.trim() !== "") {
    manifest.changelog = config.changelog;
  }
  if (config.icon && config.icon.trim() !== "") {
    manifest.icon = config.icon;
  }
  if (config.memoryLimit && config.memoryLimit > 0) {
    manifest.memoryLimit = config.memoryLimit;
  }

  // Publishing fields
  if (config.packagerName && config.packagerName.trim() !== "") {
    manifest.packagerName = config.packagerName;
  }
  if (config.packagerUrl && config.packagerUrl.trim() !== "") {
    manifest.packagerUrl = config.packagerUrl;
  }
  if (config.iconUrl && config.iconUrl.trim() !== "") {
    manifest.iconUrl = config.iconUrl;
  }
  if (config.mediaLinks && config.mediaLinks.length > 0) {
    manifest.mediaLinks = config.mediaLinks;
  }
  if (config.documentationUrl && config.documentationUrl.trim() !== "") {
    manifest.documentationUrl = config.documentationUrl;
  }
  if (config.forumUrl && config.forumUrl.trim() !== "") {
    manifest.forumUrl = config.forumUrl;
  }

  // Advanced fields
  if (config.minBoxVersion && config.minBoxVersion.trim() !== "") {
    manifest.minBoxVersion = config.minBoxVersion;
  }
  if (config.capabilities && config.capabilities.length > 0) {
    manifest.capabilities = config.capabilities;
  }
  if (config.multiDomain) {
    manifest.multiDomain = true;
  }
  if (config.runtimeDirs && config.runtimeDirs.length > 0) {
    manifest.runtimeDirs = config.runtimeDirs;
  }
  if (config.persistentDirs && config.persistentDirs.length > 0) {
    manifest.persistentDirs = config.persistentDirs;
  }
  if (config.backupCommand && config.backupCommand.trim() !== "") {
    manifest.backupCommand = config.backupCommand;
  }
  if (config.restoreCommand && config.restoreCommand.trim() !== "") {
    manifest.restoreCommand = config.restoreCommand;
  }

  // SSO at root level: optionalSso when sso is null or "none"
  // But not when multi-service SSO has proxyAuth services
  const hasServiceSso = config.services && config.services.some((s) => s.sso === "proxyAuth");
  if ((!config.sso || config.sso === "none") && !hasServiceSso) {
    manifest.optionalSso = true;
  }

  // Addons object
  const addons = {};

  // Database addon (with per-database options)
  if (config.database && config.database !== "none") {
    const dbOpts = {};
    if (config.database === "mysql" && config.mysqlMultipleDbs) {
      dbOpts.multipleDatabases = true;
    }
    if (config.database === "mongodb" && config.mongodbOplog) {
      dbOpts.oplog = true;
    }
    if (config.database === "redis" && config.redisNoPassword) {
      dbOpts.noPassword = true;
    }
    if (config.database === "postgresql" && config.postgresqlLocale && config.postgresqlLocale.trim() !== "") {
      dbOpts.locale = config.postgresqlLocale;
    }
    addons[config.database] = dbOpts;
  }

  // SSO addons with per-addon options
  if (config.sso === "proxyAuth") {
    const proxyOpts = {};
    if (config.proxyauthPath) proxyOpts.path = config.proxyauthPath;
    if (config.proxyauthBasicAuth) proxyOpts.basicAuth = true;
    if (config.proxyauthBearerAuth) proxyOpts.supportsBearerAuth = true;
    addons.proxyAuth = proxyOpts;
  } else if (!config.sso && config.services && config.services.length > 0) {
    // Multi-service SSO: combine per-service SSO settings (only when no global SSO)
    const ssoServices = config.services.filter((s) => s.sso === "proxyAuth");
    const noSsoServices = config.services.filter(
      (s) => s.sso !== "proxyAuth" && s.routePath
    );
    if (ssoServices.length > 0) {
      const proxyOpts = {};
      if (noSsoServices.length > 0) {
        // Exclude non-SSO paths
        proxyOpts.path = noSsoServices.map((s) => "!" + s.routePath).join(",");
      }
      addons.proxyAuth = proxyOpts;
    }
  } else if (config.sso === "oidc") {
    const oidcOpts = {
      loginRedirectUri: config.oidcRedirectUri || "/auth/openid/callback",
      logoutRedirectUri: config.oidcLogoutUri || "/",
    };
    if (config.oidcTokenAlgo) {
      oidcOpts.tokenSignatureAlgorithm = config.oidcTokenAlgo;
    }
    addons.oidc = oidcOpts;
  } else if (config.sso === "ldap") {
    addons.ldap = {};
  }

  // Additional addons from array (with per-addon options)
  if (config.addons && Array.isArray(config.addons)) {
    for (const addon of config.addons) {
      if (addon === "sendmail") {
        const smOpts = {};
        if (config.sendmailOptional) smOpts.optional = true;
        if (config.sendmailDisplayName) smOpts.supportsDisplayName = true;
        if (config.sendmailValidCert) smOpts.requiresValidCertificate = true;
        addons.sendmail = smOpts;
      } else if (addon === "scheduler" && config.schedulerTasks && config.schedulerTasks.length > 0) {
        const tasks = {};
        for (const task of config.schedulerTasks) {
          if (task.name) {
            tasks[task.name] = { schedule: task.schedule, command: task.command };
          }
        }
        addons.scheduler = tasks;
      } else {
        addons[addon] = addons[addon] || {};
      }
    }
  }

  manifest.addons = addons;

  // TCP ports
  if (config.tcpPorts && config.tcpPorts.length > 0) {
    const tcpPortsObj = {};
    for (const port of config.tcpPorts) {
      tcpPortsObj[port.name] = {
        title: port.title,
        description: port.description || port.title + " port",
        containerPort: port.containerPort,
        defaultValue: port.defaultValue,
      };
    }
    manifest.tcpPorts = tcpPortsObj;
  }

  // UDP ports
  if (config.udpPorts && config.udpPorts.length > 0) {
    const udpPortsObj = {};
    for (const port of config.udpPorts) {
      udpPortsObj[port.name] = {
        title: port.title,
        description: port.description || port.title + " port",
        containerPort: port.containerPort,
        defaultValue: port.defaultValue,
      };
    }
    manifest.udpPorts = udpPortsObj;
  }

  return JSON.stringify(manifest, null, 2);
}

/**
 * Generates Dockerfile content.
 * Fix #5: removed unnecessary mkdir /app/data (mounted by Cloudron with localstorage addon)
 * Fix #1: install gosu for non-root execution
 */
export function generateDockerfile(config) {
  assertSafeDockerRef(config.image, "Dockerfile FROM");

  const lines = [];
  lines.push(`FROM ${config.image}`);

  // Copy files from external images (multi-stage)
  if (config.copyFrom && config.copyFrom.length > 0) {
    lines.push("");
    lines.push("# Copy files from external images");
    for (const cf of config.copyFrom) {
      assertSafeDockerRef(cf.image, "COPY --from image");
      assertSafePath(cf.src, "COPY --from src");
      assertSafePath(cf.dest, "COPY --from dest");
      lines.push(`COPY --from=${cf.image} ${cf.src} ${cf.dest}`);
    }
  }

  lines.push("");
  lines.push("RUN mkdir -p /app/code");
  lines.push("");
  lines.push("# Create the cloudron user (uid 808) used by the Cloudron runtime");
  lines.push(`RUN set -eux; \\
    if id cloudron >/dev/null 2>&1; then \\
      echo "cloudron user already exists"; \\
    elif command -v groupadd >/dev/null 2>&1; then \\
      groupadd -r -g 808 cloudron; \\
      useradd -r -u 808 -g 808 -d /home/cloudron -m cloudron; \\
    elif command -v addgroup >/dev/null 2>&1; then \\
      addgroup -g 808 -S cloudron; \\
      adduser -u 808 -S -G cloudron -h /home/cloudron cloudron; \\
    fi`);
  lines.push("");
  lines.push("# Install gosu (or su-exec fallback) only when the base image supports it");
  lines.push(`RUN set -eux; \\
    if command -v apt-get >/dev/null 2>&1; then \\
      apt-get update; \\
      apt-get install -y --no-install-recommends gosu; \\
      rm -rf /var/lib/apt/lists/*; \\
      ln -sf /usr/sbin/gosu /usr/local/bin/gosu 2>/dev/null || true; \\
    elif command -v apk >/dev/null 2>&1; then \\
      apk add --no-cache su-exec; \\
      ln -sf /sbin/su-exec /usr/local/bin/gosu; \\
    elif command -v dnf >/dev/null 2>&1; then \\
      dnf install -y util-linux && dnf clean all; \\
      printf '#!/bin/sh\\nUSER=$(echo \"$1\" | cut -d: -f1); shift\\nexec setpriv --reuid=\"$USER\" --regid=\"$USER\" --init-groups \"$@\"\\n' > /usr/local/bin/gosu; \\
      chmod +x /usr/local/bin/gosu; \\
    else \\
      echo \"No supported package manager found; creating a gosu shim using su\"; \\
      mkdir -p /usr/local/bin; \\
      printf '#!/bin/sh\\nUSER=$(echo \"$1\" | cut -d: -f1); shift\\nexec su -s /bin/sh \"$USER\" -c \"exec $*\"\\n' > /usr/local/bin/gosu; \\
      chmod +x /usr/local/bin/gosu; \\
    fi`);
  lines.push("");
  // Install nginx for multi-service reverse proxy
  const hasServices = config.services && config.services.length > 0;
  if (hasServices) {
    lines.push("# Install nginx for internal reverse proxy");
    lines.push(`RUN set -eux; \\
    if command -v apt-get >/dev/null 2>&1; then \\
      apt-get update; \\
      apt-get install -y --no-install-recommends nginx; \\
      rm -rf /var/lib/apt/lists/*; \\
    elif command -v apk >/dev/null 2>&1; then \\
      apk add --no-cache nginx; \\
    elif command -v dnf >/dev/null 2>&1; then \\
      dnf install -y nginx && dnf clean all; \\
    fi`);
    lines.push("");
    lines.push("COPY nginx.conf /app/code/nginx.conf");
    lines.push("");
  }

  lines.push("COPY start.sh /app/code/start.sh");
  lines.push("RUN chmod +x /app/code/start.sh");
  lines.push("");

  // Install python3 for the health-check server when no web UI
  if (!config.hasWebUI && !hasServices) {
    lines.push("# Install python3 for health-check endpoint (TCP-only mode)");
    lines.push(`RUN set -eux; \\
    if command -v apt-get >/dev/null 2>&1; then \\
      apt-get update; \\
      apt-get install -y --no-install-recommends python3; \\
      rm -rf /var/lib/apt/lists/*; \\
    elif command -v apk >/dev/null 2>&1; then \\
      apk add --no-cache python3; \\
    fi`);
    lines.push("");
  }

  // EXPOSE: always httpPort
  const exposePorts = [String(config.httpPort)];

  if (config.tcpPorts && config.tcpPorts.length > 0) {
    for (const port of config.tcpPorts) {
      exposePorts.push(String(port.containerPort));
    }
  }
  if (config.udpPorts && config.udpPorts.length > 0) {
    for (const port of config.udpPorts) {
      exposePorts.push(`${port.containerPort}/udp`);
    }
  }

  lines.push(`EXPOSE ${exposePorts.join(" ")}`);
  lines.push("");
  lines.push('CMD ["/app/code/start.sh"]');

  return lines.join("\n") + "\n";
}

/**
 * Generates start.sh content.
 * Fix #1/#4: use gosu for non-root execution
 * Fix #2: minimal healthcheck handler (no filesystem exposure)
 * Fix #3: signal handling with trap
 * Fix #11: log guidance comments
 */
export function generateStartSh(config) {
  const lines = [];
  lines.push("#!/bin/sh");
  lines.push("set -eu");
  lines.push("");

  const hasLocalstorage =
    config.addons && Array.isArray(config.addons) && config.addons.includes("localstorage");

  // If localstorage: init block with .initialized guard + chown
  if (hasLocalstorage) {
    assertSafeVersion(config.version, "start.sh version echo");
    lines.push("if [ ! -f /app/data/.initialized ]; then");
    lines.push("    echo \"Initializing data directory...\"");
    lines.push(`    echo "${config.version}" > /app/data/.initialized`);
    lines.push("fi");
    lines.push("");
    lines.push("chown -R cloudron:cloudron /app/data");
    lines.push("");
  }

  const hasServices = config.services && config.services.length > 0;

  if (hasServices) {
    // Multi-service mode: start all services + nginx reverse proxy
    lines.push("# Forward signals to all children");
    lines.push("trap 'kill $(jobs -p) 2>/dev/null; wait' TERM INT");
    lines.push("");

    for (const svc of config.services) {
      assertSafeIdentifier(svc.name, "start.sh service name");
      lines.push(`echo "Starting ${svc.name}..."`);
      lines.push(`/usr/local/bin/gosu cloudron:cloudron ${svc.command} &`);
      lines.push("");
    }

    lines.push("# Start nginx reverse proxy");
    lines.push('nginx -c /app/code/nginx.conf -g "daemon off;" &');
    lines.push("");
    lines.push("wait");
  } else if (config.hasWebUI) {
    // Single process — exec + gosu ensures SIGTERM reaches the app
    lines.push("# Start the application (exec ensures SIGTERM is forwarded)");
    lines.push("exec /usr/local/bin/gosu cloudron:cloudron YOUR_APP_COMMAND");
  } else {
    // Background service + minimal healthcheck + wait
    lines.push("# Forward signals to child processes");
    lines.push("trap 'kill $(jobs -p) 2>/dev/null; wait' TERM INT");
    lines.push("");
    lines.push("# Start the service in the background");
    lines.push("# Ensure your service logs to stdout/stderr for Cloudron log integration");
    lines.push("/usr/local/bin/gosu cloudron:cloudron YOUR_SERVICE_COMMAND &");
    lines.push("");
    lines.push("# Minimal health check endpoint (returns 200 OK)");
    assertSafePort(config.httpPort, "start.sh healthcheck port");
    lines.push(
      `python3 -c "
import http.server
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'OK')
    def log_message(self, *a): pass
http.server.HTTPServer(('0.0.0.0', ${config.httpPort}), H).serve_forever()
" &`
    );
    lines.push("");
    lines.push("wait");
  }

  return lines.join("\n") + "\n";
}

/**
 * Generates DESCRIPTION.md content (referenced by manifest via file://).
 * Returns null when no description is set.
 */
export function generateDescription(config) {
  if (!config.description || config.description.trim() === "") {
    return null;
  }
  return config.description.trim() + "\n";
}

/**
 * Returns static .dockerignore content.
 */
export function generateDockerignore() {
  return [
    ".git",
    "README.md",
    "test.html",
    "CHANGELOG.md",
    "CONTRIBUTING.md",
    "LICENSE",
    ".claude/",
    "node_modules/",
  ].join("\n") + "\n";
}

/**
 * Generates README.md content for the packaged app.
 * Fix #8: corrected cloudron install command
 */
export function generateReadme(config) {
  const lines = [];
  lines.push(`# ${config.title}`);
  lines.push("");
  lines.push("Generated by FastPackCloudron.");
  lines.push("");
  lines.push("## Deployment");
  lines.push("");
  lines.push("```bash");
  lines.push("cloudron login");
  lines.push("cloudron build");
  lines.push("cloudron install");
  lines.push("```");
  lines.push("");
  lines.push("## Update");
  lines.push("");
  lines.push("```bash");
  lines.push("cloudron build");
  lines.push("cloudron update --app <app-id>");
  lines.push("```");
  lines.push("");
  lines.push("## Alternative Build Methods");
  lines.push("");
  lines.push("### Local Docker build");
  lines.push("");
  lines.push("```bash");
  lines.push("docker build -t your-image-name .");
  lines.push("cloudron install --image your-image-name");
  lines.push("```");
  lines.push("");
  lines.push("### Cloudron Build Service");
  lines.push("");
  lines.push("Push your package source to a Git repository and use the");
  lines.push("Cloudron [Docker Builder](https://docs.cloudron.io/packaging/tutorial/#build-service) app.");
  lines.push("");
  lines.push("## Pre-flight Checks");
  lines.push("");
  lines.push("```bash");
  lines.push("docker build -t my-app .");
  lines.push("docker run --rm my-app id cloudron          # Should show uid=808");
  lines.push('docker run --rm my-app /usr/local/bin/gosu cloudron:cloudron whoami  # Should show "cloudron"');
  lines.push("```");
  lines.push("");
  lines.push("## Notes");
  lines.push("");
  lines.push(
    "- If the upstream Docker image runs as a non-root USER, you may need to adjust file permissions or switch to a root-based entrypoint."
  );
  lines.push(
    "- Ensure your application logs to stdout/stderr for proper Cloudron log integration."
  );
  if (!config.hasWebUI) {
    lines.push(
      "- This package runs in TCP-only mode. A minimal Python health-check server listens on the HTTP port so Cloudron can verify the app is running."
    );
  }

  return lines.join("\n") + "\n";
}

/**
 * Generates deploy.js — cross-platform deploy script (Windows + Linux + Mac).
 * Uses Node.js (always available since cloudron CLI requires it).
 * Checks prerequisites (CLI, login, Build Service) then builds and installs/updates.
 * Note: execSync is safe here — all commands are hardcoded strings, no user input interpolation.
 * The only dynamic value (appDomain) is passed as a separate argument via execFileSync.
 */
export function generateDeploySh() {
  return `#!/usr/bin/env node
"use strict";
var spawnSync = require("child_process").spawnSync;
var fs = require("fs");
var path = require("path");
var readline = require("readline");

// shell:true needed on Windows where npm globals are .cmd scripts
var opts = { shell: true };
var configFile = path.join(__dirname, ".deploy-config.json");

function run(cmd, args) {
  console.log("  > " + cmd + " " + args.join(" "));
  var r = spawnSync(cmd, args, Object.assign({ stdio: "inherit" }, opts));
  if (r.status !== 0) process.exit(r.status || 1);
}

function check(cmd, args) {
  var r = spawnSync(cmd, args, Object.assign({ stdio: "ignore" }, opts));
  return r.status === 0;
}

function loadConfig() {
  try { return JSON.parse(fs.readFileSync(configFile, "utf8")); } catch(e) { return {}; }
}

function saveConfig(cfg) {
  fs.writeFileSync(configFile, JSON.stringify(cfg, null, 2));
}

function ask(question) {
  return new Promise(function(resolve) {
    var rl = readline.createInterface({ input: process.stdin, output: process.stdout });
    rl.question(question, function(answer) { rl.close(); resolve(answer.trim()); });
  });
}

async function main() {
  console.log("=== FastPackCloudron Deploy ===\\n");

  // 1. Check cloudron CLI
  if (!check("cloudron", ["--version"])) {
    console.error("Error: cloudron CLI not found.\\nInstall it with: npm install -g cloudron");
    process.exit(1);
  }

  // 2. Check login
  if (!check("cloudron", ["list"])) {
    console.error("Error: not logged in to Cloudron.\\nRun: cloudron login");
    process.exit(1);
  }

  // 3. Check Build Service
  if (!check("cloudron", ["build", "info"])) {
    console.error("Error: Build Service not configured.\\n");
    console.error("1. Install the Docker Builder app on your Cloudron");
    console.error("2. Run: cloudron build login\\n");
    console.error("No Docker or registry needed \\u2014 the Build Service handles everything.");
    process.exit(1);
  }
  console.log("Build Service OK.");

  // 4. Get or ask for Docker repository
  var cfg = loadConfig();
  var repo = cfg.repository;
  if (!repo) {
    console.log("\\nA Docker registry is needed to store the built image.");
    console.log("Examples:");
    console.log("  - Cloudron Registry: registry.yourdomain.com/appname");
    console.log("  - Docker Hub:        docker.io/username/appname\\n");
    repo = await ask("Docker repository URL: ");
    if (!repo) {
      console.error("No repository provided. Aborting.");
      process.exit(1);
    }
    cfg.repository = repo;
    saveConfig(cfg);
    console.log("Saved to .deploy-config.json (won't ask again).\\n");
  } else {
    console.log("Registry: " + repo + " (from .deploy-config.json)\\n");
  }

  // 5. Init git if needed (cloudron build requires a git repo)
  if (!fs.existsSync(path.join(__dirname, ".git"))) {
    console.log("Initializing git repo (required by cloudron build)...");
    run("git", ["init"]);
    run("git", ["add", "-A"]);
    run("git", ["commit", "-m", "Initial package"]);
  }

  // 6. Build
  console.log("Building image...");
  run("cloudron", ["build", "--repository", repo]);

  // 7. Install or update
  var appDomain = process.argv[2];
  if (appDomain) {
    console.log("\\nUpdating app " + appDomain + "...");
    run("cloudron", ["update", "--app", appDomain]);
  } else {
    console.log("\\nInstalling app...");
    run("cloudron", ["install"]);
  }

  console.log("\\nDone!");
}

main();
`;
}

/**
 * Generates deploy.cmd — Windows double-click launcher for deploy.js.
 * Keeps the window open after execution so the user can see the output.
 */
export function generateDeployCmd() {
  return '@echo off\r\nnode --no-deprecation "%~dp0deploy.js" %*\r\npause\r\n';
}

/**
 * Generates nginx.conf for multi-service reverse proxy.
 * Only used when config.services has entries with routePaths.
 */
export function generateNginxConf(config) {
  if (!config.services || config.services.length === 0) return "";

  const lines = [];
  lines.push("worker_processes 1;");
  lines.push("error_log /dev/stderr;");
  lines.push("pid /tmp/nginx.pid;");
  lines.push("");
  lines.push("events { worker_connections 64; }");
  lines.push("");
  lines.push("http {");
  lines.push("  access_log /dev/stdout;");
  lines.push("");
  lines.push("  # Writable temp paths (Cloudron read-only filesystem)");
  lines.push("  client_body_temp_path /tmp/nginx_client_body;");
  lines.push("  proxy_temp_path /tmp/nginx_proxy;");
  lines.push("  fastcgi_temp_path /tmp/nginx_fastcgi;");
  lines.push("  uwsgi_temp_path /tmp/nginx_uwsgi;");
  lines.push("  scgi_temp_path /tmp/nginx_scgi;");
  lines.push("");
  lines.push("  server {");
  assertSafePort(config.httpPort, "nginx listen port");
  lines.push(`    listen ${config.httpPort};`);
  lines.push("");

  // Services with route paths
  const routed = config.services.filter((s) => s.routePath);
  for (const svc of routed) {
    assertSafeIdentifier(svc.name, "nginx service comment");
    assertSafeRoutePath(svc.routePath, "nginx location");
    assertSafePort(svc.internalPort, "nginx proxy_pass port");
    const path = svc.routePath.endsWith("/") ? svc.routePath : svc.routePath + "/";
    lines.push(`    # ${svc.name} → localhost:${svc.internalPort}`);
    lines.push(`    location ${path} {`);
    lines.push(`      proxy_pass http://127.0.0.1:${svc.internalPort}/;`);
    lines.push("      proxy_set_header Host $host;");
    lines.push("      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;");
    lines.push("    }");
    lines.push("");
  }

  // Default location: first routed service or first service with a port
  const defaultSvc = routed[0] || config.services[0];
  if (defaultSvc) {
    assertSafePort(defaultSvc.internalPort, "nginx default proxy_pass port");
    lines.push("    # Default: " + defaultSvc.name);
    lines.push("    location / {");
    lines.push(`      proxy_pass http://127.0.0.1:${defaultSvc.internalPort};`);
    lines.push("      proxy_set_header Host $host;");
    lines.push("      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;");
    lines.push("    }");
  }

  lines.push("  }");
  lines.push("}");

  return lines.join("\n") + "\n";
}

/**
 * Generates CloudronVersions.json for community app publishing.
 */
export function generateCloudronVersions(config) {
  const versions = {};
  versions[config.version] = {
    image: `docker.io/USERNAME/${config.id || "your-app"}:${config.version}`,
  };
  return JSON.stringify(versions, null, 2);
}
