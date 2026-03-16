#!/usr/bin/env node

// test-cloudron.mjs — End-to-end test: generate → docker build → cloudron install --image → healthcheck → uninstall
// Usage: node test-cloudron.mjs
// Requires: cloudron CLI logged in, Docker running

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

// Replace the placeholder command in start.sh with the real command for testing
function patchStartSh(startSh, appCommand) {
  return startSh.replace("YOUR_APP_COMMAND", appCommand).replace("YOUR_SERVICE_COMMAND", appCommand);
}

const SELFSIGNED = "--allow-selfsigned";
const BUILD_TIMEOUT = 300_000;
const INSTALL_TIMEOUT = 300_000;
const DOMAIN = "192.168.60.17.nip.io";

const TEST_CONFIGS = [
  {
    name: "python-web",
    subdomain: "test-py",
    appCommand: 'python3 -m http.server 8000',
    config: {
      id: "io.fastpack.testpython",
      title: "Test Python",
      version: "1.0.0",
      httpPort: 8000,
      healthCheckPath: "/",
      hasWebUI: true,
      image: "python:3-slim",
      database: null,
      sso: null,
      addons: ["localstorage"],
      tcpPorts: [],
      udpPorts: [],
      author: "",
      tagline: "",
      description: "",
    },
  },
];

function run(cmd, args, opts = {}) {
  // On Windows, npm global commands (cloudron) need shell:true to resolve
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
    stderr: result.stderr?.toString().trim() || result.error?.message || "",
    status: result.status,
  };
}

function step(label, fn) {
  process.stdout.write(`  ${label} ... `);
  const result = fn();
  if (result.ok) {
    console.log("\x1b[32mOK\x1b[0m");
  } else {
    console.log("\x1b[31mFAIL\x1b[0m");
    if (result.stderr) console.error(`    ${result.stderr.substring(0, 500)}`);
    if (result.stdout) console.error(`    ${result.stdout.substring(0, 500)}`);
  }
  return result;
}

async function testConfig(tc) {
  console.log(`\n--- ${tc.name} ---`);

  const dir = mkdtempSync(join(tmpdir(), `fastpack-cloudron-${tc.name}-`));
  const imageTag = `fastpack-test-${tc.name}:latest`;
  const appDomain = `${tc.subdomain}.${DOMAIN}`;

  try {
    // 1. Generate files
    step("Generate package files", () => {
      writeFileSync(join(dir, "CloudronManifest.json"), generateManifest(tc.config));
      writeFileSync(join(dir, "Dockerfile"), generateDockerfile(tc.config));
      writeFileSync(join(dir, "start.sh"), patchStartSh(generateStartSh(tc.config), tc.appCommand));
      writeFileSync(join(dir, ".dockerignore"), generateDockerignore());
      return { ok: true };
    });

    // 2. Docker build locally
    const buildResult = step("docker build", () =>
      run("docker", ["buildx", "build", "-t", imageTag, "."], { cwd: dir, timeout: BUILD_TIMEOUT })
    );
    if (!buildResult.ok) return false;

    // 3. Push image to Cloudron server (save to file, scp, load on remote)
    const tarPath = join(dir, "image.tar");
    const pushResult = step("Push image to Cloudron", () => {
      const save = run("docker", ["save", "-o", tarPath, imageTag], { timeout: BUILD_TIMEOUT });
      if (!save.ok) return save;
      const scp = run("scp", ["-o", "StrictHostKeyChecking=no", tarPath, "fastpack@192.168.60.17:/tmp/fp-test-image.tar"], { timeout: BUILD_TIMEOUT });
      if (!scp.ok) return scp;
      const load = run("ssh", ["fastpack@192.168.60.17",
        "echo fastpack | sudo -S docker load -i /tmp/fp-test-image.tar && rm /tmp/fp-test-image.tar"
      ], { timeout: BUILD_TIMEOUT });
      return load;
    });
    if (!pushResult.ok) return false;

    // 4. Install on Cloudron using the local image
    const installResult = step("cloudron install --image (from package dir)", () => {
      const r = run("cloudron", ["install", SELFSIGNED, "--image", imageTag, "--location", tc.subdomain],
        { cwd: dir, timeout: INSTALL_TIMEOUT });
      if (!r.ok) console.error(`\n    stdout: ${r.stdout.substring(0, 500)}\n    stderr: ${r.stderr.substring(0, 500)}`);
      return r;
    });
    if (!installResult.ok) return false;

    // 5. Healthcheck — wait then check via curl
    const healthResult = step("Healthcheck (wait 15s)", () => {
      spawnSync("sleep", ["15"]);
      const r = run("ssh", ["fastpack@192.168.60.17",
        `curl -sk -o /dev/null -w '%{http_code}' https://${appDomain}`]);
      const healthy = r.stdout.includes("200") || r.stdout.includes("301") || r.stdout.includes("302");
      return { ok: healthy, stdout: r.stdout, stderr: r.stderr };
    });

    // 6. Uninstall (always, to free the slot)
    step("cloudron uninstall", () =>
      run("cloudron", ["uninstall", SELFSIGNED, "--app", appDomain], { timeout: INSTALL_TIMEOUT })
    );

    return healthResult.ok;
  } finally {
    // Cleanup local image
    spawnSync("docker", ["rmi", imageTag], { stdio: "ignore" });
    rmSync(dir, { recursive: true, force: true });
  }
}

// Main
console.log("FastPackCloudron — Cloudron Integration Tests");
console.log("=============================================");

let passed = 0;
let failed = 0;

for (const tc of TEST_CONFIGS) {
  const ok = await testConfig(tc);
  if (ok) passed++;
  else failed++;
}

console.log(`\n${passed}/${TEST_CONFIGS.length} integration tests passed`);
if (failed > 0) {
  console.log(`\x1b[31m${failed} FAILED\x1b[0m`);
  process.exit(1);
}
console.log("\x1b[32mAll integration tests passed!\x1b[0m");
