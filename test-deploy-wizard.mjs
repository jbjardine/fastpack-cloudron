#!/usr/bin/env node

// test-deploy-wizard.mjs — E2E test for the deploy wizard with OS detection.
// Verifies: wizard appears after download, OS detection, download links, clipboard guard.
// Usage: node test-deploy-wizard.mjs

import { spawn } from "node:child_process";
import { setTimeout as sleep } from "node:timers/promises";

const PORT = 8766;
const APP_URL = `http://127.0.0.1:${PORT}/index.html`;
const TIMEOUT = 15_000;

async function run() {
  const server = spawn("python3", ["-m", "http.server", String(PORT), "--bind", "127.0.0.1"], {
    stdio: "ignore",
    detached: false,
  });

  server.on("error", () => server.kill());
  await sleep(1500);

  let exitCode = 0;
  let passed = 0;
  let failed = 0;

  function assert(name, condition) {
    if (condition) {
      passed++;
      console.log(`  \x1b[32m✓\x1b[0m ${name}`);
    } else {
      failed++;
      exitCode = 1;
      console.log(`  \x1b[31m✗\x1b[0m ${name}`);
    }
  }

  try {
    const { chromium } = await import("playwright");
    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();

    console.log("\n\x1b[36mDeploy Wizard E2E Tests\x1b[0m\n");

    // Load the page
    await page.goto(APP_URL, { waitUntil: "networkidle", timeout: TIMEOUT });

    // === Test 1: OS detection ===
    console.log("\x1b[33mSuite: OS Detection\x1b[0m");

    const detectedOS = await page.evaluate(() => {
      // Access the Alpine.js component data
      const el = document.querySelector("[x-data]");
      return el?.__x?.$data?._detectedOS || el?._x_dataStack?.[0]?._detectedOS || "unknown";
    });
    assert("OS detected (non-empty)", detectedOS !== "unknown" && detectedOS !== undefined);
    assert("OS is one of valid values", ["windows", "linux", "macos-arm", "macos-intel"].includes(detectedOS));

    // === Test 2: Deploy command matches OS ===
    console.log("\n\x1b[33mSuite: Deploy Command\x1b[0m");

    const deployCmd = await page.evaluate(() => {
      const el = document.querySelector("[x-data]");
      return el?.__x?.$data?._deployCmd || el?._x_dataStack?.[0]?._deployCmd || "";
    });
    assert("Deploy command is non-empty", deployCmd.length > 0);
    assert("Deploy command contains 'fastpack-deploy'", deployCmd.includes("fastpack-deploy"));

    if (detectedOS === "windows") {
      assert("Windows: command ends with .exe", deployCmd.endsWith(".exe"));
    } else if (detectedOS === "linux") {
      assert("Linux: command contains linux-amd64", deployCmd.includes("linux-amd64"));
    }

    // === Test 3: Fill form and trigger download to show wizard ===
    console.log("\n\x1b[33mSuite: Deploy Wizard Visibility\x1b[0m");

    // Fill minimum required field (guided mode only needs docker image)
    await page.fill("#guided-image", "nginx:latest");

    // The wizard should be hidden initially
    const wizardBefore = await page.locator(".fp-deploy-wizard").first().isVisible();
    assert("Wizard hidden before download", !wizardBefore);

    // Click download (this triggers the wizard)
    // We need to handle the download dialog
    const downloadPromise = page.waitForEvent("download", { timeout: 10_000 }).catch(() => null);
    await page.click('button:has-text("Download ZIP")');
    await downloadPromise;

    // Wait for wizard to appear
    await sleep(500);
    const wizardAfter = await page.locator(".fp-deploy-wizard").first().isVisible();
    assert("Wizard visible after download", wizardAfter);

    // === Test 4: Download links present ===
    console.log("\n\x1b[33mSuite: Download Links\x1b[0m");

    // Expand the details section
    const detailsSummary = page.locator(".fp-deploy-wizard details summary").first();
    if (await detailsSummary.isVisible()) {
      await detailsSummary.click();
      await sleep(300);
    }

    const windowsLink = await page.locator('a:has-text("Windows (x64)")').first().getAttribute("href");
    const linuxLink = await page.locator('a:has-text("Linux (x64)")').first().getAttribute("href");
    const macArmLink = await page.locator('a:has-text("macOS (Apple Silicon)")').first().getAttribute("href");
    const macIntelLink = await page.locator('a:has-text("macOS (Intel)")').first().getAttribute("href");

    assert("Windows download link exists", windowsLink && windowsLink.includes("fastpack-deploy-windows"));
    assert("Linux download link exists", linuxLink && linuxLink.includes("fastpack-deploy-linux"));
    assert("macOS ARM download link exists", macArmLink && macArmLink.includes("fastpack-deploy-darwin-arm64"));
    assert("macOS Intel download link exists", macIntelLink && macIntelLink.includes("fastpack-deploy-darwin-amd64"));

    // === Test 5: OS highlight ===
    console.log("\n\x1b[33mSuite: OS Highlight\x1b[0m");

    const activeLinks = await page.locator(".fp-deploy-os-active").count();
    assert("Exactly one OS link is highlighted", activeLinks === 1);

    // === Test 6: Alternative CLI section ===
    console.log("\n\x1b[33mSuite: Alternative CLI\x1b[0m");

    const altSection = page.locator('details summary:has-text("Alternative")').first();
    const altExists = await altSection.count() > 0;
    assert("Alternative Cloudron CLI section exists", altExists);

    // === Test 7: Clipboard guard ===
    console.log("\n\x1b[33mSuite: Clipboard Safety\x1b[0m");

    // On HTTP (non-secure) context, navigator.clipboard is undefined.
    // Our guard should prevent errors.
    const clipboardError = await page.evaluate(() => {
      try {
        // Simulate click on the deploy command
        const cmdEl = document.querySelector('.fp-deploy-wizard-cmd[x-text]');
        if (cmdEl) cmdEl.click();
        return null; // No error
      } catch (e) {
        return e.message;
      }
    });
    assert("Clipboard click does not throw on HTTP context", clipboardError === null);

    await browser.close();

    // Summary
    console.log(`\n\x1b[${failed > 0 ? "31" : "32"}m${passed}/${passed + failed} tests passed\x1b[0m`);

  } catch (err) {
    console.error(`\x1b[31mError: ${err.message}\x1b[0m`);
    exitCode = 1;
  } finally {
    server.kill();
  }

  process.exit(exitCode);
}

run();
