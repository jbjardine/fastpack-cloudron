# UI Redesign Brief — "So Easy a 10-Year-Old Can Use It"

## Vision

Refonte complète de l'UI FastPackCloudron pour que le workflow complet (package + deploy) soit faisable en 3 clicks, que ce soit en mode simple ou DooD.

## Problemes actuels

1. **Trop de champs visibles** — Le formulaire montre beaucoup d'options qui intimident
2. **Pas de wizard/stepper** — Tout est sur une seule page
3. **Pas de mode guide** — L'utilisateur doit savoir quoi remplir
4. **Le deploy est externe** — Il faut telecharger le ZIP, extraire, ouvrir un terminal
5. **DooD est "advanced"** — Devrait etre aussi simple que le mode normal

## Proposition : Wizard en 3 etapes

### Etape 1 : "Quelle app ?"
```
┌─────────────────────────────────────────┐
│  Quelle app Docker voulez-vous          │
│  deployer sur Cloudron ?                │
│                                         │
│  [ nginx:latest          ] [Suivant →]  │
│                                         │
│  Ou choisissez une recette :            │
│  [WordPress] [n8n] [Ghost] [Gitea]      │
│  [NextCloud] [Custom...]                │
└─────────────────────────────────────────┘
```

### Etape 2 : "Options"
Smart defaults, progressive disclosure :
```
┌─────────────────────────────────────────┐
│  nginx:latest                           │
│                                         │
│  Database ?     [Aucune ▾]              │
│  Authentification ? [Aucune ▾]          │
│                                         │
│  [+ Ajouter une 2eme app Docker]  ← DooD│
│                                         │
│  [← Retour]              [Generer →]    │
└─────────────────────────────────────────┘
```

Le DooD n'est plus un mode separe — c'est juste "Ajouter une 2eme app Docker".

### Etape 3 : "Deployer"
```
┌─────────────────────────────────────────┐
│  ✅ Package pret !                      │
│                                         │
│  [Deployer sur mon Cloudron]            │
│      OU                                 │
│  [Telecharger le ZIP]                   │
│                                         │
│  Preview : [manifest] [dockerfile] ...  │
└─────────────────────────────────────────┘
```

## Principes de design

1. **Progressive disclosure** — Montrer le minimum, cacher la complexite
2. **Smart defaults** — Tout est pre-rempli intelligemment
3. **Pas de jargon** — "Ajouter une 2eme app" au lieu de "DooD sub-container"
4. **Wizard > Formulaire** — Etapes guidees au lieu d'un long formulaire
5. **Deploy integre** — Pas besoin de terminal si possible
6. **Recipe Gallery** — Les combos pre-testees en un click
7. **Mobile-friendly** — Utilisable sur telephone

## User Personas

### Alice (10 ans)
- Veut installer Minecraft server sur le Cloudron de papa
- Ne sait pas ce qu'est un Dockerfile
- Doit pouvoir le faire en 3 clicks

### Bob (dev junior)
- A une image Docker custom
- Sait ce qu'est un port et une database
- Veut packager sans lire la doc Cloudron

### Charlie (devops)
- Veut combiner n8n + FastAPI + Redis
- Connait Docker Compose
- Veut le mode expert avec tous les reglages

## Prochaine session : implementer

1. `/octo:design-ui-ux` pour le design system complet
2. `/octo:embrace` pour l'implementation
3. Test sur le Cloudron de test
