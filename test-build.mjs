#!/usr/bin/env node

// test-build.mjs — Verifies generated Dockerfiles build AND work correctly.
// Usage: node test-build.mjs [--no-cache] [--checks] [--filter NAME]
// --checks: also run post-build validation (docker run checks)
// --filter: only run tests matching NAME
// Requires: Docker daemon running, Node.js 18+

import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  generateDockerfile,
  generateStartSh,
  generateDockerignore,
} from "./generators.js";

const NO_CACHE = process.argv.includes("--no-cache");
const RUN_CHECKS = process.argv.includes("--checks");
const FILTER = process.argv.find((a, i) => process.argv[i - 1] === "--filter") || "";
const TIMEOUT = 300_000;

const TEST_CONFIGS = [
  // --- Existing: apt-get path ---
  {
    name: "debian-web",
    description: "Debian apt-get path (gosu + groupadd)",
    config: {
      image: "nginx:latest", httpPort: 8080, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  {
    name: "slim-web",
    description: "Debian slim apt-get path",
    config: {
      image: "python:3-slim", httpPort: 8000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- Existing: apk path ---
  {
    name: "alpine-web",
    description: "Alpine apk path (su-exec + addgroup)",
    config: {
      image: "node:20-alpine", httpPort: 3000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- Existing: TCP mode ---
  {
    name: "tcp-mosquitto",
    description: "TCP mode with Alpine (healthcheck Python + TCP port)",
    config: {
      image: "eclipse-mosquitto:2.0", httpPort: 8080, hasWebUI: false,
      tcpPorts: [{ name: "MQTT_PORT", title: "MQTT", containerPort: 1883, defaultValue: 1883 }],
      udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: pure Alpine ---
  {
    name: "alpine-pure",
    description: "Pure Alpine (apk adduser + su-exec)",
    config: {
      image: "alpine:3.19", httpPort: 8000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: Alpine TCP (python3 via apk) ---
  {
    name: "alpine-tcp",
    description: "Alpine TCP mode (python3 via apk for healthcheck)",
    config: {
      image: "alpine:3.19", httpPort: 8080, hasWebUI: false,
      tcpPorts: [{ name: "TCP_PORT", title: "TCP", containerPort: 9000, defaultValue: 9000 }],
      udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: Fedora (no apt-get, no apk → gosu shim) ---
  {
    name: "fedora-shim",
    description: "Fedora (gosu shim fallback, groupadd GNU)",
    config: {
      image: "fedora:latest", httpPort: 8000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: No localstorage ---
  {
    name: "no-storage",
    description: "Web mode without localstorage (no chown, no .initialized)",
    config: {
      image: "python:3-slim", httpPort: 8000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: [], version: "1.0.0",
    },
  },
  // --- NEW: UDP ports ---
  {
    name: "udp-only",
    description: "TCP mode with UDP port",
    config: {
      image: "python:3-slim", httpPort: 8080, hasWebUI: false,
      tcpPorts: [],
      udpPorts: [{ name: "DNS_PORT", title: "DNS", containerPort: 5353, defaultValue: 5353 }],
      addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: Multiple ports (TCP + UDP) ---
  {
    name: "multi-ports",
    description: "TCP mode with 2 TCP + 1 UDP ports",
    config: {
      image: "python:3-slim", httpPort: 8080, hasWebUI: false,
      tcpPorts: [
        { name: "PORT_A", title: "Port A", containerPort: 9000, defaultValue: 9000 },
        { name: "PORT_B", title: "Port B", containerPort: 9001, defaultValue: 9001 },
      ],
      udpPorts: [{ name: "UDP_PORT", title: "UDP", containerPort: 5353, defaultValue: 5353 }],
      addons: ["localstorage"], version: "1.0.0",
    },
  },
  // --- NEW: BusyBox (minimal, tests shim + busybox addgroup) ---
  {
    name: "busybox",
    description: "BusyBox (no apt/apk, gosu shim + busybox addgroup)",
    config: {
      image: "busybox:latest", httpPort: 8000, hasWebUI: true,
      tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
    },
  },
];

function docker(args, opts = {}) {
  const result = spawnSync("docker", args, {
    timeout: opts.timeout || 30_000,
    stdio: ["ignore", "pipe", "pipe"],
    cwd: opts.cwd,
  });
  return {
    ok: result.status === 0,
    stdout: result.stdout?.toString().trim() || "",
    stderr: result.stderr?.toString().trim() || "",
  };
}

function runBuild(testCase) {
  const tag = `fastpack-test-${testCase.name}`;
  const dir = mkdtempSync(join(tmpdir(), `fastpack-build-${testCase.name}-`));

  try {
    // Generate files
    const dockerfile = generateDockerfile(testCase.config);
    const startsh = generateStartSh(testCase.config);
    writeFileSync(join(dir, "Dockerfile"), dockerfile);
    writeFileSync(join(dir, "start.sh"), startsh);
    writeFileSync(join(dir, ".dockerignore"), generateDockerignore());

    // Build
    const args = ["buildx", "build", "-t", tag, "."];
    if (NO_CACHE) args.splice(2, 0, "--no-cache");

    const buildResult = docker(args, { cwd: dir, timeout: TIMEOUT });

    if (!buildResult.ok) {
      return { success: false, error: buildResult.stderr || buildResult.stdout, tag, dir };
    }

    // Post-build checks
    if (RUN_CHECKS) {
      const checks = runChecks(tag, testCase.config, startsh);
      const failedChecks = checks.filter(c => !c.ok);
      if (failedChecks.length > 0) {
        docker(["rmi", tag]);
        return {
          success: false,
          error: failedChecks.map(c => `  CHECK FAIL: ${c.name} → ${c.detail}`).join("\n"),
          tag, dir,
        };
      }
    }

    // Cleanup
    docker(["rmi", tag]);
    return { success: true, tag, dir };
  } catch (err) {
    return { success: false, error: err.message, tag, dir };
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

function runChecks(tag, config, startsh) {
  const results = [];

  // 1. User cloudron exists with uid 808
  const userCheck = docker(["run", "--rm", tag, "id", "cloudron"]);
  results.push({
    name: "cloudron user (uid 808)",
    ok: userCheck.ok && userCheck.stdout.includes("808"),
    detail: userCheck.stdout || userCheck.stderr,
  });

  // 2. gosu works
  const gosuCheck = docker(["run", "--rm", tag, "/usr/local/bin/gosu", "cloudron:cloudron", "whoami"]);
  results.push({
    name: "gosu executes as cloudron",
    ok: gosuCheck.ok && gosuCheck.stdout.includes("cloudron"),
    detail: gosuCheck.stdout || gosuCheck.stderr,
  });

  // 3. start.sh is executable
  const startCheck = docker(["run", "--rm", tag, "test", "-x", "/app/code/start.sh"]);
  results.push({
    name: "start.sh executable",
    ok: startCheck.ok,
    detail: startCheck.ok ? "OK" : startCheck.stderr,
  });

  // 4. /app/code exists
  const codeCheck = docker(["run", "--rm", tag, "test", "-d", "/app/code"]);
  results.push({
    name: "/app/code exists",
    ok: codeCheck.ok,
    detail: codeCheck.ok ? "OK" : codeCheck.stderr,
  });

  // 5. TCP mode: python3 available
  if (!config.hasWebUI) {
    const pyCheck = docker(["run", "--rm", tag, "python3", "--version"]);
    results.push({
      name: "python3 available (TCP mode)",
      ok: pyCheck.ok && pyCheck.stdout.includes("Python"),
      detail: pyCheck.stdout || pyCheck.stderr,
    });
  }

  // 6. No localstorage: start.sh should NOT contain chown
  if (!config.addons || !config.addons.includes("localstorage")) {
    const hasChown = startsh.includes("chown");
    results.push({
      name: "no chown without localstorage",
      ok: !hasChown,
      detail: hasChown ? "start.sh contains chown but localstorage is not enabled" : "OK",
    });
  }

  return results;
}

// Main
const configs = FILTER
  ? TEST_CONFIGS.filter(c => c.name.includes(FILTER))
  : TEST_CONFIGS;

console.log(`FastPackCloudron — Docker Build Tests${RUN_CHECKS ? " + Post-build Checks" : ""}`);
console.log(`${"=".repeat(55)}\n`);

let passed = 0;
let failed = 0;

for (const testCase of configs) {
  process.stdout.write(`  ${testCase.name}: ${testCase.description} ... `);

  const result = runBuild(testCase);

  if (result.success) {
    console.log("\x1b[32mPASS\x1b[0m");
    passed++;
  } else {
    console.log("\x1b[31mFAIL\x1b[0m");
    console.error(`    ${result.error?.substring(0, 800)}`);
    failed++;
  }
}

console.log(`\n${passed}/${configs.length} tests passed`);
if (failed > 0) {
  console.log(`\x1b[31m${failed} FAILED\x1b[0m`);
  process.exit(1);
}
console.log("\x1b[32mAll tests passed!\x1b[0m");
