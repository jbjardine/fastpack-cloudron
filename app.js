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
  const packagerNameEl = document.getElementById('packager-name');
  const packagerUrlEl = document.getElementById('packager-url');
  const iconUrlEl = document.getElementById('icon-url');
  const mediaLinksEl = document.getElementById('media-links');
  const documentationUrlEl = document.getElementById('documentation-url');
  const forumUrlEl = document.getElementById('forum-url');

  // Advanced fields
  const minBoxVersionEl = document.getElementById('min-box-version');
  const multiDomainEl = document.getElementById('multi-domain');
  const runtimeDirsEl = document.getElementById('runtime-dirs');
  const persistentDirsEl = document.getElementById('persistent-dirs');
  const backupCommandEl = document.getElementById('backup-command');
  const restoreCommandEl = document.getElementById('restore-command');

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
    database,
    sso,
    addons,
    tcpPorts,
    udpPorts,
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
    packagerName: packagerNameEl.value.trim(),
    packagerUrl: packagerUrlEl.value.trim(),
    iconUrl: iconUrlEl.value.trim(),
    mediaLinks: mediaLinksEl.value.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0),
    documentationUrl: documentationUrlEl.value.trim(),
    forumUrl: forumUrlEl.value.trim(),
    // Advanced
    minBoxVersion: minBoxVersionEl.value.trim(),
    capabilities,
    multiDomain: multiDomainEl.checked,
    runtimeDirs: parseDirs(runtimeDirsEl),
    persistentDirs: parseDirs(persistentDirsEl),
    backupCommand: backupCommandEl.value.trim(),
    restoreCommand: restoreCommandEl.value.trim(),
    // ProxyAuth options
    proxyauthPath: proxyauthPathEl.value.trim(),
    proxyauthBasicAuth: proxyauthBasicAuthEl.checked,
    proxyauthBearerAuth: proxyauthBearerAuthEl.checked,
    // Database-specific options
    mysqlMultipleDbs: mysqlMultipleDbsEl.checked,
    mongodbOplog: mongodbOplogEl.checked,
    redisNoPassword: redisNoPasswordEl.checked,
    postgresqlLocale: postgresqlLocaleEl.value.trim(),
    // Sendmail options
    sendmailOptional: sendmailOptionalEl.checked,
    sendmailDisplayName: sendmailDisplayNameEl.checked,
    sendmailValidCert: sendmailValidCertEl.checked,
    // Scheduler tasks
    schedulerTasks,
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

  // Build ZIP using JSZip (loaded as global from CDN)
  const zip = new JSZip();
  zip.file('CloudronManifest.json', generateManifest(config));
  zip.file('Dockerfile', generateDockerfile(config));
  zip.file('start.sh', generateStartSh(config));
  zip.file('.dockerignore', generateDockerignore());
  zip.file('README.md', generateReadme(config));
  zip.file('CloudronVersions.json', generateCloudronVersions(config));

  // Add DESCRIPTION.md if description is provided
  const descContent = generateDescription(config);
  if (descContent) {
    zip.file('DESCRIPTION.md', descContent);
  }

  const blob = await zip.generateAsync({ type: 'blob' });
  const filename = `${sanitizeImageName(config.image) || 'cloudron-app'}-cloudron.zip`;
  saveAs(blob, filename);
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

  // Add scheduler task button
  document.getElementById('add-scheduler-task').addEventListener('click', function () {
    addSchedulerTaskRow();
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
  };

  // Preview tab switching
  const tabs = document.querySelectorAll('.preview-tab');
  const copyBtn = document.querySelector('.copy-btn');
  for (const tab of tabs) {
    tab.addEventListener('click', function () {
      // Remove active from all tabs
      for (const t of tabs) {
        t.classList.remove('active');
      }
      // Add active to clicked tab
      tab.classList.add('active');

      // Toggle preview content panels
      const target = tab.dataset.target;
      const panels = document.querySelectorAll('.preview-content');
      for (const panel of panels) {
        if (panel.dataset.panel === target) {
          panel.classList.add('active');
        } else {
          panel.classList.remove('active');
        }
      }

      // Update copy button target
      if (copyBtn && tabToCopyId[target]) {
        copyBtn.dataset.copy = tabToCopyId[target];
        copyBtn.textContent = 'Copy';
      }
    });
  }

  // Initial preview
  updatePreview();
});
