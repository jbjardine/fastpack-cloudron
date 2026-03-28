// app.js — Alpine.js UI logic for FastPackCloudron v2

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
 * Default config values — mirrors the shape expected by generators.js.
 */
function defaultConfig() {
  return {
    image: '',
    id: '',
    title: '',
    version: '1.0.0',
    httpPort: 8000,
    healthCheckPath: '/',
    hasWebUI: true,
    stack: '',
    database: '',
    sso: '',
    addons: ['localstorage'],

    // Metadata
    author: '',
    tagline: '',
    description: '',
    website: '',
    contactEmail: '',
    tags: [],
    configurePath: '',
    upstreamVersion: '',
    postInstallMessage: '',
    changelog: '',
    icon: '',
    memoryLimit: 0,

    // OIDC
    oidcRedirectUri: '/auth/openid/callback',
    oidcLogoutUri: '/',
    oidcTokenAlgo: '',

    // ProxyAuth
    proxyauthPath: '',
    proxyauthBasicAuth: false,
    proxyauthBearerAuth: false,

    // Database-specific
    mysqlMultipleDbs: false,
    mongodbOplog: false,
    redisNoPassword: false,
    postgresqlLocale: '',

    // Localstorage
    localstorageFtp: false,
    localstorageSqlite: false,
    localstorageSqlitePaths: '',

    // Sendmail
    sendmailOptional: false,
    sendmailDisplayName: false,
    sendmailValidCert: false,

    // Ports (dynamic arrays)
    tcpPorts: [],
    udpPorts: [],
    httpPorts: [],

    // Scheduler
    schedulerTasks: [],

    // Publishing
    dockerHubUsername: '',
    packagerName: '',
    packagerUrl: '',
    iconUrl: '',
    mediaLinks: '',
    documentationUrl: '',
    forumUrl: '',

    // Advanced
    minBoxVersion: '',
    maxBoxVersion: '',
    targetBoxVersion: '',
    capabilities: [],
    multiDomain: false,
    fullDomain: false,
    singleUser: false,
    dockerfileCloudron: false,
    secondarySubdomains: '',
    runtimeDirs: '',
    persistentDirs: '',
    backupCommand: '',
    restoreCommand: '',
    logPaths: '',
    checklist: '',

    // Multi-stage
    copyFrom: [],

    // Multi-service
    services: [],

    // DooD sub-containers
    subcontainers: [],
  };
}

/**
 * Available tags for checkbox grid.
 */
const AVAILABLE_TAGS = ['blog', 'chat', 'git', 'email', 'sync', 'gallery', 'notes', 'project', 'hosting', 'wiki'];

/**
 * Available capabilities for checkbox grid.
 */
const AVAILABLE_CAPABILITIES = ['net_admin', 'mlock', 'ping', 'vaapi'];

/**
 * Available addons for checkbox grid.
 */
const AVAILABLE_ADDONS = [
  { value: 'localstorage', label: 'localstorage' },
  { value: 'sendmail', label: 'sendmail' },
  { value: 'recvmail', label: 'recvmail' },
  { value: 'email', label: 'email (full)' },
  { value: 'scheduler', label: 'scheduler' },
  { value: 'tls', label: 'TLS' },
  { value: 'turn', label: 'TURN' },
  { value: 'docker', label: 'Docker' },
];

/**
 * Sections for advanced mode sidebar navigation.
 */
const SECTIONS = [
  { id: 'section-general', label: 'General' },
  { id: 'section-metadata', label: 'Metadata' },
  { id: 'section-network', label: 'Network' },
  { id: 'section-database', label: 'Database' },
  { id: 'section-auth', label: 'Auth / SSO' },
  { id: 'section-addons', label: 'Addons' },
  { id: 'section-services', label: 'Services' },
  { id: 'section-subcontainers', label: 'Sub-containers' },
  { id: 'section-ports', label: 'Ports' },
  { id: 'section-capabilities', label: 'Capabilities' },
  { id: 'section-build', label: 'Build' },
  { id: 'section-publishing', label: 'Publishing' },
];

/**
 * Preview tabs with human-readable labels.
 */
const PREVIEW_TABS = [
  { key: 'manifest', label: 'manifest.json' },
  { key: 'dockerfile', label: 'Dockerfile' },
  { key: 'startsh', label: 'start.sh' },
  { key: 'dockerignore', label: '.dockerignore' },
  { key: 'readme', label: 'README.md' },
  { key: 'versions', label: 'Versions.json' },
];

/**
 * Parses a comma-separated string to array, filtering empty entries.
 */
function parseCsv(str) {
  if (!str) return [];
  return str.split(',').map(s => s.trim()).filter(s => s.length > 0);
}

/**
 * Builds the config object for generators.js from the Alpine store state.
 * This replaces the old buildConfig() that read DOM elements manually.
 */
function buildConfigFromStore(c) {
  // Auto-generate id if not user-edited
  const id = c._idUserEdited && c.id
    ? c.id
    : (c.image ? `io.fastpack.${sanitizeImageName(c.image)}` : '');

  // Auto-generate title if not user-edited
  const title = c._titleUserEdited && c.title
    ? c.title
    : (c.image ? humanizeImageName(c.image) : '');

  const database = c.database || null;
  const sso = c.sso || null;
  const memoryLimitMB = parseInt(c.memoryLimit, 10) || 0;

  return {
    image: c.image.trim(),
    id,
    title,
    version: c.version.trim() || '1.0.0',
    httpPort: c.hasWebUI ? (parseInt(c.httpPort, 10) || 8000) : 8000,
    healthCheckPath: c.healthCheckPath.trim() || '/',
    hasWebUI: c.hasWebUI,
    stack: c.stack,
    database,
    sso,
    addons: [...c.addons],
    tcpPorts: c.tcpPorts.map(p => ({
      name: (p.name || '').trim(),
      title: (p.title || '').trim(),
      containerPort: parseInt(p.containerPort, 10) || 0,
      defaultValue: parseInt(p.defaultValue, 10) || 0,
    })),
    udpPorts: c.udpPorts.map(p => ({
      name: (p.name || '').trim(),
      title: (p.title || '').trim(),
      containerPort: parseInt(p.containerPort, 10) || 0,
      defaultValue: parseInt(p.defaultValue, 10) || 0,
    })),
    httpPorts: c.httpPorts.map(p => ({
      name: (p.name || '').trim(),
      title: (p.title || '').trim(),
      containerPort: parseInt(p.containerPort, 10) || 0,
      defaultValue: parseInt(p.defaultValue, 10) || 0,
    })),
    author: c.author.trim(),
    tagline: c.tagline.trim(),
    description: c.description.trim(),
    oidcRedirectUri: c.oidcRedirectUri.trim() || '/auth/openid/callback',
    oidcLogoutUri: c.oidcLogoutUri.trim() || '/',
    oidcTokenAlgo: c.oidcTokenAlgo || '',
    website: c.website.trim(),
    contactEmail: c.contactEmail.trim(),
    tags: [...c.tags],
    configurePath: c.configurePath.trim(),
    upstreamVersion: c.upstreamVersion.trim(),
    postInstallMessage: c.postInstallMessage.trim(),
    changelog: c.changelog.trim(),
    icon: c.icon.trim(),
    memoryLimit: memoryLimitMB > 0 ? memoryLimitMB * 1024 * 1024 : 0,
    // Publishing
    dockerHubUsername: c.dockerHubUsername.trim(),
    packagerName: c.packagerName.trim(),
    packagerUrl: c.packagerUrl.trim(),
    iconUrl: c.iconUrl.trim(),
    mediaLinks: c.mediaLinks.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0),
    documentationUrl: c.documentationUrl.trim(),
    forumUrl: c.forumUrl.trim(),
    // Advanced
    minBoxVersion: c.minBoxVersion.trim(),
    maxBoxVersion: c.maxBoxVersion.trim(),
    targetBoxVersion: c.targetBoxVersion.trim(),
    capabilities: [...c.capabilities],
    multiDomain: c.multiDomain,
    fullDomain: c.fullDomain,
    singleUser: c.singleUser,
    dockerfileCloudron: c.dockerfileCloudron,
    secondarySubdomains: parseCsv(c.secondarySubdomains),
    runtimeDirs: parseCsv(c.runtimeDirs),
    persistentDirs: parseCsv(c.persistentDirs),
    backupCommand: c.backupCommand.trim(),
    restoreCommand: c.restoreCommand.trim(),
    logPaths: parseCsv(c.logPaths),
    checklist: c.checklist.trim().split('\n').map(s => s.trim()).filter(s => s.length > 0),
    // ProxyAuth
    proxyauthPath: c.proxyauthPath.trim(),
    proxyauthBasicAuth: c.proxyauthBasicAuth,
    proxyauthBearerAuth: c.proxyauthBearerAuth,
    // Database-specific
    mysqlMultipleDbs: c.mysqlMultipleDbs,
    mongodbOplog: c.mongodbOplog,
    redisNoPassword: c.redisNoPassword,
    postgresqlLocale: c.postgresqlLocale.trim(),
    // Localstorage
    localstorageFtp: c.localstorageFtp,
    localstorageSqlite: c.localstorageSqlite,
    localstorageSqlitePaths: parseCsv(c.localstorageSqlitePaths),
    // Sendmail
    sendmailOptional: c.sendmailOptional,
    sendmailDisplayName: c.sendmailDisplayName,
    sendmailValidCert: c.sendmailValidCert,
    // Scheduler
    schedulerTasks: c.schedulerTasks.map(t => ({
      name: (t.name || '').trim(),
      schedule: (t.schedule || '').trim(),
      command: (t.command || '').trim(),
    })),
    // Multi-stage
    copyFrom: c.copyFrom
      .filter(cf => cf.image && cf.src && cf.dest)
      .map(cf => ({
        image: cf.image.trim(),
        src: cf.src.trim(),
        dest: cf.dest.trim(),
      })),
    // Multi-service
    services: c.services.map(s => ({
      name: (s.name || '').trim(),
      command: (s.command || '').trim(),
      internalPort: parseInt(s.internalPort, 10) || 0,
      routePath: (s.routePath || '').trim() || null,
      sso: s.sso || 'none',
    })),
    // DooD sub-containers
    subcontainers: c.subcontainers
      .filter(s => s.image)
      .map(s => ({
        image: s.image.trim(),
        port: parseInt(s.port, 10) || 80,
        route: (s.route || '').trim() || '/',
        memory: parseInt(s.memory, 10) || 256,
        volume: (s.volume || '').trim() || '/data',
      })),
  };
}

/**
 * Validates a config object. Returns { errors: [], warnings: [] }.
 * Ported directly from the old app.js — operates on plain objects.
 */
function validate(config) {
  const errors = [];
  const warnings = [];

  if (!config.image) {
    errors.push({ field: 'image', message: 'Enter a Docker image (e.g., nginx:latest)' });
  } else if (!SAFE_DOCKER_REF.test(config.image)) {
    errors.push({ field: 'image', message: 'Invalid image name. Use only letters, digits, dots, hyphens, slashes, colons, and @sha256 digests.' });
  }

  if (config.version && !SAFE_VERSION.test(config.version)) {
    errors.push({ field: 'version', message: 'Invalid version. Use only letters, digits, dots, and hyphens.' });
  }

  if (config.copyFrom && config.copyFrom.length > 0) {
    for (const cf of config.copyFrom) {
      if (cf.image && !SAFE_DOCKER_REF.test(cf.image)) {
        errors.push({ field: 'copyFrom', message: `Invalid COPY --from image "${cf.image}".` });
        break;
      }
      if (cf.src && !SAFE_PATH.test(cf.src)) {
        errors.push({ field: 'copyFrom', message: `Invalid source path "${cf.src}".` });
        break;
      }
      if (cf.dest && !SAFE_PATH.test(cf.dest)) {
        errors.push({ field: 'copyFrom', message: `Invalid destination path "${cf.dest}".` });
        break;
      }
    }
  }

  if (config.services && config.services.length > 0) {
    for (const svc of config.services) {
      if (svc.name && !SAFE_IDENTIFIER.test(svc.name)) {
        errors.push({ field: 'services', message: `Invalid service name "${svc.name}". Start with a letter, then letters, digits, hyphens, underscores.` });
        break;
      }
      if (svc.routePath && !SAFE_ROUTE_PATH.test(svc.routePath)) {
        errors.push({ field: 'services', message: `Invalid route path "${svc.routePath}". Must start with /.` });
        break;
      }
    }
  }

  if (config.id && !/^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$/.test(config.id)) {
    errors.push({ field: 'id', message: 'Expected format: com.example.myapp' });
  }

  if (config.httpPort < 1 || config.httpPort > 65535 || isNaN(config.httpPort)) {
    errors.push({ field: 'httpPort', message: 'Invalid port' });
  }

  if (config.healthCheckPath && !config.healthCheckPath.startsWith('/')) {
    errors.push({ field: 'healthCheckPath', message: 'Must start with /' });
  }

  const allPorts = [...config.tcpPorts, ...config.udpPorts];
  for (const port of allPorts) {
    if (port.name && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(port.name)) {
      errors.push({ field: 'ports', message: `Invalid port name "${port.name}"` });
      break;
    }
  }

  const usedTcpPorts = new Set([config.httpPort]);
  let hasPortConflict = false;
  for (const port of config.tcpPorts) {
    if (port.containerPort && usedTcpPorts.has(port.containerPort)) hasPortConflict = true;
    if (port.containerPort) usedTcpPorts.add(port.containerPort);
  }
  const usedUdpPorts = new Set();
  for (const port of config.udpPorts) {
    if (port.containerPort && usedUdpPorts.has(port.containerPort)) hasPortConflict = true;
    if (port.containerPort) usedUdpPorts.add(port.containerPort);
  }
  if (hasPortConflict) {
    errors.push({ field: 'ports', message: 'Port conflict detected' });
  }

  for (const task of config.schedulerTasks) {
    if (task.name && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(task.name)) {
      errors.push({ field: 'scheduler', message: `Invalid task name "${task.name}"` });
      break;
    }
  }

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

  if (config.minBoxVersion && !/^\d+\.\d+\.\d+$/.test(config.minBoxVersion)) {
    warnings.push({ message: 'Minimum Cloudron version should be semver (e.g., 7.0.0)' });
  }

  if (!config.hasWebUI && config.tcpPorts.length === 0 && config.udpPorts.length === 0) {
    warnings.push({ message: 'Your app exposes no service. Add at least one TCP port.' });
  }

  if (!config.addons.includes('localstorage')) {
    warnings.push({ message: 'Without localstorage, your app cannot persist data. Are you sure?' });
  }

  if (config.capabilities.length > 0) {
    warnings.push({ message: 'Capabilities grant elevated privileges. Only use if your app requires them.' });
  }

  const imageLower = (config.image || '').toLowerCase();
  if (imageLower.includes('distroless') || imageLower.includes('scratch')) {
    warnings.push({ message: 'This image has no shell. start.sh requires /bin/sh.' });
  }
  if (imageLower.includes('busybox') && !config.hasWebUI) {
    warnings.push({ message: 'BusyBox has no python3. TCP healthcheck will fail.' });
  }
  if (config.httpPort === 80 && imageLower.includes('nginx')) {
    warnings.push({ message: 'nginx needs writable /var/cache/nginx. Add runtime dirs or use a different port.' });
  }
  if (config.services && config.services.length >= 2) {
    warnings.push({ message: 'Multi-service apps run all processes in one container. Consider separate Cloudron apps.' });
  }
  if (config.copyFrom && config.copyFrom.length > 0) {
    const baseIsAlpine = imageLower.includes('alpine');
    for (const cf of config.copyFrom) {
      const cfLower = (cf.image || '').toLowerCase();
      if (baseIsAlpine !== cfLower.includes('alpine') && (baseIsAlpine || cfLower.includes('alpine'))) {
        warnings.push({ message: `COPY --from=${cf.image}: mixing Alpine (musl) and Debian (glibc). Binaries may not run.` });
        break;
      }
    }
  }
  if (/fedora|centos|rocky|amazon/.test(imageLower)) {
    warnings.push({ message: 'This image uses dnf. gosu will be installed via util-linux/setpriv.' });
  }

  return { errors, warnings };
}

// --- localStorage persistence ---
const STORAGE_KEY = 'fastpack-cloudron-config';
const STORAGE_VERSION = 2;

/**
 * Saves the Alpine store to localStorage.
 */
function saveStore(store) {
  try {
    const data = { _v: STORAGE_VERSION };
    const skip = new Set(['_mode', '_activeTab', '_activeSection', '_searchQuery', '_idUserEdited', '_titleUserEdited', '_iconFile', 'AVAILABLE_TAGS', 'AVAILABLE_CAPABILITIES', 'AVAILABLE_ADDONS', 'SECTIONS', 'PREVIEW_TABS']);
    for (const [k, v] of Object.entries(store)) {
      if (!k.startsWith('_') || k === '_idUserEdited' || k === '_titleUserEdited') {
        if (!skip.has(k) || k === '_idUserEdited' || k === '_titleUserEdited') {
          data[k] = v;
        }
      }
    }
    // Save UI-only flags that matter for persistence
    data._idUserEdited = store._idUserEdited;
    data._titleUserEdited = store._titleUserEdited;
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch (_) { /* ignore storage errors */ }
}

/**
 * Loads saved state from localStorage, handling v1 migration.
 */
function loadStore() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const data = JSON.parse(raw);

    // v2 format: direct store shape
    if (data._v === STORAGE_VERSION) {
      delete data._v;
      return data;
    }

    // v1 migration: old format used DOM element IDs as keys
    return migrateV1(data);
  } catch (_) {
    return null;
  }
}

/**
 * Migrates v1 localStorage format (keyed by DOM element IDs) to v2 store shape.
 */
function migrateV1(data) {
  const store = {};
  const fieldMap = {
    'docker-image': 'image',
    'app-id': 'id',
    'app-title': 'title',
    'app-version': 'version',
    'http-port': 'httpPort',
    'health-check-path': 'healthCheckPath',
    'app-stack': 'stack',
    'database': 'database',
    'sso': 'sso',
    'app-author': 'author',
    'app-tagline': 'tagline',
    'app-description': 'description',
    'app-website': 'website',
    'app-contact-email': 'contactEmail',
    'app-configure-path': 'configurePath',
    'app-upstream-version': 'upstreamVersion',
    'app-post-install-message': 'postInstallMessage',
    'app-changelog': 'changelog',
    'app-icon': 'icon',
    'app-memory-limit': 'memoryLimit',
    'oidc-redirect-uri': 'oidcRedirectUri',
    'oidc-logout-uri': 'oidcLogoutUri',
    'oidc-token-algo': 'oidcTokenAlgo',
    'proxyauth-path': 'proxyauthPath',
    'docker-hub-username': 'dockerHubUsername',
    'packager-name': 'packagerName',
    'packager-url': 'packagerUrl',
    'icon-url': 'iconUrl',
    'media-links': 'mediaLinks',
    'documentation-url': 'documentationUrl',
    'forum-url': 'forumUrl',
    'min-box-version': 'minBoxVersion',
    'max-box-version': 'maxBoxVersion',
    'target-box-version': 'targetBoxVersion',
    'runtime-dirs': 'runtimeDirs',
    'persistent-dirs': 'persistentDirs',
    'backup-command': 'backupCommand',
    'restore-command': 'restoreCommand',
    'secondary-subdomains': 'secondarySubdomains',
    'log-paths': 'logPaths',
    'checklist': 'checklist',
    'postgresql-locale': 'postgresqlLocale',
    'localstorage-sqlite-paths': 'localstorageSqlitePaths',
  };

  for (const [oldKey, newKey] of Object.entries(fieldMap)) {
    if (data[oldKey] !== undefined) store[newKey] = data[oldKey];
  }

  // Migrate checkboxes
  const cbMap = {
    'cb:multi-domain': 'multiDomain',
    'cb:full-domain': 'fullDomain',
    'cb:single-user': 'singleUser',
    'cb:proxyauth-basic-auth': 'proxyauthBasicAuth',
    'cb:proxyauth-bearer-auth': 'proxyauthBearerAuth',
    'cb:mysql-multiple-dbs': 'mysqlMultipleDbs',
    'cb:mongodb-oplog': 'mongodbOplog',
    'cb:redis-no-password': 'redisNoPassword',
    'cb:sendmail-optional': 'sendmailOptional',
    'cb:sendmail-display-name': 'sendmailDisplayName',
    'cb:sendmail-valid-cert': 'sendmailValidCert',
    'cb:localstorage-ftp': 'localstorageFtp',
    'cb:localstorage-sqlite': 'localstorageSqlite',
    'cb:dockerfile-cloudron': 'dockerfileCloudron',
  };

  for (const [oldKey, newKey] of Object.entries(cbMap)) {
    if (data[oldKey] !== undefined) store[newKey] = data[oldKey];
  }

  // Migrate hasWebUI from radio buttons
  if (data['cb:web-ui-yes'] !== undefined) store.hasWebUI = data['cb:web-ui-yes'];

  // Migrate arrays
  if (data.addons) store.addons = data.addons;
  if (data.tags) store.tags = data.tags;
  if (data.capabilities) store.capabilities = data.capabilities;

  // Mark id/title as user-edited if they had values
  if (store.id) store._idUserEdited = true;
  if (store.title) store._titleUserEdited = true;

  return store;
}

/**
 * Main Alpine.js data function for the app.
 */
export function fpApp() {
  const defaults = defaultConfig();
  const saved = loadStore();

  // Merge saved state over defaults
  const initial = saved ? { ...defaults, ...saved } : defaults;

  return {
    // Config fields (mirroring generators.js config shape)
    ...initial,

    // UI-only state
    _mode: 'guided',
    _activeTab: 'manifest',
    _activeSection: 'section-general',
    _searchQuery: '',
    _idUserEdited: initial._idUserEdited || false,
    _titleUserEdited: initial._titleUserEdited || false,
    _iconFile: null,
    _errors: [],
    _warnings: [],
    _previews: {},
    _downloading: false,
    _copyFeedback: '',
    _showDeployWizard: false,
    _zipFilename: '',

    // Guided mode checkbox states (controls progressive disclosure)
    _showDatabase: false,
    _showAuth: false,
    _showPort: false,
    _showStack: false,
    _showSubcontainer: false,

    // Constants exposed to template
    AVAILABLE_TAGS,
    AVAILABLE_CAPABILITIES,
    AVAILABLE_ADDONS,
    SECTIONS,
    PREVIEW_TABS,

    /**
     * Initialize: set up watchers, sync guided mode checkboxes with existing data.
     */
    init() {
      // Sync guided checkboxes with any restored data
      if (this.database) this._showDatabase = true;
      if (this.sso) this._showAuth = true;
      if (this.httpPort !== 8000) this._showPort = true;
      if (this.stack) this._showStack = true;
      if (this.subcontainers.length > 0) this._showSubcontainer = true;

      // Initial preview generation
      this.updatePreviews();

      // Watch all config changes for live preview + persistence
      this.$watch('image', () => { this.onImageChange(); this.debouncedUpdate(); });
      // Watch key fields that affect preview
      const watchFields = [
        'version', 'httpPort', 'healthCheckPath', 'hasWebUI', 'stack',
        'database', 'sso', 'author', 'tagline', 'description',
        'oidcRedirectUri', 'oidcLogoutUri', 'oidcTokenAlgo',
        'proxyauthPath', 'proxyauthBasicAuth', 'proxyauthBearerAuth',
        'mysqlMultipleDbs', 'mongodbOplog', 'redisNoPassword', 'postgresqlLocale',
        'localstorageFtp', 'localstorageSqlite', 'localstorageSqlitePaths',
        'sendmailOptional', 'sendmailDisplayName', 'sendmailValidCert',
        'dockerHubUsername', 'packagerName', 'packagerUrl', 'iconUrl',
        'mediaLinks', 'documentationUrl', 'forumUrl',
        'minBoxVersion', 'maxBoxVersion', 'targetBoxVersion',
        'multiDomain', 'fullDomain', 'singleUser', 'dockerfileCloudron',
        'secondarySubdomains', 'runtimeDirs', 'persistentDirs',
        'backupCommand', 'restoreCommand', 'logPaths', 'checklist',
        'website', 'contactEmail', 'configurePath', 'upstreamVersion',
        'postInstallMessage', 'changelog', 'icon', 'memoryLimit',
        'id', 'title',
      ];
      for (const field of watchFields) {
        this.$watch(field, () => this.debouncedUpdate());
      }

      // Deep watch arrays
      this.$watch('addons', () => this.debouncedUpdate());
      this.$watch('tags', () => this.debouncedUpdate());
      this.$watch('capabilities', () => this.debouncedUpdate());
      this.$watch('tcpPorts', () => this.debouncedUpdate());
      this.$watch('udpPorts', () => this.debouncedUpdate());
      this.$watch('httpPorts', () => this.debouncedUpdate());
      this.$watch('schedulerTasks', () => this.debouncedUpdate());
      this.$watch('copyFrom', () => this.debouncedUpdate());
      this.$watch('services', () => this.debouncedUpdate());
      this.$watch('subcontainers', () => this.debouncedUpdate());

      // Guided mode checkbox sync
      this.$watch('_showDatabase', (v) => { if (!v) this.database = ''; });
      this.$watch('_showAuth', (v) => { if (!v) this.sso = ''; });
      this.$watch('_showPort', (v) => { if (!v) this.httpPort = 8000; });
      this.$watch('_showStack', (v) => { if (!v) this.stack = ''; });
      this.$watch('_showSubcontainer', (v) => { if (!v) this.subcontainers = []; });

      // Scrollspy for advanced mode
      this.initScrollspy();
    },

    _debounceTimer: null,

    debouncedUpdate() {
      clearTimeout(this._debounceTimer);
      this._debounceTimer = setTimeout(() => {
        this.updatePreviews();
        saveStore(this.$data);
      }, 150);
    },

    /**
     * When image changes, update auto-generated id/title.
     */
    onImageChange() {
      if (!this._idUserEdited) {
        this.id = this.image ? `io.fastpack.${sanitizeImageName(this.image)}` : '';
      }
      if (!this._titleUserEdited) {
        this.title = this.image ? humanizeImageName(this.image) : '';
      }
    },

    markIdEdited() { this._idUserEdited = true; },
    markTitleEdited() { this._titleUserEdited = true; },

    /**
     * Build config, validate, and update all preview panes.
     */
    updatePreviews() {
      const config = buildConfigFromStore(this);
      const result = validate(config);
      this._errors = result.errors;
      this._warnings = result.warnings;

      try {
        this._previews = {
          manifest: generateManifest(config),
          dockerfile: generateDockerfile(config),
          startsh: generateStartSh(config),
          dockerignore: generateDockerignore(),
          readme: generateReadme(config),
          versions: generateCloudronVersions(config),
          nginx: (config.services.length > 0 || config.subcontainers.length > 0) ? generateNginxConf(config) : '',
        };
      } catch (_) {
        // generators may throw on invalid input — keep last valid previews
      }
    },

    /**
     * Returns the currently visible preview content.
     */
    get activePreview() {
      return this._previews[this._activeTab] || '';
    },

    /**
     * Whether the nginx tab should be visible.
     */
    get showNginxTab() {
      return this.services.length > 0 || this.subcontainers.length > 0;
    },

    /**
     * Computed live summary for guided mode.
     */
    get summary() {
      const parts = [];
      if (this.image) parts.push(`Image: ${this.image}`);
      parts.push(`DB: ${this.database || 'none'}`);
      parts.push(`Auth: ${this.sso || 'none'}`);
      parts.push(`Port: ${this.httpPort}`);
      if (this.stack) parts.push(`Stack: ${this.stack}`);
      if (this.subcontainers.length > 0) parts.push(`Sub-containers: ${this.subcontainers.length}`);
      return parts;
    },

    /**
     * Switch preview tab.
     */
    setTab(tab) {
      this._activeTab = tab;
      this.$nextTick(() => {
        const el = document.querySelector('[role="tab"][aria-selected="true"]');
        if (el) el.focus();
      });
    },

    /**
     * Navigate to the next/previous preview tab (arrow key support for WCAG).
     */
    navigateTab(currentKey, direction) {
      const tabs = [...PREVIEW_TABS.map(t => t.key)];
      if (this.showNginxTab) tabs.push('nginx');
      const idx = tabs.indexOf(currentKey);
      if (idx < 0) return;
      const next = direction === 'next'
        ? tabs[(idx + 1) % tabs.length]
        : tabs[(idx - 1 + tabs.length) % tabs.length];
      this.setTab(next);
    },

    /**
     * Copy current preview to clipboard.
     */
    async copyPreview() {
      try {
        await navigator.clipboard.writeText(this.activePreview);
        this._copyFeedback = 'Copied!';
        setTimeout(() => { this._copyFeedback = ''; }, 1200);
      } catch (_) { /* clipboard API may fail */ }
    },

    /**
     * Handle icon file upload validation.
     */
    handleIconFile(event) {
      const file = event.target.files[0];
      this._iconFile = file || null;
      // Validation is done at download time
    },

    /**
     * Download ZIP package.
     */
    async downloadZip() {
      const config = buildConfigFromStore(this);
      const result = validate(config);

      if (result.errors.length > 0) {
        this._errors = result.errors;
        return;
      }

      this._downloading = true;
      try {
        const zip = new JSZip();
        zip.file('CloudronManifest.json', generateManifest(config));
        const dockerfileName = config.dockerfileCloudron ? 'Dockerfile.cloudron' : 'Dockerfile';
        zip.file(dockerfileName, generateDockerfile(config));
        zip.file('start.sh', generateStartSh(config));
        zip.file('.dockerignore', generateDockerignore());
        zip.file('README.md', generateReadme(config));
        zip.file('CloudronVersions.json', generateCloudronVersions(config));

        const descContent = generateDescription(config);
        if (descContent) zip.file('DESCRIPTION.md', descContent);

        if (config.services.length > 0 || config.subcontainers.length > 0) {
          zip.file('nginx.conf', generateNginxConf(config));
        }

        // Icon file
        if (this._iconFile) {
          const iconData = await this._iconFile.arrayBuffer();
          zip.file('icon.png', iconData);
        }

        zip.file('POSTINSTALL.md', generatePostInstall(config));
        zip.file('CHANGELOG.md', generateChangelog(config));
        zip.file('deploy.js', generateDeploySh());
        zip.file('deploy.cmd', generateDeployCmd());

        const blob = await zip.generateAsync({ type: 'blob' });
        const filename = `${sanitizeImageName(config.image) || 'cloudron-app'}-cloudron.zip`;
        saveAs(blob, filename);
        this._zipFilename = filename;
        this._showDeployWizard = true;
      } finally {
        this._downloading = false;
      }
    },

    // --- Dynamic array management ---

    addTcpPort() { this.tcpPorts.push({ name: '', title: '', containerPort: '', defaultValue: '' }); },
    removeTcpPort(i) { this.tcpPorts.splice(i, 1); },

    addUdpPort() { this.udpPorts.push({ name: '', title: '', containerPort: '', defaultValue: '' }); },
    removeUdpPort(i) { this.udpPorts.splice(i, 1); },

    addHttpPort() { this.httpPorts.push({ name: '', title: '', containerPort: '', defaultValue: '' }); },
    removeHttpPort(i) { this.httpPorts.splice(i, 1); },

    addSchedulerTask() { this.schedulerTasks.push({ name: '', schedule: '', command: '' }); },
    removeSchedulerTask(i) { this.schedulerTasks.splice(i, 1); },

    addCopyFrom() { this.copyFrom.push({ image: '', src: '', dest: '' }); },
    removeCopyFrom(i) { this.copyFrom.splice(i, 1); },

    addService() { this.services.push({ name: '', command: '', internalPort: '', routePath: '', sso: 'none' }); },
    removeService(i) { this.services.splice(i, 1); },

    addSubcontainer() { this.subcontainers.push({ image: '', port: 80, route: '/', memory: 256, volume: '/data' }); },
    removeSubcontainer(i) { this.subcontainers.splice(i, 1); },

    // --- Addon toggle helpers ---

    hasAddon(name) { return this.addons.includes(name); },
    toggleAddon(name) {
      const i = this.addons.indexOf(name);
      if (i >= 0) this.addons.splice(i, 1);
      else this.addons.push(name);
    },

    hasTag(name) { return this.tags.includes(name); },
    toggleTag(name) {
      const i = this.tags.indexOf(name);
      if (i >= 0) this.tags.splice(i, 1);
      else this.tags.push(name);
    },

    hasCapability(name) { return this.capabilities.includes(name); },
    toggleCapability(name) {
      const i = this.capabilities.indexOf(name);
      if (i >= 0) this.capabilities.splice(i, 1);
      else this.capabilities.push(name);
    },

    // --- Mode switching ---

    switchToAdvanced() { this._mode = 'advanced'; },
    switchToGuided() { this._mode = 'guided'; },

    get isGuided() { return this._mode === 'guided'; },
    get isAdvanced() { return this._mode === 'advanced'; },

    // --- Scrollspy for advanced mode sidebar ---

    initScrollspy() {
      if (typeof IntersectionObserver === 'undefined') return;
      const observer = new IntersectionObserver((entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            this._activeSection = entry.target.id;
          }
        }
      }, { rootMargin: '-20% 0px -70% 0px' });

      // Defer to next tick to ensure elements exist
      this.$nextTick(() => {
        for (const s of SECTIONS) {
          const el = document.getElementById(s.id);
          if (el) observer.observe(el);
        }
      });
    },

    scrollToSection(sectionId) {
      const el = document.getElementById(sectionId);
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
      this._activeSection = sectionId;
    },

    // --- Search filter for advanced mode ---

    matchesSearch(label) {
      if (!this._searchQuery) return true;
      return label.toLowerCase().includes(this._searchQuery.toLowerCase());
    },

    // --- Presets ---

    applyPreset(name) {
      const presets = {
        simple: { image: 'nginx:latest', database: '', sso: '', stack: '' },
        custom: { image: '', database: '', sso: '', stack: '' },
        multi: { image: '', database: 'postgresql', sso: 'oidc', stack: '' },
      };
      const preset = presets[name];
      if (!preset) return;

      this.image = preset.image;
      this.database = preset.database;
      this.sso = preset.sso;
      this.stack = preset.stack;

      if (name === 'multi') {
        if (!this.hasAddon('docker')) this.addons.push('docker');
        this._showDatabase = true;
        this._showAuth = true;
      }

      // Reset auto-generation
      this._idUserEdited = false;
      this._titleUserEdited = false;
      this.onImageChange();
    },

    /**
     * Error message for a specific field.
     */
    errorFor(field) {
      const err = this._errors.find(e => e.field === field);
      return err ? err.message : '';
    },
  };
}
