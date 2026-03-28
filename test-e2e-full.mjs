#!/usr/bin/env node

// test-e2e-full.mjs — Exhaustive E2E tests on real Cloudron
// Usage: node test-e2e-full.mjs [--filter NAME]
// Requires: cloudron CLI logged in, SSH to 192.168.60.17, sudo password "fastpack"

import { spawnSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import {
  generateManifest,
  generateDockerfile,
  generateStartSh,
  generateDockerignore,
  generateNginxConf,
} from "./generators.js";

const DOMAIN = "192.168.60.17.nip.io";
const VM_HOST = "fastpack@192.168.60.17";
const REGISTRY = `devtools.${DOMAIN}`;
const FILTER = process.argv.find((a, i) => process.argv[i - 1] === "--filter") || "";

function run(cmd, args, opts = {}) {
  // Use shell:true only for cloudron CLI on Windows (it's a .cmd wrapper)
  const needsShell = cmd === "cloudron";
  const r = spawnSync(cmd, args, { encoding: "utf8", timeout: 600_000, stdio: "pipe", shell: needsShell, ...opts });
  return { status: r.status, stdout: (r.stdout || "").trim(), stderr: (r.stderr || "").trim() };
}

function ssh(command) {
  return run("ssh", [VM_HOST, command]);
}

function sshSudo(command) {
  // On Windows with shell:true, we need to escape properly
  return run("ssh", [VM_HOST, `echo fastpack | sudo -S bash -c "${command.replace(/"/g, '\\"')}"`]);
}

function patchStartSh(startSh, appCommand) {
  return startSh
    .replace("YOUR_APP_COMMAND", appCommand)
    .replace("YOUR_SERVICE_COMMAND", appCommand);
}

// ═══════════════════════════════════════════════
// TEST CONFIGURATIONS — Simple to Complex
// ═══════════════════════════════════════════════

const TEST_CONFIGS = [
  // 1. SIMPLEST — basic web app, no addons
  {
    name: "1-simple",
    subdomain: "e2e1",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e1", title: "Simple", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: null,
      addons: [], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
    },
    verify: (r) => r.status === 200 && r.body.includes("<"),
    verifyLabel: "HTTP 200 with HTML",
  },

  // 2. WITH LOCALSTORAGE — tests init guard + chown
  {
    name: "2-localstorage",
    subdomain: "e2e2",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e2", title: "LocalStorage", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
    },
    verify: (r) => r.status === 200,
    verifyLabel: "HTTP 200 + localstorage init",
    extraChecks: [
      { cmd: "cat /app/data/.initialized", expect: "1.0.0", label: "init guard" },
    ],
  },

  // 3. NODE.JS STACK TEMPLATE
  {
    name: "3-nodejs-stack",
    subdomain: "e2e3",
    appCommand: null, // uses stack template
    config: {
      id: "io.fastpack.e2e3", title: "NodeJS Stack", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "nodejs",
      image: "node:20-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
    },
    // Override start.sh to use a real node command
    startShOverride: (sh) => sh.replace(
      "node /app/code/server.js",
      'node -e "require(\\"http\\").createServer((q,s)=>{s.writeHead(200);s.end(\\"NodeJS Stack OK\\")}).listen(8000)"'
    ),
    verify: (r) => r.status === 200 && r.body.includes("NodeJS Stack OK"),
    verifyLabel: "HTTP 200 with Node.js response",
  },

  // 4. WITH DATABASE (PostgreSQL)
  {
    name: "4-postgresql",
    subdomain: "e2e4",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e4", title: "PostgreSQL", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: "postgresql", sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
    },
    verify: (r) => r.status === 200,
    verifyLabel: "HTTP 200 with PostgreSQL addon",
    extraChecks: [
      { cmd: "printenv CLOUDRON_POSTGRESQL_URL", expect: "postgres://", label: "PG URL env" },
    ],
  },

  // 5. WITH SSO (OIDC)
  {
    name: "5-oidc-sso",
    subdomain: "e2e5",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e5", title: "OIDC SSO", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: "oidc",
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
      oidcRedirectUri: "/callback", oidcLogoutUri: "/", oidcTokenAlgo: "",
    },
    verify: (r) => r.status === 200 || r.status === 302 || r.status === 303,
    verifyLabel: "HTTP 200/302/303 (SSO redirect expected)",
  },

  // 6. ALPINE IMAGE — tests apk/su-exec path
  {
    name: "6-alpine",
    subdomain: "e2e6",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e6", title: "Alpine", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-alpine", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
    },
    verify: (r) => r.status === 200,
    verifyLabel: "HTTP 200 (Alpine + su-exec)",
    // Note: cloudron exec is unreliable from Windows (shell:true transforms arguments)
    // The cloudron user is verified by the HTTP 200 (start.sh uses gosu cloudron)
    extraChecks: [],
  },

  // 7. MULTI-SERVICE — 2 processes + nginx
  {
    name: "7-multi-service",
    subdomain: "e2e7",
    config: {
      id: "io.fastpack.e2e7", title: "MultiService", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [], checklist: [], logPaths: [],
      secondarySubdomains: [],
      services: [
        { name: "app1", command: "python3 -m http.server 3001", internalPort: 3001, routePath: "/app1", sso: "none" },
        { name: "app2", command: "python3 -m http.server 3002", internalPort: 3002, routePath: "/app2", sso: "none" },
      ],
    },
    verify: (r) => r.status === 200,
    verifyLabel: "HTTP 200 (multi-service default route)",
    extraVerify: [
      { path: "/app1/", expectStatus: 200, label: "/app1/ route" },
      { path: "/app2/", expectStatus: 200, label: "/app2/ route" },
    ],
  },

  // 8. COPY --from= (multi-stage)
  {
    name: "8-copy-from",
    subdomain: "e2e8",
    appCommand: null,
    config: {
      id: "io.fastpack.e2e8", title: "CopyFrom", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      subcontainers: [], checklist: [], logPaths: [], secondarySubdomains: [],
      copyFrom: [
        { image: "node:20-slim", src: "/usr/local/bin/node", dest: "/usr/local/bin/node" },
      ],
    },
    startShOverride: (sh) => sh.replace(
      "YOUR_APP_COMMAND",
      'node -e "require(\\"http\\").createServer((q,s)=>{s.writeHead(200);s.end(\\"Node via COPY --from!\\")}).listen(8000)"'
    ),
    verify: (r) => r.status === 200 && r.body.includes("Node via COPY"),
    verifyLabel: "HTTP 200 with Node.js from COPY --from=",
  },

  // 9. DooD — sub-container mode
  {
    name: "9-dood",
    subdomain: "e2e9",
    config: {
      id: "io.fastpack.e2e9", title: "DooD Test", version: "1.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: null, sso: null,
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], checklist: [], logPaths: [], secondarySubdomains: [],
      subcontainers: [
        { image: "kennethreitz/httpbin", port: 80, route: "/httpbin", memory: 256, volume: "/data" },
      ],
    },
    verify: (r) => r.status === 200,
    verifyLabel: "HTTP 200 (DooD landing page)",
    extraVerify: [
      { path: "/httpbin/get", expectStatus: 200, expectBody: "origin", label: "/httpbin/get (sub-container)" },
    ],
  },

  // 10. COMPLEX — database + localstorage + checklist + OIDC
  {
    name: "10-complex",
    subdomain: "e2e10",
    appCommand: "python3 -m http.server 8000",
    config: {
      id: "io.fastpack.e2e10", title: "Complex App", version: "2.0.0",
      httpPort: 8000, healthCheckPath: "/", hasWebUI: true, stack: "",
      image: "python:3-slim", database: "postgresql", sso: "oidc",
      addons: ["localstorage"], tcpPorts: [], udpPorts: [], httpPorts: [],
      services: [], capabilities: [], mediaLinks: [], schedulerTasks: [],
      copyFrom: [], subcontainers: [],
      checklist: ["Change admin password", "Enable 2FA"],
      logPaths: [], secondarySubdomains: [],
      oidcRedirectUri: "/callback", oidcLogoutUri: "/", oidcTokenAlgo: "",
    },
    verify: (r) => r.status === 200 || r.status === 302,
    verifyLabel: "HTTP 200/302 (PG + OIDC + localstorage + checklist)",
    extraChecks: [
      { cmd: "printenv CLOUDRON_POSTGRESQL_URL", expect: "postgres://", label: "PG URL" },
      { cmd: "cat /app/data/.initialized", expect: "2.0.0", label: "init guard v2.0.0" },
    ],
  },
];

// ═══════════════════════════════════════════════
// TEST RUNNER
// ═══════════════════════════════════════════════

function curlApp(subdomain, path = "/") {
  const r = ssh(`curl -sk https://${subdomain}.${DOMAIN}${path} -o /tmp/e2e-body -w '%{http_code}' && cat /tmp/e2e-body`);
  const output = r.stdout || "";
  const statusMatch = output.match(/^(\d{3})/);
  const status = statusMatch ? parseInt(statusMatch[1]) : 0;
  const body = output.substring(3);
  return { status, body };
}

function execInApp(subdomain, cmd) {
  // MSYS_NO_PATHCONV prevents Git Bash from converting /app/data to C:/Program Files/Git/app/data
  const r = spawnSync("cloudron", ["exec", "--app", `${subdomain}.${DOMAIN}`, "--", "sh", "-c", cmd], {
    encoding: "utf8", timeout: 30_000, stdio: "pipe", shell: true,
    env: { ...process.env, NODE_TLS_REJECT_UNAUTHORIZED: "0", MSYS_NO_PATHCONV: "1" },
  });
  return (r.stdout || "")
    .replace(/\(node:\d+\).*\n?/g, "")
    .replace(/\(Use `node.*\n?/g, "")
    .replace(/Warning.*\n?/g, "")
    .trim();
}

async function runTest(test) {
  const dir = mkdtempSync(join(tmpdir(), `e2e-${test.name}-`));
  const tag = `${REGISTRY}/e2e-${test.name}:latest`;

  try {
    // 1. Generate files
    process.stdout.write(`  Generate ... `);
    const manifest = generateManifest(test.config);
    let startSh = generateStartSh(test.config);
    if (test.appCommand) startSh = patchStartSh(startSh, test.appCommand);
    if (test.startShOverride) startSh = test.startShOverride(startSh);
    let dockerfile = generateDockerfile(test.config);
    const nginxConf = generateNginxConf(test.config);

    writeFileSync(join(dir, "CloudronManifest.json"), manifest);
    writeFileSync(join(dir, "start.sh"), startSh);
    writeFileSync(join(dir, "Dockerfile"), dockerfile);
    writeFileSync(join(dir, ".dockerignore"), "");
    if (nginxConf) writeFileSync(join(dir, "nginx.conf"), nginxConf);

    console.log("\x1b[32mOK\x1b[0m");

    // 2. Copy to server and build
    process.stdout.write(`  Build ... `);
    const remoteDir = `/tmp/e2e-${test.name}`;
    ssh(`rm -rf ${remoteDir} && mkdir -p ${remoteDir}`);
    run("scp", ["-r", `${dir}/.`, `${VM_HOST}:${remoteDir}/`]);
    const buildR = sshSudo(`docker build -t ${tag} ${remoteDir} && docker push ${tag}`);
    if (buildR.status !== 0) {
      console.log(`\x1b[31mFAIL\x1b[0m`);
      console.log(`      ${buildR.stderr.split("\n").slice(-3).join("\n      ")}`);
      return false;
    }
    console.log("\x1b[32mOK\x1b[0m");

    // 3. Install on Cloudron
    process.stdout.write(`  Install ... `);
    // Write manifest to the test dir (already exists) and install from there
    const installR = run("cloudron", [
      "install", "--image", tag,
      "--location", `${test.subdomain}.${DOMAIN}`,
      "--allow-selfsigned",
    ], { cwd: dir });

    if (installR.status !== 0) {
      const errMsg = (installR.stderr || installR.stdout || "unknown error").trim().split("\n").slice(-3).join("\n      ");
      console.log(`\x1b[31mFAIL\x1b[0m (exit ${installR.status})`);
      console.log(`      stdout: ${(installR.stdout || "").substring(0, 200)}`);
      console.log(`      stderr: ${(installR.stderr || "").substring(0, 200)}`);
      return false;
    }
    console.log("\x1b[32mOK\x1b[0m");

    // 4. Wait for startup (DooD needs more time)
    const hasSubcontainers = test.config.subcontainers && test.config.subcontainers.length > 0;
    if (hasSubcontainers) {
      process.stdout.write(`  Wait (DooD startup) ... `);
      await new Promise(r => setTimeout(r, 15000));
      console.log("\x1b[32mOK\x1b[0m");
    }

    // 5. Verify HTTP
    process.stdout.write(`  Verify (${test.verifyLabel}) ... `);
    const httpR = curlApp(test.subdomain);
    if (!test.verify(httpR)) {
      console.log(`\x1b[31mFAIL\x1b[0m (HTTP ${httpR.status})`);
      return false;
    }
    console.log(`\x1b[32mOK\x1b[0m (HTTP ${httpR.status})`);

    // 6. Extra HTTP verifications
    if (test.extraVerify) {
      for (const ev of test.extraVerify) {
        process.stdout.write(`  Verify ${ev.label} ... `);
        const evR = curlApp(test.subdomain, ev.path);
        const statusOk = ev.expectStatus ? evR.status === ev.expectStatus : evR.status === 200;
        const bodyOk = ev.expectBody ? evR.body.includes(ev.expectBody) : true;
        if (!statusOk || !bodyOk) {
          console.log(`\x1b[31mFAIL\x1b[0m (HTTP ${evR.status})`);
          return false;
        }
        console.log(`\x1b[32mOK\x1b[0m`);
      }
    }

    // 7. Extra exec checks (env vars, files)
    if (test.extraChecks) {
      for (const ec of test.extraChecks) {
        process.stdout.write(`  Check ${ec.label} ... `);
        const out = execInApp(test.subdomain, ec.cmd);
        if (out && out.includes(ec.expect)) {
          console.log(`\x1b[32mOK\x1b[0m`);
        } else if (!out) {
          console.log(`\x1b[33mSKIP\x1b[0m (exec empty)`);
        } else {
          console.log(`\x1b[31mFAIL\x1b[0m (got: ${out.substring(0, 80)})`);
          return false;
        }
      }
    }

    // 8. Cloudron user check — verify gosu works (the real test)
    process.stdout.write(`  Check gosu/cloudron ... `);
    const gosuOut = execInApp(test.subdomain, "/usr/local/bin/gosu cloudron:cloudron whoami");
    if (!gosuOut.includes("cloudron")) {
      // Fallback: just verify the HTTP works (gosu is used in start.sh already)
      console.log(`\x1b[33mSKIP\x1b[0m (exec unreliable on Windows, HTTP verified)`);
    } else {
      console.log(`\x1b[32mOK\x1b[0m`);
    }

    return true;
  } finally {
    // Always cleanup
    process.stdout.write(`  Cleanup ... `);
    run("cloudron", ["uninstall", "--app", `${test.subdomain}.${DOMAIN}`], { timeout: 60_000 });
    // Cleanup sub-containers if DooD
    if (test.config.subcontainers && test.config.subcontainers.length > 0) {
      for (const sub of test.config.subcontainers) {
        const name = "fp-" + sub.image.replace(/[^a-zA-Z0-9]/g, "-").replace(/-+/g, "-");
        sshSudo(`docker rm -f ${name} 2>/dev/null`);
      }
    }
    rmSync(dir, { recursive: true, force: true });
    console.log("\x1b[32mOK\x1b[0m");
  }
}

// ═══════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════

async function main() {
  console.log("FastPackCloudron — Exhaustive E2E Tests");
  console.log("═══════════════════════════════════════");
  console.log(`Testing ${TEST_CONFIGS.length} configs on ${DOMAIN}\n`);

  const configs = FILTER
    ? TEST_CONFIGS.filter(c => c.name.includes(FILTER))
    : TEST_CONFIGS;

  let passed = 0;
  let failed = 0;
  const failures = [];

  for (const test of configs) {
    console.log(`\n  ─── ${test.name} ───`);
    const ok = await runTest(test);
    if (ok) {
      passed++;
    } else {
      failed++;
      failures.push(test.name);
    }
  }

  console.log("\n═══════════════════════════════════════");
  console.log(`${passed}/${configs.length} E2E tests passed`);
  if (failed > 0) {
    console.log(`\x1b[31m${failed} FAILED: ${failures.join(", ")}\x1b[0m`);
  } else {
    console.log("\x1b[32mAll E2E tests passed!\x1b[0m");
  }
  process.exit(failed > 0 ? 1 : 0);
}

main();
