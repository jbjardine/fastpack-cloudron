// generators.js — Pure generator functions for FastPackCloudron

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
  if (!config.sso || config.sso === "none") {
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
    addons.proxyauth = proxyOpts;
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
  const lines = [];
  lines.push(`FROM ${config.image}`);
  lines.push("");
  lines.push("RUN mkdir -p /app/code");
  lines.push("");
  lines.push("# Create the cloudron user (uid 808) used by the Cloudron runtime");
  lines.push(`RUN set -eux; \\
    if command -v groupadd >/dev/null 2>&1; then \\
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
    else \\
      echo \"No supported package manager found; creating a gosu shim using su\"; \\
      mkdir -p /usr/local/bin; \\
      printf '#!/bin/sh\\nUSER=$(echo \"$1\" | cut -d: -f1); shift\\nexec su -s /bin/sh \"$USER\" -c \"exec $*\"\\n' > /usr/local/bin/gosu; \\
      chmod +x /usr/local/bin/gosu; \\
    fi`);
  lines.push("");
  lines.push("COPY start.sh /app/code/start.sh");
  lines.push("RUN chmod +x /app/code/start.sh");
  lines.push("");

  // Install python3 for the health-check server when no web UI
  if (!config.hasWebUI) {
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
    lines.push("if [ ! -f /app/data/.initialized ]; then");
    lines.push("    echo \"Initializing data directory...\"");
    lines.push(`    echo "${config.version}" > /app/data/.initialized`);
    lines.push("fi");
    lines.push("");
    lines.push("chown -R cloudron:cloudron /app/data");
    lines.push("");
  }

  if (config.hasWebUI) {
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
 * Generates CloudronVersions.json for community app publishing.
 */
export function generateCloudronVersions(config) {
  const versions = {};
  versions[config.version] = {
    image: `docker.io/USERNAME/${config.id || "your-app"}:${config.version}`,
  };
  return JSON.stringify(versions, null, 2);
}
