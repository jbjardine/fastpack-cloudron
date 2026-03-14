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

  // author and description only if non-empty
  if (config.author && config.author.trim() !== "") {
    manifest.author = config.author;
  }
  if (config.description && config.description.trim() !== "") {
    manifest.description = config.description;
  }

  // SSO at root level: optionalSso when sso is null or "none"
  if (!config.sso || config.sso === "none") {
    manifest.optionalSso = true;
  }

  // Addons object
  const addons = {};

  // Database addon
  if (config.database && config.database !== "none") {
    addons[config.database] = {};
  }

  // SSO addons
  if (config.sso === "proxyAuth") {
    addons.proxyAuth = {};
  } else if (config.sso === "oidc") {
    addons.oidc = {
      loginRedirectUri: "/auth/callback",
      logoutRedirectUri: "/",
    };
  } else if (config.sso === "ldap") {
    addons.ldap = {};
  }

  // Additional addons from array
  if (config.addons && Array.isArray(config.addons)) {
    for (const addon of config.addons) {
      addons[addon] = {};
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
 */
export function generateDockerfile(config) {
  const lines = [];
  lines.push(`FROM ${config.image}`);
  lines.push("");
  lines.push("RUN mkdir -p /app/code /app/data");
  lines.push("");
  lines.push("COPY start.sh /app/code/start.sh");
  lines.push("RUN chmod +x /app/code/start.sh");
  lines.push("");

  // EXPOSE: always httpPort
  const exposePorts = [String(config.httpPort)];

  // If not hasWebUI, also expose tcp and udp container ports
  if (!config.hasWebUI) {
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
  }

  lines.push(`EXPOSE ${exposePorts.join(" ")}`);
  lines.push("");
  lines.push('CMD ["/app/code/start.sh"]');

  return lines.join("\n") + "\n";
}

/**
 * Generates start.sh content.
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
    lines.push("    touch /app/data/.initialized");
    lines.push("fi");
    lines.push("");
    lines.push("chown -R cloudron:cloudron /app/data");
    lines.push("");
  }

  if (config.hasWebUI) {
    // Single process — exec placeholder
    lines.push("# Start the application");
    lines.push("exec /app/code/YOUR_APP_COMMAND");
  } else {
    // Background service + Python mini healthcheck + wait
    lines.push("# Start the service in the background");
    lines.push("/app/code/YOUR_SERVICE_COMMAND &");
    lines.push("");
    lines.push("# Health check endpoint");
    lines.push(
      `python3 -c "
import http.server
handler = http.server.SimpleHTTPRequestHandler
httpd = http.server.HTTPServer(('0.0.0.0', ${config.httpPort}), handler)
httpd.serve_forever()
" &`
    );
    lines.push("");
    lines.push("wait");
  }

  return lines.join("\n") + "\n";
}

/**
 * Returns static .dockerignore content.
 */
export function generateDockerignore() {
  return ".git\nREADME.md\n";
}

/**
 * Generates README.md content for the packaged app.
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
  lines.push(`cloudron install --image ${config.id}`);
  lines.push("```");
  lines.push("");
  lines.push("## Update");
  lines.push("");
  lines.push("```bash");
  lines.push("cloudron build");
  lines.push(`cloudron update --app <app-id>`);
  lines.push("```");
  lines.push("");
  lines.push("## Notes");
  lines.push("");
  lines.push(
    "- If the upstream Docker image runs as a non-root USER, you may need to adjust file permissions or switch to a root-based entrypoint."
  );

  return lines.join("\n") + "\n";
}
