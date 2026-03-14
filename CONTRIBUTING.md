# Contributing

Contributions are welcome! Here's how to get started.

## Development Setup

No build step required. Just serve the directory:

```bash
cd fastpack-cloudron
python3 -m http.server 8080
```

Open `http://localhost:8080` to see the app, and `http://localhost:8080/test.html` to run tests.

## Project Structure

```
index.html      HTML structure + embedded CSS
app.js          UI logic (form handling, validation, preview, ZIP download)
generators.js   Pure functions that generate Cloudron package files
test.html       In-browser test suite
```

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
4. Ensure all tests pass in `test.html`
5. Submit a pull request
