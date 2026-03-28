# FastPackCloudron UI v2 — Spec complète

**Date:** 2026-03-28
**Status:** Validé post-débat multi-AI (Claude Opus + Gemini + Codex gpt-5.4)
**Intent Contract:** .claude/session-intent.md
**Plan d'exécution:** .claude/session-plan.md

---

## 1. Objectif

Un novice réussit le cas standard en 2 minutes, sans documentation.
Charlie (devops) trouve toutes les options avancées.
Déployé sur GitHub Pages (client-side only, single-page).

## 2. Architecture

### Pattern retenu : "Guided + Advanced"

Deux modes, une seule page, un seul modèle de données :

- **Mode Guidé** (défaut) : 1 input Docker + checkboxes qui révèlent les options à la demande. Live summary. CTA "Générer". C'est ce que le novice voit.
- **Mode Avancé** : formulaire complet avec navigation latérale par sections. Tous les 50+ champs visibles. C'est ce que le devops voit.

Le mode Guidé ne fait que **cacher** des champs via CSS (`display: none`). Il ne supprime ni n'écrase jamais de valeurs. Passer de Guidé à Avancé révèle tous les champs avec les valeurs déjà saisies intactes.

### Stack technique

| Choix | Justification |
|-------|--------------|
| **Alpine.js** (15kb) | Réactivité `x-model` pour 50 champs sans spaghetti DOM. Vendoré dans le repo, PAS de CDN. |
| **system-ui** | Zéro fonts externes, zéro latence |
| **generators.js** inchangé | Contrat stable, 313 tests CI de régression |
| **Single HTML file** | GitHub Pages compatible, pas de build step |

### Flow de données

```
Alpine.js x-data (store central = objet config)
    ↓ x-model binding bidirectionnel
Mode Guidé / Mode Avancé (vues CSS du même état)
    ↓ watch/effect
generators.js (14 fonctions, inchangé)
    ↓
Preview live + Download ZIP
```

## 3. Contrat generators.js (gelé)

L'API publique de generators.js est le contrat stable. La nouvelle UI doit construire le même objet `config` que `buildConfig()` dans l'app.js actuel.

### Fonctions exportées (14)

| Fonction | Signature | Retour |
|----------|-----------|--------|
| `sanitizeImageName(image)` | string → string | ID-safe name |
| `humanizeImageName(image)` | string → string | Human title |
| `generateManifest(config)` | config → string | JSON |
| `generateDockerfile(config)` | config → string | Dockerfile |
| `generateStartSh(config)` | config → string | Shell script |
| `generateDescription(config)` | config → string | Markdown |
| `generateDockerignore()` | void → string | Text |
| `generateReadme(config)` | config → string | Markdown |
| `generateDeploySh()` | void → string | Node.js script |
| `generateDeployCmd()` | void → string | CMD script |
| `generateNginxConf(config)` | config → string | nginx.conf |
| `generateCloudronVersions(config)` | config → string | JSON |
| `generatePostInstall(config)` | config → string | Markdown |
| `generateChangelog(config)` | config → string | Markdown |

### Regex exportées (5)

`SAFE_DOCKER_REF`, `SAFE_PATH`, `SAFE_IDENTIFIER`, `SAFE_VERSION`, `SAFE_ROUTE_PATH`

### Objet config (champs principaux)

```js
{
  // Requis
  image: string,           // seul champ obligatoire

  // Auto-générés si non édités
  id: string,              // io.fastpack.{sanitized}
  title: string,           // humanized image name

  // Core options
  version: string,         // default "1.0.0"
  httpPort: number,        // default 8000
  healthCheckPath: string, // default "/"
  hasWebUI: boolean,       // default true
  stack: string,           // "", "nodejs", "php", "python", "java", "go"
  database: string|null,   // null, "postgresql", "mysql", "mongodb", "redis"
  sso: string|null,        // null, "proxyAuth", "oidc", "ldap", "oauth", "simpleauth"
  addons: string[],        // ["localstorage", "sendmail", ...]

  // Ports
  tcpPorts: Array<{name, title, containerPort, defaultValue}>,
  udpPorts: Array<{name, title, containerPort, defaultValue}>,
  httpPorts: Array<{name, title, containerPort, defaultValue}>,

  // Metadata
  author, tagline, description, website, contactEmail, tags,
  configurePath, upstreamVersion, postInstallMessage, changelog,
  icon, memoryLimit,

  // OIDC options
  oidcRedirectUri, oidcLogoutUri, oidcTokenAlgo,

  // ProxyAuth options
  proxyauthPath, proxyauthBasicAuth, proxyauthBearerAuth,

  // Database-specific
  mysqlMultipleDbs, mongodbOplog, redisNoPassword, postgresqlLocale,

  // Localstorage
  localstorageFtp, localstorageSqlite, localstorageSqlitePaths,

  // Sendmail
  sendmailOptional, sendmailDisplayName, sendmailValidCert,

  // Scheduler
  schedulerTasks: Array<{name, schedule, command}>,

  // Publishing
  dockerHubUsername, packagerName, packagerUrl, iconUrl,
  mediaLinks, documentationUrl, forumUrl,

  // Advanced
  minBoxVersion, maxBoxVersion, targetBoxVersion,
  capabilities, multiDomain, fullDomain, singleUser,
  dockerfileCloudron, secondarySubdomains,
  runtimeDirs, persistentDirs,
  backupCommand, restoreCommand, logPaths, checklist,

  // Multi-stage
  copyFrom: Array<{image, src, dest}>,

  // Multi-service
  services: Array<{name, command, internalPort, routePath, sso}>,

  // DooD sub-containers
  subcontainers: Array<{image, port, route, memory, volume}>,
}
```

## 4. Wireframes

### Mode Guidé (défaut)

```
┌──────────────────────────────────────────────────┐
│  📦 FastPackCloudron                             │
│                                                  │
│  Quelle image Docker ?                           │
│  [ nginx:latest                            ]     │
│                                                  │
│  Options                                         │
│  ☐ Ajouter une base de données                   │
│     → [PostgreSQL ▾] (visible seulement si coché)│
│  ☐ Activer l'authentification                    │
│     → [OIDC ▾]       (visible seulement si coché)│
│  ☐ Changer le port HTTP                          │
│     → [8000]          (visible seulement si coché)│
│  ☐ Choisir un stack applicatif                   │
│     → [Node.js ▾]    (visible seulement si coché)│
│  ☐ Ajouter une 2e app Docker                     │
│     → (sub-container fields)                     │
│                                                  │
│  ┌─ Résumé ────────────────────────────────┐     │
│  │ Image: nginx:latest                     │     │
│  │ DB: aucune · Auth: aucune · Port: 8000  │     │
│  └─────────────────────────────────────────┘     │
│                                                  │
│  [████████ Générer le package ████████]           │
│                                                  │
│  ↓ Toutes les options (mode avancé)              │
├──────────────────────────────────────────────────┤
│  manifest │ dockerfile │ start.sh │ ...    Copy  │
│  { "id": "io.fastpack.nginx", ... }              │
└──────────────────────────────────────────────────┘
```

### Mode Avancé

```
┌────────────┬─────────────────────────────────────┐
│ Navigation │ Package Cloudron       [🔍 Recherche]│
│            │                                     │
│ ● Général  │ ── Général ───────────────────      │
│   Runtime  │ Docker Image [ nginx:latest    ]    │
│   Réseau   │ Database     [ PostgreSQL ▾    ]    │
│   Base de  │ SSO          [ OIDC ▾         ]    │
│    données │ Stack        [ Node.js ▾      ]    │
│   Auth/SSO │ Web UI       (●) Oui  (○) Non      │
│   Addons   │                                     │
│   Services │ ── Metadata ──────────────────      │
│   Sub-cont.│ App ID    [ io.fastpack.nginx ]     │
│   Ports    │ Title     [ Nginx             ]     │
│   Capabilit│ Version   [ 1.0.0             ]     │
│   Build    │ ...                                 │
│   Publishing│                                    │
│            │ ── Addons ────────────────────      │
│ Résumé     │ ☑ localstorage ☐ sendmail ...      │
│ ┌────────┐ │                                     │
│ │nginx   │ │ ── Services ─────────────────      │
│ │PG,OIDC │ │ (Add Service button)                │
│ │port8000│ │                                     │
│ └────────┘ │ [████ Générer ████] [↑ Mode guidé] │
├────────────┴─────────────────────────────────────┤
│ manifest │ dockerfile │ start.sh │ ...     Copy  │
└──────────────────────────────────────────────────┘
```

## 5. Palette de couleurs

WCAG AA validé. Inspiré Industrial Minimalist.

### Light mode

| Token | Valeur | Usage |
|-------|--------|-------|
| `--fp-primary` | `#0F766E` (teal-700) | Actions, liens, nav active. Ratio 5.1:1 sur blanc |
| `--fp-primary-hover` | `#0D6960` | Hover states |
| `--fp-primary-light` | `#CCFBF1` | Backgrounds actifs |
| `--fp-accent` | `#EA580C` (orange-600) | CTA Générer. Ratio 4.6:1 avec blanc |
| `--fp-accent-hover` | `#C2410C` | Hover CTA |
| `--fp-bg` | `#F8FAFC` | Page background |
| `--fp-surface` | `#FFFFFF` | Cards, form areas |
| `--fp-text` | `#1E293B` | Body text. Ratio 12.6:1 |
| `--fp-text-muted` | `#64748B` | Labels secondaires |
| `--fp-border` | `#E2E8F0` | Bordures |
| `--fp-success` | `#16A34A` | Validation OK |
| `--fp-error` | `#DC2626` | Erreurs |

### Dark mode (prefers-color-scheme)

| Token | Valeur |
|-------|--------|
| `--fp-primary` | `#2DD4BF` |
| `--fp-accent` | `#FB923C` |
| `--fp-bg` | `#0F172A` |
| `--fp-surface` | `#1E293B` |
| `--fp-text` | `#F1F5F9` |
| `--fp-text-muted` | `#94A3B8` |
| `--fp-border` | `#334155` |

### Typographie

`system-ui, -apple-system, sans-serif` — zéro fonts externes.

## 6. Décisions de débat (post-mortem)

| Question débattue | Décision | Rationale |
|-------------------|----------|-----------|
| Wizard 3 étapes ? | **Non** | Unanimité : friction injustifiée pour 1 champ requis |
| 2 UIs séparées ? | **Non** | Unanimité : dette de maintenance insoutenable |
| Google Fonts ? | **Non** | Unanimité : zéro valeur fonctionnelle, régression perf |
| Mode Expert toggle ? | **Oui, mais intelligent** | Le mode Guidé cache, ne supprime pas. Les données persistent au switch. |
| Alpine.js ? | **Oui, vendoré** | Pas de CDN. Copie locale dans le repo. SRI hash. |
| Recettes pré-configurées ? | **Non** | Maintenance non justifiée pour un outil générique |
| Dark mode ? | **Oui** | Déjà existant, maintenu via CSS custom properties |

## 7. Plan d'implémentation (Develop)

### Slice 1 — Mode Guidé minimal
- Nouveau `index.html` avec Alpine.js vendoré
- 1 input Docker + 4 checkboxes optionnelles
- Live summary
- Download ZIP fonctionnel
- Preview tabs (manifest, dockerfile, start.sh)
- Tests E2E Playwright

### Slice 2 — Mode Avancé complet
- Sidebar navigation
- Tous les 50+ champs organisés par section
- Barre de recherche champs
- Transition fluide depuis Mode Guidé

### Slice 3 — Finitions
- localStorage persistence + migration depuis v1
- Responsive mobile
- WCAG AA audit
- Déploiement GitHub Pages

## 8. Critères de succès

- [ ] Novice réussit en 2 min sans doc (scénario nginx)
- [ ] Dev junior configure PG + OIDC en < 3 min
- [ ] Devops accède à tous les 50+ champs
- [ ] 313/313 tests CI passent (régression generators.js)
- [ ] Tests E2E Playwright passent
- [ ] WCAG AA compliant
- [ ] Mobile-friendly
- [ ] GitHub Pages déployé
- [ ] Single file, pas de build step
