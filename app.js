// app.js — UI logic for FastPackCloudron

import {
  sanitizeImageName,
  humanizeImageName,
  generateManifest,
  generateDockerfile,
  generateStartSh,
  generateDescription,
  generateDockerignore,
  generateReadme,
  generateCloudronVersions,
  generateNginxConf,
  generateDeploySh,
  generateDeployCmd,
  generatePostInstall,
  generateChangelog,
  SAFE_DOCKER_REF,
  SAFE_PATH,
  SAFE_IDENTIFIER,
  SAFE_VERSION,
  SAFE_ROUTE_PATH,
} from './generators.js';

/**
 * Reads the form state and builds a config object with smart defaults.
 */
function buildConfig() {
  const imageEl = document.getElementById('docker-image');
  const idEl = document.getElementById('app-id');
  const titleEl = document.getElementById('app-title');
  const versionEl = document.getElementById('app-version');
  const httpPortEl = document.getElementById('http-port');
  const healthCheckPathEl = document.getElementById('health-check-path');
  const stackEl = document.getElementById('app-stack');
  const hasWebUI = document.getElementById('web-ui-yes').checked;
  const databaseEl = document.getElementById('database');
  const ssoEl = document.getElementById('sso');
  const authorEl = document.getElementById('app-author');
  const taglineEl = document.getElementById('app-tagline');
  const descriptionEl = document.getElementById('app-description');
  const oidcRedirectUriEl = document.getElementById('oidc-redirect-uri');
  const oidcLogoutUriEl = document.getElementById('oidc-logout-uri');
  const oidcTokenAlgoEl = document.getElementById('oidc-token-algo');
  const websiteEl = document.getElementById('app-website');
  const contactEmailEl = document.getElementById('app-contact-email');
  const configurePathEl = document.getElementById('app-configure-path');
  const upstreamVersionEl = document.getElementById('app-upstream-version');
  const postInstallMessageEl = document.getElementById('app-post-install-message');
  const changelogEl = document.getElementById('app-changelog');
  const iconEl = document.getElementById('app-icon');
  const memoryLimitEl = document.getElementById('app-memory-limit');

  // Publishing fields
  const dockerHubUsernameEl = document.getElementById('docker-hub-username');
  const packagerNameEl = document.getElementById('packager-name');
  const packagerUrlEl = document.getElementById('packager-url');
  const iconUrlEl = document.getElementById('icon-url');
  const mediaLinksEl = document.getElementById('media-links');
  const documentationUrlEl = document.getElementById('documentation-url');
  const forumUrlEl = document.getElementById('forum-url');

  // Advanced fields
  const minBoxVersionEl = document.getElementById('min-box-version');
  const maxBoxVersionEl = document.getElementById('max-box-version');
  const targetBoxVersionEl = document.getElementById('target-box-version');
  const multiDomainEl = document.getElementById('multi-domain');
  const fullDomainEl = document.getElementById('full-domain');
  const singleUserEl = document.getElementById('single-user');
  const dockerfileCloudronEl = document.getElementById('dockerfile-cloudron');
  const secondarySubdomainsEl = document.getElementById('secondary-subdomains');
  const runtimeDirsEl = document.getElementById('runtime-dirs');
  const persistentDirsEl = document.getElementById('persistent-dirs');
  const backupCommandEl = document.getElementById('backup-command');
  const restoreCommandEl = document.getElementById('restore-command');
  const logPathsEl = document.getElementById('log-paths');
  const checklistEl = document.getElementById('checklist');

  // Localstorage sub-options
  const localstorageFtpEl = document.getElementById('localstorage-ftp');
  const localstorageSqliteEl = document.getElementById('localstorage-sqlite');
  const localstorageSqlitePathsEl = document.getElementById('localstorage-sqlite-paths');

  // ProxyAuth options
  const proxyauthPathEl = document.getElementById('proxyauth-path');
  const proxyauthBasicAuthEl = document.getElementById('proxyauth-basic-auth');
  const proxyauthBearerAuthEl = document.getElementById('proxyauth-bearer-auth');

  // Database-specific options
  const mysqlMultipleDbsEl = document.getElementById('mysql-multiple-dbs');
  const mongodbOplogEl = document.getElementById('mongodb-oplog');
  const redisNoPasswordEl = document.getElementById('redis-no-password');
  const postgresqlLocaleEl = document.getElementById('postgresql-locale');

  // Sendmail options
  const sendmailOptionalEl = document.getElementById('sendmail-optional');
  const sendmailDisplayNameEl = document.getElementById('sendmail-display-name');
  const sendmailValidCertEl = document.getElementById('sendmail-valid-cert');

  const image = imageEl.value.trim();

  // Auto-generate id if user hasn't manually edited
  let id;
  if (idEl.dataset.userEdited !== 'true') {
    id = image ? `io.fastpack.${sanitizeImageName(image)}` : '';
  } else {
    id = idEl.value.trim();
  }

  // Auto-generate title if user hasn't manually edited
  let title;
  if (titleEl.dataset.userEdited !== 'true') {
    title = image ? humanizeImageName(image) : '';
  } else {
    title = titleEl.value.trim();
  }

  const version = versionEl.value.trim() || '1.0.0';
  const httpPort = hasWebUI
    ? parseInt(httpPortEl.value, 10) || 8000
    : 8000;
  const healthCheckPath = healthCheckPathEl.value.trim() || '/';

  const databaseVal = databaseEl.value;
  const database = databaseVal === '' ? null : databaseVal;

  const ssoVal = ssoEl.value;
  const sso = ssoVal === '' ? null : ssoVal;

  // Collect checked addon checkboxes
  const addons = [];
  const addonCheckboxes = document.querySelectorAll('.addon-checkbox:checked');
  for (const cb of addonCheckboxes) {
    addons.push(cb.value);
  }

  // Collect TCP port rows
  const tcpPorts = [];
  const tcpRows = document.querySelectorAll('.tcp-port-row');
  for (const row of tcpRows) {
    tcpPorts.push({
      name: row.querySelector('.port-name').value.trim(),
      title: row.querySelector('.port-title').value.trim(),
      containerPort: parseInt(row.querySelector('.port-container').value, 10) || 0,
      defaultValue: parseInt(row.querySelector('.port-default').value, 10) || 0,
    });
  }

  // Collect UDP port rows
  const udpPorts = [];
  const udpRows = document.querySelectorAll('.udp-port-row');
  for (const row of udpRows) {
    udpPorts.push({
      name: row.querySelector('.port-name').value.trim(),
      title: row.querySelector('.port-title').value.trim(),
      containerPort: parseInt(row.querySelector('.port-container').value, 10) || 0,
      defaultValue: parseInt(row.querySelector('.port-default').value, 10) || 0,
    });
  }

  // Collect HTTP port rows (extra HTTP services with separate subdomains)
  const httpPorts = [];
  const httpRows = document.querySelectorAll('.http-port-row');
  for (const row of httpRows) {
    httpPorts.push({
      name: row.querySelector('.port-name').value.trim(),
      title: row.querySelector('.port-title').value.trim(),
      containerPort: parseInt(row.querySelector('.port-container').value, 10) || 0,
      defaultValue: parseInt(row.querySelector('.port-default').value, 10) || 0,
    });
  }

  // Collect selected tags
  const tags = [];
  const tagCheckboxes = document.querySelectorAll('.tag-checkbox:checked');
  for (const cb of tagCheckboxes) {
    tags.push(cb.value);
  }

  // Collect capabilities
  const capabilities = [];
  const capCheckboxes = document.querySelectorAll('.capability-checkbox:checked');
  for (const cb of capCheckboxes) {
    capabilities.push(cb.value);
  }

  // Collect scheduler task rows
  const schedulerTasks = [];
  const taskRows = document.querySelectorAll('.scheduler-task-row');
  for (const row of taskRows) {
    schedulerTasks.push({
      name: row.querySelector('.task-name').value.trim(),
      schedule: row.querySelector('.task-schedule').value.trim(),
      command: row.querySelector('.task-command').value.trim(),
    });
  }

  // Parse memoryLimit (MB -> bytes)
  const memoryLimitMB = parseInt(memoryLimitEl.value, 10) || 0;
  const memoryLimit = memoryLimitMB > 0 ? memoryLimitMB * 1024 * 1024 : 0;

  // Collect copy-from rows (multi-stage COPY --from=)
  const copyFrom = [];
  const copyFromRows = document.querySelectorAll('.copy-from-row');
  for (const row of copyFromRows) {
    copyFrom.push({
      image: row.querySelector('.copy-from-image').value.trim(),
      src: row.querySelector('.copy-from-src').value.trim(),
      dest: row.querySelector('.copy-from-dest').value.trim(),
    });
  }

  // Collect service rows
  const services = [];
  const serviceRows = document.querySelectorAll('.service-row');
  for (const row of serviceRows) {
    services.push({
      name: row.querySelector('.service-name').value.trim(),
      command: row.querySelector('.service-command').value.trim(),
      internalPort: parseInt(row.querySelector('.service-port').value, 10) || 0,
      routePath: row.querySelector('.service-route').value.trim() || null,
      sso: row.querySelector('.service-sso').value,
    });
  }

  // Parse comma-separated directory lists
  const parseDirs = (el) => el.value.trim().split(',').map(s => s.trim()).filter(s => s.length > 0);

  return {
    image,
    id,
    title,
    version,
    httpPort,
    healthCheckPath,
    hasWebUI,
    stack: stackEl.value,
    database,
    sso,
    addons,
    tcpPorts,
    udpPorts,
    httpPorts,
    author: authorEl.value.trim(),
    tagline: taglineEl.value.trim(),
    description: descriptionEl.value.trim(),
    oidcRedirectUri: oidcRedirectUriEl.value.trim() || '/auth/openid/callback',
    oidcLogoutUri: oidcLogoutUriEl.value.trim() || '/',
    oidcTokenAlgo: oidcTokenAlgoEl.value || '',
    website: websiteEl.value.trim(),
    contactEmail: contactEmailEl.value.trim(),
    tags,
    configurePath: configurePathEl.value.trim(),
    upstreamVersion: upstreamVersionEl.value.trim(),
    postInstallMessage: postInstallMessageEl.value.trim(),
    changelog: changelogEl.value.trim(),
    icon: iconEl.value.trim(),
    memoryLimit,
    // Publishing
    dockerHubUsername: dockerHubUsernameEl.value.trim(),
    packagerName: packagerNameEl.value.trim(),
    packagerUrl: packagerUrlEl.value.trim(),
    iconUrl: iconUrlEl.value.trim(),
    mediaLinks: mediaLinksEl.value.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0),
    documentationUrl: documentationUrlEl.value.trim(),
    forumUrl: forumUrlEl.value.trim(),
    // Advanced
    minBoxVersion: minBoxVersionEl.value.trim(),
    maxBoxVersion: maxBoxVersionEl.value.trim(),
    targetBoxVersion: targetBoxVersionEl.value.trim(),
    capabilities,
    multiDomain: multiDomainEl.checked,
    fullDomain: fullDomainEl.checked,
    singleUser: singleUserEl.checked,
    dockerfileCloudron: dockerfileCloudronEl.checked,
    secondarySubdomains: parseDirs(secondarySubdomainsEl),
    runtimeDirs: parseDirs(runtimeDirsEl),
    persistentDirs: parseDirs(persistentDirsEl),
    backupCommand: backupCommandEl.value.trim(),
    restoreCommand: restoreCommandEl.value.trim(),
    logPaths: parseDirs(logPathsEl),
    checklist: checklistEl.value.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0),
    // ProxyAuth options
    proxyauthPath: proxyauthPathEl.value.trim(),
    proxyauthBasicAuth: proxyauthBasicAuthEl.checked,
    proxyauthBearerAuth: proxyauthBearerAuthEl.checked,
    // Database-specific options
    mysqlMultipleDbs: mysqlMultipleDbsEl.checked,
    mongodbOplog: mongodbOplogEl.checked,
    redisNoPassword: redisNoPasswordEl.checked,
    postgresqlLocale: postgresqlLocaleEl.value.trim(),
    // Localstorage sub-options
    localstorageFtp: localstorageFtpEl.checked,
    localstorageSqlite: localstorageSqliteEl.checked,
    localstorageSqlitePaths: parseDirs(localstorageSqlitePathsEl),
    // Sendmail options
    sendmailOptional: sendmailOptionalEl.checked,
    sendmailDisplayName: sendmailDisplayNameEl.checked,
    sendmailValidCert: sendmailValidCertEl.checked,
    // Scheduler tasks
    schedulerTasks,
    // Copy from image (multi-stage)
    copyFrom: copyFrom.filter(cf => cf.image && cf.src && cf.dest),
    // Multi-service
    services,
  };
}

/**
 * Toggles visibility of an element by ID based on a condition.
 */
function toggleVisibility(elementId, visible) {
  document.getElementById(elementId).style.display = visible ? '' : 'none';
}

/**
 * Validates a config object. Returns { errors: [], warnings: [] }.
 * Each error/warning is { field, message }.
 */
function validate(config) {
  const errors = [];
  const warnings = [];

  // image is required
  if (!config.image) {
    errors.push({
      field: 'docker-image',
      message: 'Enter a Docker image (e.g., nginx:latest)',
    });
  } else if (!SAFE_DOCKER_REF.test(config.image)) {
    errors.push({
      field: 'docker-image',
      message: 'Invalid image name. Use only letters, digits, dots, hyphens, slashes, colons, and @sha256 digests.',
    });
  }

  // version format (interpolated into shell scripts)
  if (config.version && !SAFE_VERSION.test(config.version)) {
    errors.push({
      field: 'app-version',
      message: 'Invalid version. Use only letters, digits, dots, and hyphens.',
    });
  }

  // COPY --from= validation
  if (config.copyFrom && config.copyFrom.length > 0) {
    for (const cf of config.copyFrom) {
      if (cf.image && !SAFE_DOCKER_REF.test(cf.image)) {
        errors.push({
          field: 'copy-from',
          message: `Invalid COPY --from image "${cf.image}". Use only letters, digits, dots, hyphens, slashes, colons.`,
        });
        break;
      }
      if (cf.src && !SAFE_PATH.test(cf.src)) {
        errors.push({
          field: 'copy-from',
          message: `Invalid source path "${cf.src}". Use only letters, digits, dots, hyphens, slashes.`,
        });
        break;
      }
      if (cf.dest && !SAFE_PATH.test(cf.dest)) {
        errors.push({
          field: 'copy-from',
          message: `Invalid destination path "${cf.dest}". Use only letters, digits, dots, hyphens, slashes.`,
        });
        break;
      }
    }
  }

  // Service name validation (interpolated into shell echo and nginx config)
  if (config.services && config.services.length > 0) {
    for (const svc of config.services) {
      if (svc.name && !SAFE_IDENTIFIER.test(svc.name)) {
        errors.push({
          field: 'services',
          message: `Invalid service name "${svc.name}". Start with a letter, then letters, digits, hyphens, underscores.`,
        });
        break;
      }
      if (svc.routePath && !SAFE_ROUTE_PATH.test(svc.routePath)) {
        errors.push({
          field: 'services',
          message: `Invalid route path "${svc.routePath}". Must start with / and contain only letters, digits, dots, hyphens, slashes.`,
        });
        break;
      }
    }
  }

  // id format
  if (config.id && !/^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$/.test(config.id)) {
    errors.push({
      field: 'app-id',
      message: 'Expected format: com.example.myapp',
    });
  }

  // httpPort range
  if (config.httpPort < 1 || config.httpPort > 65535 || isNaN(config.httpPort)) {
    errors.push({
      field: 'http-port',
      message: 'Invalid port',
    });
  }

  // healthCheckPath must start with /
  if (config.healthCheckPath && !config.healthCheckPath.startsWith('/')) {
    errors.push({
      field: 'health-check-path',
      message: 'Must start with /',
    });
  }

  // Validate port names (must be valid env var identifiers)
  const allPorts = [...config.tcpPorts, ...config.udpPorts];
  for (const port of allPorts) {
    if (port.name && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(port.name)) {
      errors.push({
        field: 'ports',
        message: `Invalid port name "${port.name}": use letters, digits, underscores (no leading digit)`,
      });
      break;
    }
  }

  // Port conflicts — TCP and UDP are separate namespaces
  const usedTcpPorts = new Set();
  usedTcpPorts.add(config.httpPort); // httpPort is TCP
  let hasPortConflict = false;

  for (const port of config.tcpPorts) {
    if (port.containerPort && usedTcpPorts.has(port.containerPort)) {
      hasPortConflict = true;
    }
    if (port.containerPort) usedTcpPorts.add(port.containerPort);
  }

  const usedUdpPorts = new Set();
  for (const port of config.udpPorts) {
    if (port.containerPort && usedUdpPorts.has(port.containerPort)) {
      hasPortConflict = true;
    }
    if (port.containerPort) usedUdpPorts.add(port.containerPort);
  }

  if (hasPortConflict) {
    errors.push({
      field: 'ports',
      message: 'This port is already used',
    });
  }

  // Validate scheduler task names
  for (const task of config.schedulerTasks) {
    if (task.name && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(task.name)) {
      errors.push({
        field: 'scheduler',
        message: `Invalid task name "${task.name}": use letters, digits, underscores`,
      });
      break;
    }
  }

  // Validate URLs (warning if not https://)
  const urlFields = [
    { value: config.packagerUrl, label: 'Packager URL' },
    { value: config.iconUrl, label: 'Icon URL' },
    { value: config.documentationUrl, label: 'Documentation URL' },
    { value: config.forumUrl, label: 'Forum URL' },
  ];
  for (const f of urlFields) {
    if (f.value && !f.value.startsWith('https://')) {
      warnings.push({ message: `${f.label} should use https://` });
    }
  }
  for (const link of config.mediaLinks) {
    if (link && !link.startsWith('https://')) {
      warnings.push({ message: 'Media links should use https://' });
      break;
    }
  }

  // Validate minBoxVersion format
  if (config.minBoxVersion && !/^\d+\.\d+\.\d+$/.test(config.minBoxVersion)) {
    warnings.push({ message: 'Minimum Cloudron version should be semver (e.g., 7.0.0)' });
  }

  // Warning: TCP mode (no web UI) with no ports
  if (
    !config.hasWebUI &&
    config.tcpPorts.length === 0 &&
    config.udpPorts.length === 0
  ) {
    warnings.push({
      message: 'Your app exposes no service. Add at least one TCP port.',
    });
  }

  // Warning: no localstorage addon
  if (!config.addons.includes('localstorage')) {
    warnings.push({
      message:
        'Without localstorage, your app cannot persist data. Are you sure?',
    });
  }

  // Warning: capabilities selected
  if (config.capabilities.length > 0) {
    warnings.push({
      message: 'Capabilities grant elevated privileges. Only use if your app requires them.',
    });
  }

  // Smart image warnings
  const imageLower = (config.image || '').toLowerCase();

  if (imageLower.includes('distroless') || imageLower.includes('scratch')) {
    warnings.push({
      message: 'This image has no shell. start.sh requires /bin/sh.',
    });
  }

  if (imageLower.includes('busybox') && !config.hasWebUI) {
    warnings.push({
      message: 'BusyBox has no python3. TCP healthcheck will fail.',
    });
  }

  if (config.httpPort === 80 && imageLower.includes('nginx')) {
    warnings.push({
      message: 'nginx needs writable /var/cache/nginx. Add runtime dirs or use a different port.',
    });
  }

  // Warning: multi-service complexity
  if (config.services && config.services.length >= 2) {
    warnings.push({
      message: 'Multi-service apps run all processes in one container. Consider separate Cloudron apps for independent services (easier updates, isolation, debugging).',
    });
  }

  // Warning: COPY --from= Alpine/Debian mismatch
  if (config.copyFrom && config.copyFrom.length > 0) {
    const baseIsAlpine = imageLower.includes('alpine');
    for (const cf of config.copyFrom) {
      const cfLower = (cf.image || '').toLowerCase();
      const cfIsAlpine = cfLower.includes('alpine');
      if (baseIsAlpine !== cfIsAlpine && (baseIsAlpine || cfIsAlpine)) {
        warnings.push({
          message: `COPY --from=${cf.image}: mixing Alpine (musl) and Debian (glibc) images. Compiled binaries may not run. Use matching distros.`,
        });
        break;
      }
    }
  }

  if (/fedora|centos|rocky|amazon/.test(imageLower)) {
    warnings.push({
      message: 'This image uses dnf. gosu will be installed via util-linux/setpriv.',
    });
  }

  return { errors, warnings };
}

let _debounceTimer = null;

/**
 * Debounced wrapper around updatePreviewNow.
 */
function updatePreview() {
  clearTimeout(_debounceTimer);
  _debounceTimer = setTimeout(updatePreviewNow, 150);
}

/**
 * Rebuilds the preview panes and shows validation messages.
 */
function updatePreviewNow() {
  const config = buildConfig();
  const result = validate(config);

  // Clear all field-error spans
  const fieldErrors = document.querySelectorAll('.field-error');
  for (const el of fieldErrors) {
    el.textContent = '';
  }

  // Set field-level errors
  for (const err of result.errors) {
    const el = document.querySelector(`.field-error[data-error-for="${err.field}"]`);
    if (el) {
      el.textContent = err.message;
    }
  }

  // Clear and populate warnings
  const warningsContainer = document.getElementById('warnings');
  warningsContainer.replaceChildren();
  for (const warn of result.warnings) {
    const div = document.createElement('div');
    div.className = 'warning';
    div.textContent = warn.message;
    warningsContainer.appendChild(div);
  }

  // Update previews via textContent
  document.getElementById('preview-manifest').textContent =
    generateManifest(config);
  document.getElementById('preview-dockerfile').textContent =
    generateDockerfile(config);
  document.getElementById('preview-startsh').textContent =
    generateStartSh(config);
  document.getElementById('preview-dockerignore').textContent =
    generateDockerignore();
  document.getElementById('preview-readme').textContent =
    generateReadme(config);
  document.getElementById('preview-versions').textContent =
    generateCloudronVersions(config);

  // nginx preview — show tab and content when services are configured
  const hasServices = config.services && config.services.length > 0;
  const nginxTab = document.getElementById('nginx-tab');
  if (nginxTab) {
    nginxTab.style.display = hasServices ? '' : 'none';
  }
  document.getElementById('preview-nginx').textContent =
    hasServices ? generateNginxConf(config) : '';

  // Show/hide HTTP port group based on web UI selection
  toggleVisibility('http-port-group', config.hasWebUI);

  // Show/hide SSO-specific options
  toggleVisibility('oidc-redirect-group', config.sso === 'oidc');
  toggleVisibility('oidc-logout-group', config.sso === 'oidc');
  toggleVisibility('oidc-token-algo-group', config.sso === 'oidc');
  toggleVisibility('proxyauth-options-group', config.sso === 'proxyAuth');

  // Show/hide database-specific options
  toggleVisibility('mysql-options-group', config.database === 'mysql');
  toggleVisibility('mongodb-options-group', config.database === 'mongodb');
  toggleVisibility('redis-options-group', config.database === 'redis');
  toggleVisibility('postgresql-options-group', config.database === 'postgresql');

  // Show/hide addon-specific options
  toggleVisibility('sendmail-options-group', config.addons.includes('sendmail'));
  toggleVisibility('scheduler-options-group', config.addons.includes('scheduler'));
  toggleVisibility('localstorage-options-group', config.addons.includes('localstorage'));
  toggleVisibility('localstorage-sqlite-paths-group', config.localstorageSqlite);

  // Update auto-generated fields if not user-edited
  const idEl = document.getElementById('app-id');
  if (idEl.dataset.userEdited !== 'true') {
    idEl.value = config.id;
  }

  const titleEl = document.getElementById('app-title');
  if (titleEl.dataset.userEdited !== 'true') {
    titleEl.value = config.title;
  }
}

/**
 * Copies text content of a preview pane to clipboard.
 */
function copyPreview(panelId) {
  const el = document.getElementById(panelId);
  if (!el) return;
  navigator.clipboard.writeText(el.textContent).then(function () {
    // Flash the copy button
    const btn = document.querySelector(`.copy-btn[data-copy="${panelId}"]`);
    if (btn) {
      const original = btn.textContent;
      btn.textContent = 'Copied!';
      setTimeout(function () { btn.textContent = original; }, 1200);
    }
  });
}

/**
 * Validates, builds the ZIP, and triggers download.
 */
async function downloadZip() {
  const btn = document.getElementById('download-btn');
  const config = buildConfig();
  const result = validate(config);

  // Show errors if any
  const errorsContainer = document.getElementById('errors');
  errorsContainer.replaceChildren();

  if (result.errors.length > 0) {
    for (const err of result.errors) {
      const div = document.createElement('div');
      div.className = 'error';
      div.textContent = err.message;
      errorsContainer.appendChild(div);
    }
    return;
  }

  // Disable button during ZIP generation
  btn.disabled = true;
  btn.setAttribute('aria-busy', 'true');
  btn.textContent = 'Generating...';

  try {
    // Build ZIP using JSZip (loaded as global from CDN)
    const zip = new JSZip();
    zip.file('CloudronManifest.json', generateManifest(config));
    const dockerfileName = config.dockerfileCloudron ? 'Dockerfile.cloudron' : 'Dockerfile';
    zip.file(dockerfileName, generateDockerfile(config));
    zip.file('start.sh', generateStartSh(config));
    zip.file('.dockerignore', generateDockerignore());
    zip.file('README.md', generateReadme(config));
    zip.file('CloudronVersions.json', generateCloudronVersions(config));

    // Add DESCRIPTION.md if description is provided
    const descContent = generateDescription(config);
    if (descContent) {
      zip.file('DESCRIPTION.md', descContent);
    }

    // Add nginx.conf if services are configured
    if (config.services && config.services.length > 0) {
      zip.file('nginx.conf', generateNginxConf(config));
    }

    // Add icon file if uploaded
    const iconFileInput = document.getElementById('app-icon-file');
    const iconError = document.getElementById('err-icon-file');
    if (iconError) iconError.textContent = '';
    if (iconFileInput && iconFileInput.files.length > 0) {
      const iconFile = iconFileInput.files[0];
      const iconData = await iconFile.arrayBuffer();
      zip.file('icon.png', iconData);
    }

    // Add publishing stubs
    zip.file('POSTINSTALL.md', generatePostInstall(config));
    zip.file('CHANGELOG.md', generateChangelog(config));

    // Add cross-platform deploy script (Node.js — works on Windows, Linux, Mac)
    zip.file('deploy.js', generateDeploySh());
    // Add Windows double-click launcher
    zip.file('deploy.cmd', generateDeployCmd());

    const blob = await zip.generateAsync({ type: 'blob' });
    const filename = `${sanitizeImageName(config.image) || 'cloudron-app'}-cloudron.zip`;
    saveAs(blob, filename);
  } finally {
    btn.disabled = false;
    btn.removeAttribute('aria-busy');
    btn.textContent = 'Download ZIP';
  }
}

/**
 * Adds a port row (TCP or UDP) to the appropriate container.
 */
function addPortRow(type) {
  const container = document.getElementById(`${type}-ports-list`);

  const row = document.createElement('div');
  row.className = `${type}-port-row port-row`;

  const nameInput = document.createElement('input');
  nameInput.type = 'text';
  nameInput.className = 'port-name';
  nameInput.placeholder = 'Name (e.g., MQTT_PORT)';

  const titleInput = document.createElement('input');
  titleInput.type = 'text';
  titleInput.className = 'port-title';
  titleInput.placeholder = 'Title';

  const containerInput = document.createElement('input');
  containerInput.type = 'number';
  containerInput.className = 'port-container';
  containerInput.placeholder = 'Container port';

  const defaultInput = document.createElement('input');
  defaultInput.type = 'number';
  defaultInput.className = 'port-default';
  defaultInput.placeholder = 'Default port';

  const removeBtn = document.createElement('button');
  removeBtn.type = 'button';
  removeBtn.className = 'remove-port';
  removeBtn.textContent = '\u2715';

  // Wire input events for live preview (with debounce)
  nameInput.addEventListener('input', updatePreview);
  titleInput.addEventListener('input', updatePreview);
  containerInput.addEventListener('input', updatePreview);
  defaultInput.addEventListener('input', updatePreview);

  // Remove button
  removeBtn.addEventListener('click', function () {
    row.remove();
    updatePreview();
  });

  row.appendChild(nameInput);
  row.appendChild(titleInput);
  row.appendChild(containerInput);
  row.appendChild(defaultInput);
  row.appendChild(removeBtn);

  container.appendChild(row);
}

/**
 * Adds a scheduler task row to the scheduler tasks container.
 */
function addSchedulerTaskRow() {
  const container = document.getElementById('scheduler-tasks-list');

  const row = document.createElement('div');
  row.className = 'scheduler-task-row port-row';

  const nameInput = document.createElement('input');
  nameInput.type = 'text';
  nameInput.className = 'task-name';
  nameInput.placeholder = 'Task name';

  const scheduleInput = document.createElement('input');
  scheduleInput.type = 'text';
  scheduleInput.className = 'task-schedule';
  scheduleInput.placeholder = '*/5 * * * *';

  const commandInput = document.createElement('input');
  commandInput.type = 'text';
  commandInput.className = 'task-command';
  commandInput.placeholder = '/app/code/task.sh';

  const removeBtn = document.createElement('button');
  removeBtn.type = 'button';
  removeBtn.className = 'remove-port';
  removeBtn.textContent = '\u2715';

  nameInput.addEventListener('input', updatePreview);
  scheduleInput.addEventListener('input', updatePreview);
  commandInput.addEventListener('input', updatePreview);

  removeBtn.addEventListener('click', function () {
    row.remove();
    updatePreview();
  });

  row.appendChild(nameInput);
  row.appendChild(scheduleInput);
  row.appendChild(commandInput);
  row.appendChild(removeBtn);

  container.appendChild(row);
}

/**
 * Adds a copy-from row (COPY --from= multi-stage source).
 */
function addCopyFromRow() {
  const container = document.getElementById('copy-from-list');

  const row = document.createElement('div');
  row.className = 'copy-from-row service-row';

  const imageInput = document.createElement('input');
  imageInput.type = 'text';
  imageInput.className = 'copy-from-image';
  imageInput.placeholder = 'Image (e.g., registry:2)';

  const srcInput = document.createElement('input');
  srcInput.type = 'text';
  srcInput.className = 'copy-from-src';
  srcInput.placeholder = 'Source path (e.g., /bin/registry)';

  const destInput = document.createElement('input');
  destInput.type = 'text';
  destInput.className = 'copy-from-dest';
  destInput.placeholder = 'Dest path (e.g., /usr/local/bin/registry)';

  const removeBtn = document.createElement('button');
  removeBtn.type = 'button';
  removeBtn.className = 'remove-port';
  removeBtn.textContent = '\u2715';

  imageInput.addEventListener('input', updatePreview);
  srcInput.addEventListener('input', updatePreview);
  destInput.addEventListener('input', updatePreview);

  removeBtn.addEventListener('click', function () {
    row.remove();
    updatePreview();
  });

  row.appendChild(imageInput);
  row.appendChild(srcInput);
  row.appendChild(destInput);
  row.appendChild(removeBtn);

  container.appendChild(row);
}

/**
 * Adds a service row to the services container.
 */
function addServiceRow() {
  const container = document.getElementById('services-list');

  const row = document.createElement('div');
  row.className = 'service-row';

  const nameInput = document.createElement('input');
  nameInput.type = 'text';
  nameInput.className = 'service-name';
  nameInput.placeholder = 'Name (e.g., n8n)';

  const commandInput = document.createElement('input');
  commandInput.type = 'text';
  commandInput.className = 'service-command';
  commandInput.placeholder = 'Command (e.g., n8n start)';

  const portInput = document.createElement('input');
  portInput.type = 'number';
  portInput.className = 'service-port';
  portInput.placeholder = 'Port';

  const routeInput = document.createElement('input');
  routeInput.type = 'text';
  routeInput.className = 'service-route';
  routeInput.placeholder = 'Route (e.g., /n8n)';

  const ssoSelect = document.createElement('select');
  ssoSelect.className = 'service-sso';
  const noneOpt = document.createElement('option');
  noneOpt.value = 'none';
  noneOpt.textContent = 'No SSO';
  const proxyOpt = document.createElement('option');
  proxyOpt.value = 'proxyAuth';
  proxyOpt.textContent = 'ProxyAuth';
  ssoSelect.appendChild(noneOpt);
  ssoSelect.appendChild(proxyOpt);

  const removeBtn = document.createElement('button');
  removeBtn.type = 'button';
  removeBtn.className = 'remove-port';
  removeBtn.textContent = '\u2715';

  nameInput.addEventListener('input', updatePreview);
  commandInput.addEventListener('input', updatePreview);
  portInput.addEventListener('input', updatePreview);
  routeInput.addEventListener('input', updatePreview);
  ssoSelect.addEventListener('change', updatePreview);

  removeBtn.addEventListener('click', function () {
    row.remove();
    updatePreview();
  });

  row.appendChild(nameInput);
  row.appendChild(commandInput);
  row.appendChild(portInput);
  row.appendChild(routeInput);
  row.appendChild(ssoSelect);
  row.appendChild(removeBtn);

  container.appendChild(row);
}

/**
 * Marks a field as user-edited so auto-generation skips it.
 */
function markUserEdited(e) {
  e.target.dataset.userEdited = 'true';
}

/**
 * Wire everything up once the DOM is ready.
 */
document.addEventListener('DOMContentLoaded', function () {
  // input on text/number inputs -> updatePreview (debounced)
  const textAndNumberInputs = document.querySelectorAll(
    'input[type="text"], input[type="number"], textarea'
  );
  for (const input of textAndNumberInputs) {
    input.addEventListener('input', updatePreview);
  }

  // change on selects -> updatePreview
  const selects = document.querySelectorAll('select');
  for (const select of selects) {
    select.addEventListener('change', updatePreview);
  }

  // change on checkboxes -> updatePreview
  const checkboxes = document.querySelectorAll('input[type="checkbox"]');
  for (const cb of checkboxes) {
    cb.addEventListener('change', updatePreview);
  }

  // change on radios -> updatePreview
  const radios = document.querySelectorAll('input[type="radio"]');
  for (const radio of radios) {
    radio.addEventListener('change', updatePreview);
  }

  // input on #app-id and #app-title -> markUserEdited
  document.getElementById('app-id').addEventListener('input', markUserEdited);
  document.getElementById('app-title').addEventListener('input', markUserEdited);

  // Add port buttons
  document.getElementById('add-tcp-port').addEventListener('click', function () {
    addPortRow('tcp');
  });
  document.getElementById('add-udp-port').addEventListener('click', function () {
    addPortRow('udp');
  });
  document.getElementById('add-http-port').addEventListener('click', function () {
    addPortRow('http');
  });

  // Add scheduler task button
  document.getElementById('add-scheduler-task').addEventListener('click', function () {
    addSchedulerTaskRow();
  });

  // Add copy-from button
  document.getElementById('add-copy-from').addEventListener('click', function () {
    addCopyFromRow();
  });

  // Add service button
  document.getElementById('add-service').addEventListener('click', function () {
    addServiceRow();
  });

  // Download button
  document.getElementById('download-btn').addEventListener('click', downloadZip);

  // Copy buttons
  const copyBtns = document.querySelectorAll('.copy-btn');
  for (const btn of copyBtns) {
    btn.addEventListener('click', function () {
      copyPreview(btn.dataset.copy);
    });
  }

  // Map tab targets to preview element IDs
  const tabToCopyId = {
    manifest: 'preview-manifest',
    dockerfile: 'preview-dockerfile',
    startsh: 'preview-startsh',
    dockerignore: 'preview-dockerignore',
    readme: 'preview-readme',
    versions: 'preview-versions',
    nginx: 'preview-nginx',
  };

  // Preview tab switching (ARIA tabs pattern with arrow key navigation)
  const tabs = document.querySelectorAll('.preview-tab');
  const copyBtn = document.querySelector('.copy-btn');

  function activateTab(tab) {
    // Deactivate all tabs
    for (const t of tabs) {
      t.classList.remove('active');
      t.setAttribute('aria-selected', 'false');
      t.setAttribute('tabindex', '-1');
    }
    // Activate selected tab
    tab.classList.add('active');
    tab.setAttribute('aria-selected', 'true');
    tab.setAttribute('tabindex', '0');
    tab.focus();

    // Toggle preview content panels
    const target = tab.dataset.target;
    const panels = document.querySelectorAll('.preview-content');
    for (const panel of panels) {
      if (panel.dataset.panel === target) {
        panel.classList.add('active');
        panel.removeAttribute('hidden');
      } else {
        panel.classList.remove('active');
        panel.setAttribute('hidden', '');
      }
    }

    // Update copy button target
    if (copyBtn && tabToCopyId[target]) {
      copyBtn.dataset.copy = tabToCopyId[target];
      copyBtn.textContent = 'Copy';
    }
  }

  for (const tab of tabs) {
    tab.addEventListener('click', function () {
      activateTab(tab);
    });

    // Arrow key navigation within tablist
    tab.addEventListener('keydown', function (e) {
      const visibleTabs = Array.from(tabs).filter(t => t.style.display !== 'none');
      const idx = visibleTabs.indexOf(tab);
      let newIdx = -1;
      if (e.key === 'ArrowRight') newIdx = (idx + 1) % visibleTabs.length;
      else if (e.key === 'ArrowLeft') newIdx = (idx - 1 + visibleTabs.length) % visibleTabs.length;
      else if (e.key === 'Home') newIdx = 0;
      else if (e.key === 'End') newIdx = visibleTabs.length - 1;
      if (newIdx >= 0) {
        e.preventDefault();
        activateTab(visibleTabs[newIdx]);
      }
    });
  }

  // Icon file validation
  document.getElementById('app-icon-file').addEventListener('change', function () {
    const errEl = document.getElementById('err-icon-file');
    errEl.textContent = '';
    const file = this.files[0];
    if (!file) return;
    if (!file.type.startsWith('image/png')) {
      errEl.textContent = 'Icon must be a PNG file';
      return;
    }
    const img = new Image();
    img.onload = function () {
      if (img.width !== 256 || img.height !== 256) {
        errEl.textContent = `Icon must be 256x256 (got ${img.width}x${img.height})`;
      }
      URL.revokeObjectURL(img.src);
    };
    img.src = URL.createObjectURL(file);
  });

  // --- localStorage persistence ---
  const STORAGE_KEY = 'fastpack-cloudron-config';

  // Fields to persist (simple inputs/selects/textareas by ID)
  const persistIds = [
    'docker-image', 'app-id', 'app-title', 'app-version', 'app-upstream-version',
    'app-author', 'app-tagline', 'app-description', 'app-website', 'app-contact-email',
    'app-configure-path', 'app-post-install-message', 'app-changelog', 'app-icon',
    'health-check-path', 'http-port', 'app-memory-limit',
    'app-stack', 'database', 'sso',
    'oidc-redirect-uri', 'oidc-logout-uri', 'oidc-token-algo',
    'proxyauth-path',
    'docker-hub-username', 'packager-name', 'packager-url', 'icon-url', 'media-links',
    'documentation-url', 'forum-url',
    'min-box-version', 'max-box-version', 'target-box-version',
    'runtime-dirs', 'persistent-dirs', 'backup-command', 'restore-command',
    'secondary-subdomains', 'log-paths', 'checklist',
    'postgresql-locale', 'localstorage-sqlite-paths',
  ];

  // Checkboxes to persist by ID
  const persistCheckboxes = [
    'web-ui-yes', 'web-ui-no', 'multi-domain', 'full-domain', 'single-user',
    'proxyauth-basic-auth', 'proxyauth-bearer-auth',
    'mysql-multiple-dbs', 'mongodb-oplog', 'redis-no-password',
    'sendmail-optional', 'sendmail-display-name', 'sendmail-valid-cert',
    'localstorage-ftp', 'localstorage-sqlite', 'dockerfile-cloudron',
  ];

  function saveToLocalStorage() {
    try {
      const data = {};
      for (const id of persistIds) {
        const el = document.getElementById(id);
        if (el) data[id] = el.value;
      }
      for (const id of persistCheckboxes) {
        const el = document.getElementById(id);
        if (el) data['cb:' + id] = el.checked;
      }
      // Persist addon and tag checkboxes
      data['addons'] = Array.from(document.querySelectorAll('.addon-checkbox:checked')).map(c => c.value);
      data['tags'] = Array.from(document.querySelectorAll('.tag-checkbox:checked')).map(c => c.value);
      data['capabilities'] = Array.from(document.querySelectorAll('.capability-checkbox:checked')).map(c => c.value);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
    } catch (e) { /* ignore storage errors */ }
  }

  function loadFromLocalStorage() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const data = JSON.parse(raw);
      for (const id of persistIds) {
        const el = document.getElementById(id);
        if (el && data[id] !== undefined) el.value = data[id];
      }
      for (const id of persistCheckboxes) {
        const el = document.getElementById(id);
        if (el && data['cb:' + id] !== undefined) el.checked = data['cb:' + id];
      }
      // Restore addon/tag/capability checkboxes
      if (data['addons']) {
        for (const cb of document.querySelectorAll('.addon-checkbox')) {
          cb.checked = data['addons'].includes(cb.value);
        }
      }
      if (data['tags']) {
        for (const cb of document.querySelectorAll('.tag-checkbox')) {
          cb.checked = data['tags'].includes(cb.value);
        }
      }
      if (data['capabilities']) {
        for (const cb of document.querySelectorAll('.capability-checkbox')) {
          cb.checked = data['capabilities'].includes(cb.value);
        }
      }
      // Mark auto-generated fields as user-edited if they have saved values
      if (data['app-id']) document.getElementById('app-id').dataset.userEdited = 'true';
      if (data['app-title']) document.getElementById('app-title').dataset.userEdited = 'true';
    } catch (e) { /* ignore parse errors */ }
  }

  // Save on every change (debounced via existing updatePreview)
  const origUpdatePreview = updatePreview;
  updatePreview = function () {
    origUpdatePreview();
    saveToLocalStorage();
  };

  // Load saved state
  loadFromLocalStorage();

  // Initial preview
  updatePreview();
});
