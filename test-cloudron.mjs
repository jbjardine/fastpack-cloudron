#!/usr/bin/env node

// test-cloudron.mjs — End-to-end Cloudron integration tests.
// Generates packages → docker build → push to VM → cloudron install → verify → uninstall.
// Usage: node test-cloudron.mjs [--filter NAME]
// Requires: cloudron CLI logged in, Docker running, and FASTPACK_E2E_* environment variables.

import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  generateManifest,
  generateDockerfile,
  generateStartSh,
  generateDockerignore,
  generateNginxConf,
} from "./generators.js";

const DOMAIN = process.env.FASTPACK_E2E_CLOUDRON_DOMAIN || "";
const VM_HOST = process.env.FASTPACK_E2E_SSH_HOST || "";
const SUDO_PASSWORD = process.env.FASTPACK_E2E_SUDO_PASSWORD || "";
const FILTER = process.argv.find((a, i) => process.argv[i - 1] === "--filter") || "";

function skip(reason) {
  console.log(`SKIP: ${reason}`);
  process.exit(0);
}

function patchStartSh(startSh, appCommand) {
  return startSh.replace("YOUR_APP_COMMAND", appCommand).replace("YOUR_SERVICE_COMMAND", appCommand);
}

const TEST_CONFIGS = [
  // 1. Basic web — no SSO
  {
    name: "web-nosso",
    subdomain: "t1",
    appCommand: "python3 -m http.server 8000",
    verify: (app) => {
      const r = verifyApp(app);
      if (r.status !== 200) return { ok: false, detail: `HTTP ${r.status} (expected 200)` };
      if (!r.body.includes("<")) return { ok: false, detail: "Body has no HTML content" };
      if (/login|sso|redirect/i.test(r.body)) return { ok: false, detail: "Unexpected SSO redirect in body" };
      return { ok: true, detail: `HTTP 200, body ${r.body.length} bytes, no SSO redirect` };
    },
    config: {
      id: "io.fastpack.t1", title: "Test 1", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 2. Web + proxyAuth SSO
  {
    name: "web-proxyauth",
    subdomain: "t2",
    appCommand: "python3 -m http.server 8000",
    verify: (app) => {
      const r = verifyApp(app);
      if (![401, 302, 303].includes(r.status)) return { ok: false, detail: `HTTP ${r.status} (expected 401/302/303)` };
      const hasLogin = /login|sign.?in|auth/i.test(r.body);
      return { ok: true, detail: `HTTP ${r.status}, login page present: ${hasLogin}` };
    },
    config: {
      id: "io.fastpack.t2", title: "Test 2", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "python:3-slim", database: null, sso: "proxyAuth",
      proxyauthPath: "", proxyauthBasicAuth: false, proxyauthBearerAuth: false,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 3. Alpine web — apk path end-to-end
  {
    name: "alpine-web",
    subdomain: "t3",
    appCommand: "python3 -m http.server 8000",
    verify: (app) => {
      const r = verifyApp(app);
      if (r.status !== 200) return { ok: false, detail: `HTTP ${r.status} (expected 200)` };
      if (r.body.length === 0) return { ok: false, detail: "Body is empty" };
      return { ok: true, detail: `HTTP 200, body ${r.body.length} bytes` };
    },
    config: {
      id: "io.fastpack.t3", title: "Test 3", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "python:3-alpine", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 4. TCP mode — Python healthcheck
  {
    name: "tcp-mode",
    subdomain: "t4",
    appCommand: "sleep infinity",
    verify: (app) => {
      const r = verifyApp(app);
      if (r.status !== 200) return { ok: false, detail: `HTTP ${r.status} (expected 200)` };
      if (!r.body.includes("OK")) return { ok: false, detail: `Healthcheck body "${r.body}" does not contain "OK"` };
      return { ok: true, detail: `HTTP 200, healthcheck body: "${r.body}"` };
    },
    config: {
      id: "io.fastpack.t4", title: "Test 4", version: "1.0.0",
      httpPort: 8080, healthCheckPath: "/", hasWebUI: false,
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"],
      tcpPorts: [{ name: "SVC_PORT", title: "Service", containerPort: 9000, defaultValue: 9000 }],
      udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 5. No localstorage
  {
    name: "no-storage",
    subdomain: "t5",
    appCommand: "python3 -m http.server 8000",
    verify: (app) => httpCheck(app, 200),
    config: {
      id: "io.fastpack.t5", title: "Test 5", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "python:3-slim", database: null, sso: null,
      addons: [], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 6. Fedora (dnf/setpriv path)
  {
    name: "fedora",
    subdomain: "t6",
    appCommand: "python3 -m http.server 8000",
    extraDockerLines: "RUN dnf install -y python3 && dnf clean all",
    verify: (app) => httpCheck(app, 200),
    config: {
      id: "io.fastpack.t6", title: "Test 6", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "fedora:latest", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 7. Multi-ports (2 TCP + 1 UDP)
  {
    name: "multi-ports",
    subdomain: "t7",
    appCommand: "sleep infinity",
    verify: (app) => httpCheck(app, 200),
    config: {
      id: "io.fastpack.t7", title: "Test 7", version: "1.0.0",
      httpPort: 8080, healthCheckPath: "/", hasWebUI: false,
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"],
      tcpPorts: [
        { name: "PORT_A", title: "Port A", containerPort: 9000, defaultValue: 9000 },
        { name: "PORT_B", title: "Port B", containerPort: 9001, defaultValue: 9001 },
      ],
      udpPorts: [{ name: "UDP_PORT", title: "UDP", containerPort: 5353, defaultValue: 5353 }],
      author: "", tagline: "", description: "",
    },
  },
  // 8. BusyBox (shim gosu + busybox addgroup)
  {
    name: "busybox",
    subdomain: "t8",
    appCommand: "httpd -f -p 8000 -h /tmp",
    verify: (app) => {
      const r = verifyApp(app);
      if (![200, 404].includes(r.status)) return { ok: false, detail: `HTTP ${r.status} (expected 200 or 404)` };
      return { ok: true, detail: `HTTP ${r.status}, httpd responding, body ${r.body.length} bytes` };
    },
    config: {
      id: "io.fastpack.t8", title: "Test 8", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "busybox:latest", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
    },
  },
  // 9. Multi-service (two Python HTTP servers behind nginx)
  {
    name: "multi-service",
    subdomain: "t9",
    appCommand: null,  // not used — services define their own commands
    verify: (app) => {
      // /app1 → HTTP 200 (no SSO)
      const r1 = verifyApp(app, "/app1/");
      if (r1.status !== 200) return { ok: false, detail: `/app1 HTTP ${r1.status} (expected 200)` };
      // /app2 → HTTP 200 (no SSO in this test, proxyAuth needs Cloudron SSO setup)
      const r2 = verifyApp(app, "/app2/");
      if (r2.status !== 200) return { ok: false, detail: `/app2 HTTP ${r2.status} (expected 200)` };
      return { ok: true, detail: `/app1 OK, /app2 OK` };
    },
    config: {
      id: "io.fastpack.t9", title: "Test 9", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
      services: [
        { name: "app1", command: "python3 -m http.server 3001", internalPort: 3001, routePath: "/app1", sso: "none" },
        { name: "app2", command: "python3 -m http.server 3002", internalPort: 3002, routePath: "/app2", sso: "none" },
      ],
    },
  },
];

function run(cmd, args, opts = {}) {
  const needsShell = cmd === "cloudron";
  const result = spawnSync(cmd, args, {
    timeout: opts.timeout || 60_000,
    stdio: ["pipe", "pipe", "pipe"],
    cwd: opts.cwd,
    shell: needsShell,
  });
  return {
    ok: result.status === 0,
    stdout: result.stdout?.toString().trim() || "",
    stderr: result.stderr?.toString().trim() || "",
  };
}

function step(label, fn) {
  process.stdout.write(`    ${label} ... `);
  const result = fn();
  if (result.ok) {
    console.log("\x1b[32mOK\x1b[0m");
  } else {
    console.log("\x1b[31mFAIL\x1b[0m");
    if (result.detail) console.error(`      ${result.detail.substring(0, 300)}`);
    if (result.stderr) console.error(`      ${result.stderr.substring(0, 300)}`);
  }
  return result;
}

function httpCheck(appDomain, ...acceptCodes) {
  const result = verifyApp(appDomain);
  const ok = acceptCodes.includes(result.status);
  return { ok, detail: `HTTP ${result.status} (expected ${acceptCodes.join(" or ")})`, stdout: String(result.status), stderr: result.stderr || "" };
}

function verifyApp(appDomain, path = "/") {
  const url = `https://${appDomain}${path}`;
  const statusR = run("ssh", [VM_HOST, `curl -sk -o /dev/null -w '%{http_code}' ${url}`]);
  const bodyR = run("ssh", [VM_HOST, `curl -sk ${url}`]);
  return {
    status: parseInt(statusR.stdout) || 0,
    body: bodyR.stdout || "",
    stderr: statusR.stderr || bodyR.stderr || "",
  };
}

async function testConfig(tc) {
  console.log(`\n  --- ${tc.name} ---`);

  const dir = mkdtempSync(join(tmpdir(), `fp-cloudron-${tc.name}-`));
  const imageTag = `fastpack-e2e-${tc.name}:latest`;
  const appDomain = `${tc.subdomain}.${DOMAIN}`;
  let installed = false;

  try {
    // 1. Generate
    const genResult = step("Generate", () => {
      writeFileSync(join(dir, "CloudronManifest.json"), generateManifest(tc.config));
      let dockerfile = generateDockerfile(tc.config);
      if (tc.extraDockerLines) {
        // Insert extra lines before CMD
        dockerfile = dockerfile.replace('CMD ["/app/code/start.sh"]',
          tc.extraDockerLines + '\n\nCMD ["/app/code/start.sh"]');
      }
      writeFileSync(join(dir, "Dockerfile"), dockerfile);
      const startSh = tc.appCommand
        ? patchStartSh(generateStartSh(tc.config), tc.appCommand)
        : generateStartSh(tc.config);
      writeFileSync(join(dir, "start.sh"), startSh);
      writeFileSync(join(dir, ".dockerignore"), generateDockerignore());
      // Generate nginx.conf for multi-service configs
      if (tc.config.services && tc.config.services.length > 0) {
        writeFileSync(join(dir, "nginx.conf"), generateNginxConf(tc.config));
      }
      return { ok: true };
    });
    if (!genResult.ok) return false;

    // 2. Docker build
    const buildResult = step("Docker build", () =>
      run("docker", ["buildx", "build", "-t", imageTag, "."], { cwd: dir, timeout: 300_000 }));
    if (!buildResult.ok) return false;

    // 3. Push to VM
    const tarPath = join(dir, "image.tar");
    const pushResult = step("Push to Cloudron VM", () => {
      const save = run("docker", ["save", "-o", tarPath, imageTag], { timeout: 120_000 });
      if (!save.ok) return save;
      const scp = run("scp", [tarPath, `${VM_HOST}:/tmp/fp-e2e.tar`], { timeout: 120_000 });
      if (!scp.ok) return scp;
      return run("ssh", [VM_HOST,
        `printf '%s\\n' '${SUDO_PASSWORD.replace(/'/g, "'\\''")}' | sudo -S docker load -i /tmp/fp-e2e.tar && rm /tmp/fp-e2e.tar`], { timeout: 60_000 });
    });
    if (!pushResult.ok) return false;

    // 4. Cloudron install
    const installResult = step("Cloudron install", () =>
      run("cloudron", ["install", "--allow-selfsigned", "--image", imageTag, "--location", tc.subdomain],
        { cwd: dir, timeout: 300_000 }));
    if (!installResult.ok) return false;
    installed = true;

    // 5. Wait + verify
    step("Wait 20s", () => { spawnSync("sleep", ["20"]); return { ok: true }; });

    const verifyResult = step("Verify", () => tc.verify(appDomain));

    return verifyResult.ok;
  } finally {
    // Always uninstall if installed
    if (installed) {
      step("Uninstall", () =>
        run("cloudron", ["uninstall", "--allow-selfsigned", "--app", appDomain], { timeout: 60_000 }));
    }
    // Cleanup
    spawnSync("docker", ["rmi", imageTag], { stdio: "ignore" });
    spawnSync("ssh", [VM_HOST, `printf '%s\\n' '${SUDO_PASSWORD.replace(/'/g, "'\\''")}' | sudo -S docker rmi ${imageTag} 2>/dev/null`], { stdio: "ignore" });
    try { rmSync(dir, { recursive: true, force: true }); } catch { /* Windows temp file lock */ }
  }
}

// Main
if (!DOMAIN) skip("set FASTPACK_E2E_CLOUDRON_DOMAIN to run live Cloudron integration tests");
if (!VM_HOST) skip("set FASTPACK_E2E_SSH_HOST to run live Cloudron integration tests");
if (!SUDO_PASSWORD) skip("set FASTPACK_E2E_SUDO_PASSWORD to push Docker images to the Cloudron host");

const configs = FILTER
  ? TEST_CONFIGS.filter(c => c.name.includes(FILTER))
  : TEST_CONFIGS;

console.log("FastPackCloudron — Cloudron Integration Tests");
console.log(`${"=".repeat(45)}`);
console.log(`Testing ${configs.length} configs on ${DOMAIN}\n`);

let passed = 0;
let failed = 0;
const failures = [];

for (const tc of configs) {
  const ok = await testConfig(tc);
  if (ok) passed++;
  else { failed++; failures.push(tc.name); }
}

console.log(`\n${"=".repeat(45)}`);
console.log(`${passed}/${configs.length} integration tests passed`);
if (failed > 0) {
  console.log(`\x1b[31m${failed} FAILED: ${failures.join(", ")}\x1b[0m`);
  process.exit(1);
}
console.log("\x1b[32mAll integration tests passed!\x1b[0m");
