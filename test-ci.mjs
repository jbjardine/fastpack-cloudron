#!/usr/bin/env node

// test-ci.mjs — Runs the in-browser unit tests headlessly via Playwright.
// Usage: node test-ci.mjs
// Requires: npx playwright install chromium --with-deps

import { spawn } from "node:child_process";
import { setTimeout as sleep } from "node:timers/promises";

const PORT = 8765;
const TEST_URL = `http://127.0.0.1:${PORT}/test.html`;
const TIMEOUT = 30_000;

async function run() {
  // Start HTTP server
  const server = spawn("python3", ["-m", "http.server", String(PORT), "--bind", "127.0.0.1"], {
    stdio: "ignore",
    detached: false,
  });

  server.on("error", () => {
    server.kill();
  });

  await sleep(1500);

  let exitCode = 1;

  try {
    const { chromium } = await import("playwright");

    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();

    await page.goto(TEST_URL, { waitUntil: "networkidle" });
    await page.waitForSelector("#summary", { timeout: TIMEOUT });

    const summaryText = await page.textContent("#summary");
    const dataStatus = await page.getAttribute("#summary", "data-status");

    console.log(`Test result: ${summaryText}`);

    if (dataStatus === "pass" && !summaryText.includes("FAILED")) {
      console.log("\x1b[32mAll unit tests passed!\x1b[0m");
      exitCode = 0;
    } else {
      console.log("\x1b[31mSome tests FAILED\x1b[0m");

      // Print failed test details
      const failElements = await page.locator(".fail").all();
      for (const el of failElements) {
        const text = await el.textContent();
        console.log(`  ${text}`);
      }
      exitCode = 1;
    }

    await browser.close();
  } catch (err) {
    console.error(`Error running tests: ${err.message}`);
    exitCode = 1;
  } finally {
    server.kill();
  }

  process.exit(exitCode);
}

run();
