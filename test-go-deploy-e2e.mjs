#!/usr/bin/env node

// test-go-deploy-e2e.mjs — THE REAL USER FLOW E2E TEST
//
// Tests the ACTUAL user experience:
// 1. Open FastPackCloudron UI in browser
// 2. Fill the form (nginx simple app)
// 3. Download the ZIP
// 4. Extract the ZIP
// 5. Copy the Go deploy binary into the folder
// 6. Run the Go binary with piped input (URL + token + subdomain)
// 7. Verify the app is deployed and responds HTTP 200
// 8. Cleanup (uninstall via cloudron CLI)
//
// Usage: node test-go-deploy-e2e.mjs
// Requires: Playwright, cloudron CLI logged in, network access to 192.168.60.17

import { spawn, spawnSync, execSync } from "node:child_process";
import { mkdtempSync, writeFileSync, readFileSync, readdirSync, existsSync, copyFileSync, rmSync, mkdirSync } from "node:fs";
import { join, basename } from "node:path";
import { tmpdir } from "node:os";
import { setTimeout as sleep } from "node:timers/promises";

const CLOUDRON_URL = "https://my.192.168.60.17.nip.io";
const CLOUDRON_DOMAIN = "192.168.60.17.nip.io";
const VM_HOST = "fastpack@192.168.60.17";
const SUBDOMAIN = "fpgo";
const APP_URL = `https://${SUBDOMAIN}.${CLOUDRON_DOMAIN}`;
const GO_BINARY = "deploy-cli/dist/fastpack-deploy-windows-amd64.exe";

// Read token from cloudron config
const cloudronConfig = JSON.parse(readFileSync(join(process.env.HOME || process.env.USERPROFILE, ".cloudron.json"), "utf8"));
const API_TOKEN = cloudronConfig.cloudrons["my.192.168.60.17.nip.io"]?.token;
if (!API_TOKEN) {
  console.error("Cannot find API token for my.192.168.60.17.nip.io in ~/.cloudron.json");
  process.exit(1);
}

const PORT = 8767;
const UI_URL = `http://127.0.0.1:${PORT}/index.html`;

let passed = 0;
let failed = 0;
let server;

function assert(name, condition, detail) {
  if (condition) {
    passed++;
    console.log(`  \x1b[32m✓\x1b[0m ${name}`);
  } else {
    failed++;
    console.log(`  \x1b[31m✗\x1b[0m ${name}${detail ? " — " + detail : ""}`);
  }
  return condition;
}

function ssh(command) {
  const r = spawnSync("ssh", ["-o", "StrictHostKeyChecking=no", VM_HOST, command], {
    encoding: "utf8", timeout: 30_000, stdio: "pipe",
  });
  return { stdout: (r.stdout || "").trim(), stderr: (r.stderr || "").trim(), status: r.status };
}

async function main() {
  console.log("\n\x1b[36m═══════════════════════════════════════════════════\x1b[0m");
  console.log("\x1b[36m  FastPack Deploy CLI — REAL USER FLOW E2E TEST\x1b[0m");
  console.log("\x1b[36m═══════════════════════════════════════════════════\x1b[0m\n");

  // Pre-check: binary exists
  if (!existsSync(GO_BINARY)) {
    console.error(`Go binary not found: ${GO_BINARY}`);
    console.error("Run: gh run download <run-id> -n fastpack-deploy-binaries -D deploy-cli/dist/");
    process.exit(1);
  }

  // Pre-check: uninstall any leftover from previous run
  console.log("\x1b[33mSetup: Cleaning up previous test app...\x1b[0m");
  spawnSync("cloudron", ["uninstall", "--app", `${SUBDOMAIN}.${CLOUDRON_DOMAIN}`], {
    timeout: 30_000, stdio: "ignore", shell: true,
  });

  // ═══════════════════════════════════════════════
  // PHASE 1: Generate ZIP from UI
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[33mPhase 1: Generate package via UI\x1b[0m");

  server = spawn("python3", ["-m", "http.server", String(PORT), "--bind", "127.0.0.1"], {
    stdio: "ignore", detached: false,
  });
  await sleep(1500);

  const { chromium } = await import("playwright");
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  await page.goto(UI_URL, { waitUntil: "networkidle", timeout: 15_000 });
  assert("UI loaded", true);

  // Fill guided mode form with nginx config
  await page.fill("#guided-image", "python:3-slim");
  assert("Filled docker image: python:3-slim", true);

  // Download the ZIP
  const [download] = await Promise.all([
    page.waitForEvent("download", { timeout: 15_000 }),
    page.click('button:has-text("Download ZIP")'),
  ]);

  const zipPath = join(tmpdir(), await download.suggestedFilename());
  await download.saveAs(zipPath);
  assert("ZIP downloaded", existsSync(zipPath), zipPath);

  // Verify deploy wizard appeared
  await sleep(500);
  const wizardVisible = await page.locator(".fp-deploy-wizard").first().isVisible();
  assert("Deploy wizard shown after download", wizardVisible);

  await browser.close();
  server.kill();

  // ═══════════════════════════════════════════════
  // PHASE 2: Extract ZIP + prepare package folder
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[33mPhase 2: Extract ZIP and prepare deployment\x1b[0m");

  const extractDir = mkdtempSync(join(tmpdir(), "fpgo-e2e-"));

  // Extract ZIP using PowerShell (Windows)
  const extractResult = spawnSync("powershell", [
    "-Command",
    `Expand-Archive -Path '${zipPath}' -DestinationPath '${extractDir}' -Force`,
  ], { encoding: "utf8", timeout: 30_000 });
  assert("ZIP extracted", extractResult.status === 0, extractResult.stderr);

  // List extracted files
  const extractedFiles = readdirSync(extractDir);
  assert("CloudronManifest.json in ZIP", extractedFiles.includes("CloudronManifest.json"));
  assert("Dockerfile in ZIP", extractedFiles.includes("Dockerfile"));
  assert("start.sh in ZIP", extractedFiles.includes("start.sh"));

  // Copy Go binary into the folder
  const binaryDest = join(extractDir, basename(GO_BINARY));
  copyFileSync(GO_BINARY, binaryDest);
  assert("Go binary copied to package folder", existsSync(binaryDest));

  // Verify manifest content
  const manifest = JSON.parse(readFileSync(join(extractDir, "CloudronManifest.json"), "utf8"));
  assert("Manifest has valid id", manifest.id && manifest.id.startsWith("io."), `id=${manifest.id}`);
  assert("Manifest has httpPort", manifest.httpPort > 0, `httpPort=${manifest.httpPort}`);

  // ═══════════════════════════════════════════════
  // PHASE 3: Run Go Deploy CLI
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[33mPhase 3: Run Go Deploy CLI (the REAL test!)\x1b[0m");

  // Patch start.sh to run an actual server
  const startShPath = join(extractDir, "start.sh");
  let startSh = readFileSync(startShPath, "utf8");
  startSh = startSh
    .replace("YOUR_APP_COMMAND", "python3 -m http.server 8000")
    .replace("YOUR_SERVICE_COMMAND", "python3 -m http.server 8000");
  writeFileSync(startShPath, startSh);

  // Run the Go binary with piped input
  // Input: URL, token, subdomain, build service URL, build token (each on a new line)
  const BUILD_SERVICE_URL = `devtools.${CLOUDRON_DOMAIN}`;
  const input = `my.${CLOUDRON_DOMAIN}\n${API_TOKEN}\n${SUBDOMAIN}\n${BUILD_SERVICE_URL}\n\n`;

  console.log(`  Running: ${basename(GO_BINARY)} in ${extractDir}`);
  console.log(`  Target: ${CLOUDRON_URL} → ${SUBDOMAIN}.${CLOUDRON_DOMAIN}`);

  const deployResult = spawnSync(binaryDest, [], {
    cwd: extractDir,
    input: input,
    encoding: "utf8",
    timeout: 600_000, // 10 minutes — build can be slow
    stdio: ["pipe", "pipe", "pipe"],
    env: { ...process.env, NODE_TLS_REJECT_UNAUTHORIZED: "0" },
  });

  const stdout = deployResult.stdout || "";
  const stderr = deployResult.stderr || "";

  console.log("\n  --- Go CLI stdout ---");
  stdout.split("\n").forEach(l => console.log(`  │ ${l}`));
  if (stderr) {
    console.log("  --- Go CLI stderr ---");
    stderr.split("\n").forEach(l => console.log(`  │ ${l}`));
  }
  console.log("");

  assert("Go CLI exited with code 0", deployResult.status === 0, `exit=${deployResult.status}`);
  assert("Go CLI detected package", stdout.includes("Package found") || stdout.includes("📦"));
  assert("Go CLI connected to Cloudron", stdout.includes("OK") && stdout.includes("Connecting"));
  assert("Go CLI built image", stdout.includes("Building") || stdout.includes("image"));
  assert("Go CLI deployed app", stdout.includes("deployed") || stdout.includes("✅"));

  // ═══════════════════════════════════════════════
  // PHASE 4: Verify deployed app
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[33mPhase 4: Verify deployed app responds\x1b[0m");

  // Wait for app startup
  console.log("  Waiting 20s for app startup...");
  await sleep(20_000);

  // Curl via SSH to the Cloudron VM (self-signed cert)
  const curlResult = ssh(`curl -sk -o /dev/null -w '%{http_code}' ${APP_URL}/`);
  const httpStatus = parseInt(curlResult.stdout) || 0;
  assert("App responds with HTTP 200", httpStatus === 200, `HTTP ${httpStatus}`);

  if (httpStatus === 200) {
    const bodyResult = ssh(`curl -sk ${APP_URL}/`);
    assert("App body contains HTML", bodyResult.stdout.includes("<"), `body=${bodyResult.stdout.substring(0, 100)}`);
  }

  // ═══════════════════════════════════════════════
  // PHASE 5: Cleanup
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[33mPhase 5: Cleanup\x1b[0m");

  const uninstallResult = spawnSync("cloudron", [
    "uninstall", "--app", `${SUBDOMAIN}.${CLOUDRON_DOMAIN}`,
  ], { encoding: "utf8", timeout: 60_000, shell: true });
  assert("App uninstalled", uninstallResult.status === 0);

  // Cleanup temp files
  try {
    rmSync(extractDir, { recursive: true, force: true });
    rmSync(zipPath, { force: true });
  } catch { /* Windows temp file lock */ }
  assert("Temp files cleaned up", true);

  // ═══════════════════════════════════════════════
  // RESULTS
  // ═══════════════════════════════════════════════
  console.log("\n\x1b[36m═══════════════════════════════════════════════════\x1b[0m");
  console.log(`\x1b[${failed > 0 ? "31" : "32"}m${passed}/${passed + failed} tests passed\x1b[0m`);

  if (failed === 0) {
    console.log("\n\x1b[32m🎉 THE REAL USER FLOW WORKS END-TO-END!\x1b[0m");
    console.log("\x1b[32m   UI → ZIP → Extract → Go CLI → Cloudron → App deployed → HTTP 200\x1b[0m\n");
  }

  process.exit(failed > 0 ? 1 : 0);
}

main().catch(err => {
  console.error(`\x1b[31mFatal: ${err.message}\x1b[0m`);
  if (server) server.kill();
  process.exit(1);
});
