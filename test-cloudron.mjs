#!/usr/bin/env node

// test-cloudron.mjs — End-to-end Cloudron integration tests.
// Generates packages → docker build → push to VM → cloudron install → verify → uninstall.
// Usage: node test-cloudron.mjs [--filter NAME]
// Requires: cloudron CLI logged in, Docker running, SSH to 192.168.60.17

import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  generateManifest,
  generateDockerfile,
  generateStartSh,
  generateDockerignore,
} from "./generators.js";

const DOMAIN = "192.168.60.17.nip.io";
const VM_HOST = "fastpack@192.168.60.17";
const FILTER = process.argv.find((a, i) => process.argv[i - 1] === "--filter") || "";

function patchStartSh(startSh, appCommand) {
  return startSh.replace("YOUR_APP_COMMAND", appCommand).replace("YOUR_SERVICE_COMMAND", appCommand);
}

const TEST_CONFIGS = [
  // 1. Basic web — no SSO
  {
    name: "web-nosso",
    subdomain: "t1",
    appCommand: "python3 -m http.server 8000",
    verify: (app) => httpCheck(app, 200),
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
    verify: (app) => httpCheck(app, 401, 302, 303),
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
    verify: (app) => httpCheck(app, 200),
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
    verify: (app) => httpCheck(app, 200),  // healthcheck Python server responds
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
    verify: (app) => httpCheck(app, 200, 404),  // busybox httpd may return 404 for / but at least responds
    config: {
      id: "io.fastpack.t8", title: "Test 8", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true,
      image: "busybox:latest", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [],
      author: "", tagline: "", description: "",
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
  const r = run("ssh", [VM_HOST,
    `curl -sk -o /dev/null -w '%{http_code}' https://${appDomain}`]);
  const code = parseInt(r.stdout);
  const ok = acceptCodes.includes(code);
  return { ok, detail: `HTTP ${code} (expected ${acceptCodes.join(" or ")})`, stdout: r.stdout, stderr: r.stderr };
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
      writeFileSync(join(dir, "Dockerfile"), generateDockerfile(tc.config));
      writeFileSync(join(dir, "start.sh"), patchStartSh(generateStartSh(tc.config), tc.appCommand));
      writeFileSync(join(dir, ".dockerignore"), generateDockerignore());
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
        "echo fastpack | sudo -S docker load -i /tmp/fp-e2e.tar && rm /tmp/fp-e2e.tar"], { timeout: 60_000 });
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
    spawnSync("ssh", [VM_HOST, `echo fastpack | sudo -S docker rmi ${imageTag} 2>/dev/null`], { stdio: "ignore" });
    try { rmSync(dir, { recursive: true, force: true }); } catch { /* Windows temp file lock */ }
  }
}

// Main
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
