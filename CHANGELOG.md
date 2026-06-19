# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.1.1] - 2026-06-19

### Fixed

- Explicit `allowSelfSigned: true` in deploy config now always prints the TLS verification warning.

## [2.1.0] - 2026-06-19

### Added

- Deploy CLI configuration file support via `fastpack-deploy.json` or `FASTPACK_DEPLOY_CONFIG`.
- Reusable deploy settings for `cloudronUrl`, `username`, `password`, `token`, `subdomain`, and `allowSelfSigned`.
- Partial config support: the deploy wizard now prompts only for missing values.

### Fixed

- `CLOUDRON_TOKEN` remains the highest-priority token source for existing scripts.
- Explicit `allowSelfSigned: false` is now honored for localhost, private IP, and `*.nip.io` Cloudron URLs.

## [1.0.0] - 2026-03-14

### Added

- Single-page form to generate Cloudron custom app packages
- Smart defaults: auto-generated app ID, title, version from Docker image name
- Database support: PostgreSQL, MySQL, MongoDB, Redis
- SSO support: ProxyAuth, OIDC, LDAP, or no SSO (optionalSso)
- System addons: localstorage, sendmail, recvmail, scheduler, TLS, TURN, Docker
- TCP/UDP port configuration with dynamic form rows
- Automatic healthcheck server generation for non-web (TCP) services
- Web vs TCP mode toggle with appropriate start.sh generation
- Live preview of all generated files (manifest, Dockerfile, start.sh, .dockerignore, README)
- ZIP download with all files ready for `cloudron build && cloudron install`
- Progressive disclosure UI: simple 3-click path with collapsible advanced sections
- Form validation with field-level errors and warnings
- In-browser test suite (47 tests)
- Zero dependencies: vanilla HTML/CSS/JS, CDN-only for JSZip and FileSaver
