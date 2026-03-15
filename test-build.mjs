#!/usr/bin/env node

// test-build.mjs — Verifies generated Dockerfiles actually build with Docker.
// Usage: node test-build.mjs [--no-cache]
// Requires: Docker daemon running, Node.js 18+

import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  generateDockerfile,
  generateStartSh,
  generateDockerignore,
} from "./generators.js";

const NO_CACHE = process.argv.includes("--no-cache");
const TIMEOUT = 180_000; // 3 minutes per build (image pulls can be slow)

const TEST_CONFIGS = [
  {
    name: "debian-web",
    description: "Debian-based web app (apt-get path for gosu)",
    config: {
      image: "nginx:latest",
      httpPort: 8080,
      hasWebUI: true,
      tcpPorts: [],
      udpPorts: [],
      addons: ["localstorage"],
      version: "1.0.0",
    },
  },
  {
    name: "alpine-web",
    description: "Alpine-based web app (apk path for su-exec)",
    config: {
      image: "node:20-alpine",
      httpPort: 3000,
      hasWebUI: true,
      tcpPorts: [],
      udpPorts: [],
      addons: ["localstorage"],
      version: "1.0.0",
    },
  },
  {
    name: "tcp-only",
    description: "TCP-only service (Python healthcheck + gosu shim fallback)",
    config: {
      image: "eclipse-mosquitto:2.0",
      httpPort: 8080,
      hasWebUI: false,
      tcpPorts: [
        { name: "MQTT_PORT", title: "MQTT", containerPort: 1883, defaultValue: 1883 },
      ],
      udpPorts: [],
      addons: ["localstorage"],
      version: "1.0.0",
    },
  },
  {
    name: "slim-web",
    description: "Slim Debian web app (apt-get on slim image)",
    config: {
      image: "python:3-slim",
      httpPort: 8000,
      hasWebUI: true,
      tcpPorts: [],
      udpPorts: [],
      addons: ["localstorage"],
      version: "1.0.0",
    },
  },
];

function runBuild(testCase) {
  const tag = `fastpack-test-${testCase.name}`;
  const dir = mkdtempSync(join(tmpdir(), `fastpack-build-${testCase.name}-`));

  try {
    // Generate files
    writeFileSync(join(dir, "Dockerfile"), generateDockerfile(testCase.config));
    writeFileSync(join(dir, "start.sh"), generateStartSh(testCase.config));
    writeFileSync(join(dir, ".dockerignore"), generateDockerignore());

    // Build with docker buildx (spawnSync to check exit code, not throw on stderr)
    const args = ["buildx", "build", "-t", tag, "."];
    if (NO_CACHE) args.splice(2, 0, "--no-cache");

    const buildResult = spawnSync("docker", args, {
      cwd: dir,
      timeout: TIMEOUT,
      stdio: ["ignore", "pipe", "pipe"],
    });

    if (buildResult.status !== 0) {
      const stderr = buildResult.stderr?.toString() || "";
      const stdout = buildResult.stdout?.toString() || "";
      return { success: false, error: stderr || stdout || `exit code ${buildResult.status}` };
    }

    // Cleanup image
    spawnSync("docker", ["rmi", tag], { stdio: "ignore" });

    return { success: true };
  } catch (err) {
    return { success: false, error: err.message };
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}

// Run all tests
console.log("FastPackCloudron — Docker Build Tests");
console.log("=====================================\n");

let passed = 0;
let failed = 0;

for (const testCase of TEST_CONFIGS) {
  process.stdout.write(`  ${testCase.name}: ${testCase.description} ... `);

  const result = runBuild(testCase);

  if (result.success) {
    console.log("\x1b[32mPASS\x1b[0m");
    passed++;
  } else {
    console.log("\x1b[31mFAIL\x1b[0m");
    console.error(`    Error: ${result.error?.substring(0, 500)}`);
    failed++;
  }
}

console.log(`\n${passed}/${TEST_CONFIGS.length} builds passed`);
if (failed > 0) {
  console.log(`\x1b[31m${failed} FAILED\x1b[0m`);
  process.exit(1);
}
console.log("\x1b[32mAll builds passed!\x1b[0m");
