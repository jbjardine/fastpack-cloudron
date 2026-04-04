# FastPackCloudron Testing Improvement Requirements

**Date:** 2026-03-27
**Scope:** Test infrastructure, CI pipeline, and test coverage for the FastPackCloudron vanilla JS Cloudron package generator.

---

## Current State Summary

| Layer | File | What it does | Assertions / Scenarios |
|-------|------|-------------|----------------------|
| Unit (browser) | `test.html` | 22 suites testing all `generators.js` exports except `generateDeployCmd()` | 207 assertions |
| Unit CI runner | `test-ci.mjs` | Playwright headless runner for `test.html` | Pass/fail via DOM class |
| Build | `test-build.mjs` | Docker build + optional `--checks` for 11 configs | 11 build scenarios, 6 post-build checks |
| E2E | `test-cloudron.mjs` | Full Cloudron install/verify/uninstall cycle | 9 scenarios (local VM only) |
| CI | `.github/workflows/ci.yml` | Two parallel jobs: `unit-tests`, `build-tests` | No caching, no artifacts, no `--checks` |

---

## TIER 1: Quick Wins (each < 30 minutes)

---

### REQ-01: Enable `--checks` flag in CI build-tests job

**Gap:** `test-build.mjs` has 6 post-build checks (uid 808 verification, gosu execution, start.sh executable bit, /app/code existence, python3 availability in TCP mode, chown absence without localstorage) but CI runs `node test-build.mjs` without `--checks`. These checks only run when a developer remembers to pass the flag locally.

**What it catches that is currently missed:**
- Dockerfile generates a `useradd` command that creates uid 809 instead of 808 -- CI would not catch this.
- gosu symlink breaks on a new base image -- CI would not catch this.
- start.sh loses its executable bit due to a generator regression -- CI would not catch this.
- Python3 install fails on a new Alpine version in TCP mode -- CI would not catch this.

**File changes:**

1. `.github/workflows/ci.yml` -- change the build-tests run command:
   ```yaml
   # BEFORE (line 36):
   - name: Run build tests
     run: node test-build.mjs

   # AFTER:
   - name: Run build tests
     run: node test-build.mjs --checks
   ```

**CI time impact:** +30-60 seconds. Each of the 11 configs runs up to 6 `docker run --rm` commands. These are lightweight (no daemon, exec-and-exit) against already-built images.

**Acceptance criteria:**
- [ ] CI build-tests job invokes `node test-build.mjs --checks`
- [ ] A deliberate uid change (e.g., 809) in `generateDockerfile()` causes CI to fail on the "cloudron user (uid 808)" check
- [ ] CI logs show individual CHECK PASS/FAIL lines for each config

---

### REQ-02: Add unit tests for `generateDeployCmd()`

**Gap:** `generators.js` exports 12 functions. `test.html` imports and tests 11 of them. `generateDeployCmd()` (line 659 of generators.js) is exported, used in `app.js` (line 593: `zip.file('deploy.cmd', generateDeployCmd())`), but has zero test coverage. The function generates a Windows batch launcher script.

**What it catches that is currently missed:**
- Regression if someone changes the `@echo off` prefix or removes `%~dp0` relative path resolution.
- The `\r\n` line endings required for Windows `.cmd` files could be accidentally changed to `\n`.
- The `pause` command at the end (keeps window open) could be removed, breaking the user experience on Windows.

**File changes:**

1. `test.html` -- add import and test suite:

   Add `generateDeployCmd` to the import statement on line 26-37:
   ```javascript
   import {
     // ... existing imports ...
     generateDeploySh,
     generateDeployCmd,  // ADD THIS
   } from "./generators.js";
   ```

   Add a new test suite before the Summary section (before line 1074):
   ```javascript
   // ========================================
   // generateDeployCmd
   // ========================================
   addSuite("generateDeployCmd");

   const deployCmd = generateDeployCmd();
   assertIncludes("deployCmd: @echo off", deployCmd, "@echo off");
   assertIncludes("deployCmd: invokes deploy.js via node", deployCmd, 'node --no-deprecation "%~dp0deploy.js"');
   assertIncludes("deployCmd: passes arguments", deployCmd, "%*");
   assertIncludes("deployCmd: pause to keep window open", deployCmd, "pause");
   assertIncludes("deployCmd: uses CRLF line endings", deployCmd, "\r\n");
   ```

**CI time impact:** Zero. These are synchronous string assertions in the existing browser test run.

**Acceptance criteria:**
- [ ] `test.html` imports `generateDeployCmd` from `generators.js`
- [ ] At least 5 assertions covering: `@echo off`, node invocation with `%~dp0`, argument passthrough `%*`, `pause`, and CRLF line endings
- [ ] Total assertion count increases from 207 to at least 212
- [ ] All new assertions pass in both browser and CI (`node test-ci.mjs`)

---

### REQ-03: Add GitHub Actions caching for npm and Playwright

**Gap:** The `unit-tests` job runs `npm install playwright && npx playwright install chromium --with-deps` on every CI run. Playwright's Chromium binary is ~150 MB. With no caching, every push and PR re-downloads everything, adding 30-60 seconds of pure network time.

**What it catches that is currently missed:** This is not a coverage gap -- it is a CI velocity improvement that reduces feedback loop time, which directly supports TDD cycle speed.

**File changes:**

1. `.github/workflows/ci.yml` -- add caching steps to `unit-tests` job:

   ```yaml
   unit-tests:
     name: Unit Tests (Browser)
     runs-on: ubuntu-latest
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-node@v4
         with:
           node-version: "20"
           cache: "npm"
       - name: Cache Playwright browsers
         uses: actions/cache@v4
         id: playwright-cache
         with:
           path: ~/.cache/ms-playwright
           key: playwright-${{ runner.os }}-${{ hashFiles('package.json') }}
       - name: Install dependencies
         run: npm install
       - name: Install Playwright (if not cached)
         if: steps.playwright-cache.outputs.cache-hit != 'true'
         run: npx playwright install chromium --with-deps
       - name: Install Playwright system deps (if cached)
         if: steps.playwright-cache.outputs.cache-hit == 'true'
         run: npx playwright install-deps chromium
       - name: Run unit tests
         run: node test-ci.mjs
   ```

   Note: `setup-node` with `cache: "npm"` requires a `package-lock.json` in the repo (which currently exists as an untracked file per git status). The `package-lock.json` must be committed first.

**CI time impact:** First run: same as today. Subsequent runs: saves 20-40 seconds on npm install and 15-30 seconds on Playwright download (cache hit).

**Acceptance criteria:**
- [ ] `package-lock.json` is committed to the repository
- [ ] `actions/setup-node@v4` uses `cache: "npm"`
- [ ] Playwright browsers are cached with `actions/cache@v4` keyed on `package.json` hash
- [ ] On a cache-hit run, CI logs show "Cache restored" for both npm and Playwright
- [ ] Playwright system dependencies (`install-deps`) are still installed even on cache hit (binary cached, system libs are not)

---

### REQ-04: Upload test artifacts on CI failure

**Gap:** When CI fails, the only debugging information is whatever was printed to stdout/stderr. There are no build logs, no generated Dockerfiles, and no test output artifacts available for download from the GitHub Actions run.

**What it catches that is currently missed:** This is a debuggability improvement. Currently, a failed Docker build requires re-running locally to see the generated Dockerfile. A failed unit test requires re-running locally to see which assertions failed in what order.

**File changes:**

1. `.github/workflows/ci.yml` -- add artifact upload steps to both jobs:

   For `unit-tests` job, after the test run step:
   ```yaml
   - name: Upload test results on failure
     if: failure()
     uses: actions/upload-artifact@v4
     with:
       name: unit-test-results
       path: test-results/
       retention-days: 7
   ```

   For `build-tests` job, after the test run step:
   ```yaml
   - name: Upload build logs on failure
     if: failure()
     uses: actions/upload-artifact@v4
     with:
       name: build-test-logs
       path: build-test-output/
       retention-days: 7
   ```

2. `test-ci.mjs` -- write test output to a file in addition to stdout:
   ```javascript
   // After collecting summaryText (around line 41):
   import { mkdirSync, writeFileSync } from "node:fs";
   mkdirSync("test-results", { recursive: true });
   writeFileSync("test-results/summary.txt", summaryText);
   ```

3. `test-build.mjs` -- write per-config results to files on failure:
   ```javascript
   // In the failure branch of the main loop (around line 271-274):
   import { mkdirSync } from "node:fs";  // add to imports at top
   // Inside the failure block:
   mkdirSync("build-test-output", { recursive: true });
   writeFileSync(
     join("build-test-output", `${testCase.name}.log`),
     `Config: ${JSON.stringify(testCase.config, null, 2)}\n\nError:\n${result.error}`
   );
   ```

**CI time impact:** Zero on success (steps are conditional on `failure()`). On failure: 2-5 seconds for artifact upload.

**Acceptance criteria:**
- [ ] Both CI jobs have `upload-artifact@v4` steps with `if: failure()`
- [ ] `test-ci.mjs` writes `test-results/summary.txt` with the test summary
- [ ] `test-build.mjs` writes `build-test-output/{name}.log` per failed config, containing the config JSON and error output
- [ ] Artifacts are downloadable from the GitHub Actions run page when a job fails
- [ ] Artifacts have a 7-day retention to avoid storage bloat

---

## TIER 2: Medium Effort (1-2 hours each)

---

### REQ-05: Add security validation checks to build tests

**Gap:** The post-build checks in `test-build.mjs` (`runChecks()`, lines 193-248) verify that the cloudron user exists and gosu works, but they do not verify security-critical properties: that the container does not run as root, that no unexpected ports are exposed, and that sensitive paths are not world-writable.

**What it catches that is currently missed:**
- A Dockerfile regression that removes the `USER` switch, causing the container to run as root.
- An EXPOSE directive that leaks an internal-only port (e.g., a debug port).
- A directory permission change that makes `/app/data` world-writable.

**File changes:**

1. `test-build.mjs` -- extend `runChecks()` function with three new checks:

   ```javascript
   // 7. Container default user is not root
   const whoamiCheck = docker(["run", "--rm", "--entrypoint", "sh", tag, "-c",
     "id -u || echo 0"]);
   // Note: Cloudron overrides USER at runtime, but the Dockerfile should
   // not hardcode USER root. This checks the CMD runs in the default context.

   // 8. Only expected ports are exposed
   const inspectCheck = docker(["inspect", "--format",
     '{{json .Config.ExposedPorts}}', tag]);
   // Parse and verify only config.httpPort + declared tcp/udp ports appear.

   // 9. /app/code is not world-writable
   const permCheck = docker(["run", "--rm", tag, "stat", "-c", "%a", "/app/code"]);
   // Verify permissions are 755 or stricter (not 777).
   ```

2. `test-build.mjs` -- add per-config expected port sets to `TEST_CONFIGS` entries for validation:

   Each config already has `httpPort`, `tcpPorts`, and `udpPorts`. The check should compute the expected set from these fields and compare against `docker inspect` output.

**CI time impact:** +5-10 seconds total (3 additional `docker run`/`docker inspect` commands per config, but only when `--checks` is enabled).

**Acceptance criteria:**
- [ ] `runChecks()` includes at least 3 new checks: non-root default, expected-ports-only, directory permissions
- [ ] A Dockerfile that adds `EXPOSE 9999` (not in config) fails the expected-ports check
- [ ] A Dockerfile with `RUN chmod 777 /app/code` fails the permissions check
- [ ] All 11 existing configs pass the new security checks
- [ ] Check results appear in CI output alongside existing checks

---

### REQ-06: Add screenshot capture on unit test failure in CI

**Gap:** `test-ci.mjs` uses Playwright to run `test.html` headlessly. On failure, it prints `.fail` element text to stdout, but provides no visual evidence. Failed assertion output is often truncated or hard to parse from text logs alone.

**What it catches that is currently missed:** This is a debuggability improvement. A screenshot preserves the exact visual state of the test page at failure time, including any rendering issues, error dialogs, or unexpected page states that text extraction would miss.

**File changes:**

1. `test-ci.mjs` -- add screenshot capture in the failure branch:

   ```javascript
   // After line 56 (inside the "else" block where tests failed):
   import { mkdirSync } from "node:fs";
   mkdirSync("test-results", { recursive: true });
   await page.screenshot({
     path: "test-results/test-failure.png",
     fullPage: true,
   });
   console.log("Screenshot saved to test-results/test-failure.png");
   ```

2. `test-ci.mjs` -- also capture screenshot on exception:

   ```javascript
   // In the catch block (around line 59):
   // If page exists, take screenshot before closing
   if (typeof page !== "undefined" && page) {
     mkdirSync("test-results", { recursive: true });
     await page.screenshot({
       path: "test-results/test-error.png",
       fullPage: true,
     });
   }
   ```

   Note: The `page` variable must be declared outside the try block (move `let page;` before `try`) so it is accessible in the catch block.

3. `.github/workflows/ci.yml` -- the artifact upload step from REQ-04 already covers `test-results/`. No additional CI changes needed if REQ-04 is implemented first.

**CI time impact:** Zero on success. On failure: ~1 second for screenshot capture.

**Acceptance criteria:**
- [ ] On test failure, `test-results/test-failure.png` is created with a full-page screenshot
- [ ] On test error (exception), `test-results/test-error.png` is created
- [ ] Screenshot is included in the `unit-test-results` artifact (from REQ-04)
- [ ] Screenshot renders the full test results page (not just the visible viewport)
- [ ] `page` variable scoping allows screenshot capture in the catch block

---

### REQ-07: Expand base image coverage in build tests

**Gap:** `test-build.mjs` covers Debian (nginx, python:3-slim), Alpine (node:20-alpine, alpine:3.19), Fedora, and BusyBox. Missing coverage for: Ubuntu (different apt-get behavior from Debian in some edge cases), CentOS/Rocky (yum-based, distinct from dnf-based Fedora), and images with no shell (distroless-like, which should fail gracefully).

**What it catches that is currently missed:**
- Ubuntu-specific apt-get behavior divergence from Debian slim images.
- yum-based images (CentOS Stream, Rocky Linux) where `dnf` may not be the default.
- Graceful failure messaging when the base image has no shell or package manager.

**File changes:**

1. `test-build.mjs` -- add new entries to `TEST_CONFIGS` array:

   ```javascript
   // Ubuntu (apt-get path, distinct from Debian slim)
   {
     name: "ubuntu-web",
     description: "Ubuntu (apt-get path, full Debian variant)",
     config: {
       image: "ubuntu:24.04", httpPort: 8000, hasWebUI: true,
       tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
     },
   },
   // Rocky Linux (yum/dnf hybrid)
   {
     name: "rocky-web",
     description: "Rocky Linux 9 (dnf/yum path)",
     config: {
       image: "rockylinux:9-minimal", httpPort: 8000, hasWebUI: true,
       tcpPorts: [], udpPorts: [], addons: ["localstorage"], version: "1.0.0",
     },
   },
   ```

   Distroless images (e.g., `gcr.io/distroless/static-debian12`) will fail the Docker build because they have no shell. This is an expected failure that should be documented rather than tested as a passing case. A separate "expected-failure" test category could be added later.

**CI time impact:** +60-90 seconds (2 additional Docker builds). Ubuntu is a larger image pull than slim variants. Rocky Linux is comparable.

**Acceptance criteria:**
- [ ] `TEST_CONFIGS` includes at least `ubuntu-web` and `rocky-web` entries
- [ ] Ubuntu config builds and passes all `--checks` (uid 808, gosu, start.sh executable)
- [ ] Rocky Linux config builds and passes all `--checks`
- [ ] Total build-test config count increases from 11 to at least 13
- [ ] No existing test behavior changes

---

## TIER 3: Larger Improvements (half-day to multi-day)

---

### REQ-08: Runtime container validation

**Gap:** `test-build.mjs` verifies that containers build and that static properties are correct (user exists, file is executable), but never starts a container and validates that it actually serves HTTP responses. The `test-cloudron.mjs` E2E tests do this but require a Cloudron VM and are never run in CI.

**What it catches that is currently missed:**
- start.sh has a syntax error that prevents the container from starting.
- The healthcheck endpoint (port 8080 in TCP mode) does not actually respond.
- gosu fails at runtime (not just `gosu whoami` but actual process execution under cloudron user).
- nginx reverse proxy in multi-service mode fails to start or route correctly.

**File changes:**

1. `test-build.mjs` -- add a new `--runtime` flag and `runRuntimeCheck()` function:

   ```javascript
   const RUN_RUNTIME = process.argv.includes("--runtime");
   ```

   ```javascript
   function runRuntimeCheck(tag, config) {
     const containerName = `fp-runtime-${config.name || "test"}`;
     const results = [];

     try {
       // Start container in background
       const startArgs = ["run", "-d", "--name", containerName,
         "-p", `0:${config.httpPort}`, tag];
       const startResult = docker(startArgs, { timeout: 30_000 });
       if (!startResult.ok) {
         return [{ name: "container start", ok: false,
           detail: startResult.stderr }];
       }

       // Wait for container to be healthy (up to 15 seconds)
       // Poll with curl/wget against the mapped port
       const inspectResult = docker(["port", containerName,
         String(config.httpPort)]);
       const hostPort = inspectResult.stdout.split(":").pop();

       // HTTP check against healthcheck path
       // Use docker exec + curl or a Node http request against localhost:hostPort

       // Verify process runs as cloudron user (not root)
       const psCheck = docker(["exec", containerName, "ps", "-o", "user=",
         "-p", "1"]);

       // Check container logs for errors
       const logsCheck = docker(["logs", containerName]);

     } finally {
       // Always stop and remove
       docker(["stop", containerName], { timeout: 10_000 });
       docker(["rm", "-f", containerName], { timeout: 10_000 });
     }

     return results;
   }
   ```

2. `.github/workflows/ci.yml` -- add runtime validation as a separate job (or extend build-tests):

   ```yaml
   runtime-tests:
     name: Runtime Validation
     needs: build-tests
     runs-on: ubuntu-latest
     steps:
       - uses: actions/checkout@v4
       - uses: actions/setup-node@v4
         with:
           node-version: "20"
       - name: Run runtime checks
         run: node test-build.mjs --checks --runtime --filter debian-web
   ```

   Initially limit to a subset of configs (e.g., `debian-web`, `alpine-web`, `tcp-mosquitto`) to keep CI time reasonable.

**CI time impact:** +2-4 minutes depending on number of configs tested. Each container needs 5-15 seconds to start and respond. Recommend starting with 3 configs and expanding.

**Acceptance criteria:**
- [ ] `--runtime` flag triggers container start + HTTP response validation
- [ ] At least 3 configs are validated: one Debian web, one Alpine web, one TCP mode
- [ ] Each runtime check verifies: container starts, HTTP endpoint responds with 2xx, container does not crash within 10 seconds
- [ ] Failed runtime checks produce the container logs as output for debugging
- [ ] Container cleanup (stop + rm) is guaranteed in a finally block
- [ ] Runtime tests run in CI (as a separate job or gated step)

---

### REQ-09: Test coverage metrics and reporting

**Gap:** There is no way to know which `generators.js` functions or code paths are exercised by `test.html`. The 207 assertions cover 11 of 12 exports, but branch coverage within those functions is unknown. For example, `generateDockerfile()` has 5 conditional branches (apt-get/apk/dnf/shim/busybox), but it is unclear which branches are hit by which test configs.

**What it catches that is currently missed:**
- Dead code paths in generators that are never exercised.
- Conditional branches (e.g., the `else` fallback in gosu installation) that have no corresponding test.
- New generator logic added without corresponding tests.

**Implementation approach:**

Because the test suite is browser-based (`test.html` runs in Playwright via `test-ci.mjs`), traditional Node.js coverage tools (c8, nyc) do not apply directly. There are two viable approaches:

**Option A: Playwright coverage API (recommended)**

1. `test-ci.mjs` -- enable V8 coverage collection via Playwright's `page.coverage`:

   ```javascript
   await page.coverage.startJSCoverage();
   await page.goto(TEST_URL, { waitUntil: "networkidle" });
   await page.waitForSelector("#summary", { timeout: TIMEOUT });
   const coverage = await page.coverage.stopJSCoverage();

   // Filter to generators.js only
   const generatorsCov = coverage.find(c => c.url.includes("generators.js"));
   // Write raw coverage data
   writeFileSync("test-results/coverage.json", JSON.stringify(generatorsCov, null, 2));
   ```

2. Post-process coverage data into a summary (line/branch/function percentages).

3. `.github/workflows/ci.yml` -- add coverage artifact upload and optional threshold check.

**Option B: Port unit tests to Node.js**

Extract test logic from `test.html` into a `.mjs` file, run with `node --experimental-vm-modules` + c8 for standard coverage. This is a larger refactor but produces standard lcov output compatible with Codecov/Coveralls.

**CI time impact:** Option A: +2-3 seconds (coverage collection is nearly free in V8). Option B: significant refactor time, but no runtime overhead.

**Acceptance criteria:**
- [ ] Coverage data for `generators.js` is collected during CI unit test runs
- [ ] Coverage report includes line coverage and function coverage percentages
- [ ] Coverage data is uploaded as a CI artifact
- [ ] Baseline coverage percentage is documented (establishes a floor for future regressions)
- [ ] Optional: CI fails if coverage drops below the established baseline

---

### REQ-10: E2E test automation in CI

**Gap:** `test-cloudron.mjs` contains 9 comprehensive E2E scenarios but requires a Cloudron VM at `192.168.60.17`. These tests are never run in CI and depend on a specific developer's local environment. There is no automated way to validate end-to-end behavior on every commit.

**What it catches that is currently missed:**
- CloudronManifest.json fields that are syntactically valid but rejected by the Cloudron runtime.
- Container behavior differences between Docker-only and actual Cloudron execution (filesystem restrictions, port mapping, addon injection).
- Regression in SSO integration (proxyAuth, OIDC) that only manifests under actual Cloudron middleware.

**Implementation approach:**

Full Cloudron E2E in CI requires one of:

**Option A: Self-hosted runner with Cloudron VM (recommended for this project)**

1. Set up a persistent self-hosted GitHub Actions runner on the network with access to the Cloudron VM.
2. Add a new workflow job that runs on `self-hosted` with the `cloudron` label.
3. Gate the E2E job behind a manual trigger (`workflow_dispatch`) or schedule (`cron`) to avoid running on every push.

   ```yaml
   e2e-tests:
     name: Cloudron E2E
     runs-on: [self-hosted, cloudron]
     if: github.event_name == 'workflow_dispatch' || github.event.schedule
     steps:
       - uses: actions/checkout@v4
       - run: node test-cloudron.mjs
   ```

**Option B: Docker-in-Docker lightweight E2E**

Create a reduced E2E suite that uses `docker run` with port mapping (no Cloudron) to validate container startup, HTTP response, and signal handling. This provides ~70% of E2E value without the Cloudron infrastructure dependency.

1. New file: `test-runtime-e2e.mjs` -- builds, starts, curls, and stops containers.
2. Validates: HTTP response codes, healthcheck endpoint, graceful shutdown (SIGTERM handling).
3. Does NOT validate: Cloudron-specific behavior (addon injection, SSO, manifest parsing).

**CI time impact:**
- Option A: +5-10 minutes per run, but only on manual/scheduled triggers.
- Option B: +3-5 minutes per run, can run on every push.

**Acceptance criteria:**
- [ ] At least one form of E2E validation runs in CI (Option A or B)
- [ ] Option A: `test-cloudron.mjs` runs on a self-hosted runner with access to a Cloudron VM, triggered manually or on schedule
- [ ] Option B: A new `test-runtime-e2e.mjs` validates container start + HTTP response for at least 3 configs in CI on every push
- [ ] E2E failures produce downloadable artifacts (container logs, HTTP response bodies)
- [ ] E2E job does not block the main CI pipeline (separate job, allowed to fail on PRs)

---

## Supplementary Improvements (not prioritized into tiers)

---

### REQ-11: Decouple test-ci.mjs from DOM class name

**Current fragility:** `test-ci.mjs` line 43 checks `summaryClass === "all-pass"` to determine pass/fail. If the CSS class name in `test.html` changes (e.g., renamed to `success`), CI silently reports failure even when all tests pass.

**Recommended fix:** Add a `data-status` attribute to the `#summary` element in `test.html` that is explicitly either `"pass"` or `"fail"`, independent of styling. Have `test-ci.mjs` check `data-status` instead of the CSS class.

**File changes:**
1. `test.html` (line 1091-1094): Add `summary.dataset.status = "pass"` / `"fail"` alongside the className assignment.
2. `test-ci.mjs` (line 43): Change to `const status = await page.getAttribute("#summary", "data-status");` and check `status === "pass"`.

**Acceptance criteria:**
- [ ] `#summary` element has a `data-status` attribute set to `"pass"` or `"fail"`
- [ ] `test-ci.mjs` reads `data-status` instead of `class` for pass/fail determination
- [ ] CSS class names can be changed without breaking CI

---

## Implementation Order

The requirements have explicit dependencies. Implement in this order:

```
Phase 1 (single PR, < 1 hour):
  REQ-02  generateDeployCmd() tests       -- no dependencies
  REQ-01  Enable --checks in CI           -- no dependencies
  REQ-03  GitHub Actions caching          -- requires committing package-lock.json

Phase 2 (single PR, < 1 hour):
  REQ-04  Upload test artifacts on failure -- no dependencies
  REQ-06  Screenshot on failure            -- depends on REQ-04 for artifact upload
  REQ-11  Decouple from DOM class name     -- no dependencies

Phase 3 (separate PRs, 1-2 hours each):
  REQ-05  Security validation checks       -- depends on REQ-01 (--checks enabled)
  REQ-07  Expand base image coverage       -- depends on REQ-01 (--checks enabled)

Phase 4 (separate PRs, multi-day):
  REQ-08  Runtime container validation     -- depends on REQ-05 (security checks)
  REQ-09  Test coverage metrics            -- depends on REQ-06 (screenshot infra)
  REQ-10  E2E test automation              -- independent, but benefits from REQ-08
```

---

## CI Time Budget

| Requirement | Time Impact (per run) | Cumulative |
|-------------|----------------------|------------|
| Baseline (current) | ~3 min | 3 min |
| REQ-01 --checks | +45 sec | 3:45 |
| REQ-02 generateDeployCmd tests | +0 sec | 3:45 |
| REQ-03 Caching | -30 sec (after first run) | 3:15 |
| REQ-04 Artifacts | +0 sec (on success) | 3:15 |
| REQ-05 Security checks | +10 sec | 3:25 |
| REQ-06 Screenshot | +0 sec (on success) | 3:25 |
| REQ-07 Base image expansion | +75 sec | 4:40 |
| REQ-08 Runtime validation | +180 sec | 7:40 |
| REQ-09 Coverage metrics | +3 sec | 7:43 |
| REQ-10 E2E (if in-CI) | +300 sec (separate job) | 7:43 + 5:00 |

**Target total CI time:** Under 8 minutes for the main pipeline. E2E runs separately.
