# Contributing

Contributions are welcome! Here's how to get started.

## Development Setup

No build step is required for the app itself. Serve the directory:

```bash
cd fastpack-cloudron
python3 -m http.server 8080
```

Open `http://localhost:8080` to see the app, and `http://localhost:8080/test.html` to run tests.

Install test dependencies before running the headless suite:

```bash
npm ci
npx playwright install chromium
```

## Project Structure

```
index.html      HTML structure + embedded CSS
app.js          UI logic (form handling, validation, preview, ZIP download)
generators.js   Pure functions that generate Cloudron package files
test.html       In-browser test suite
deploy-cli/     Go deploy CLI
```

## Tests

Run the suites that match your change:

```bash
npm test                 # Browser generator tests
npm run test:build       # Docker build checks
npm run test:go          # Go CLI tests
```

Live Cloudron E2E scripts require private `FASTPACK_E2E_*` environment variables and skip when they are not configured.

## Guidelines

- **No build tools.** This project intentionally uses vanilla HTML/CSS/JS with zero build step.
- **No innerHTML.** Use `createElement`, `textContent`, `appendChild` for all DOM manipulation.
- **Pure generators.** Functions in `generators.js` must remain pure (string in, string out).
- **Test everything.** Add tests in `test.html` for any new generator logic.
- **Follow Cloudron conventions.** Generated files must comply with [Cloudron packaging docs](https://docs.cloudron.io/packaging/).

## Submitting Changes

1. Fork the repo
2. Create a feature branch
3. Make your changes
4. Ensure the relevant tests pass
5. Submit a pull request
