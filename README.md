# FastPackCloudron

Generate ready-to-deploy Cloudron custom app packages in 3 clicks.

## What is this?

A static web app that generates all the files needed to package any Docker image as a Cloudron custom app:
- `CloudronManifest.json`
- `Dockerfile`
- `start.sh`
- `.dockerignore`
- `README.md` with deployment instructions

## Usage

1. Open FastPackCloudron
2. Paste a Docker image name (e.g., `eclipse-mosquitto:2.0`)
3. Pick your options (database, SSO, ports)
4. Download the ZIP
5. Extract and deploy:
   ```bash
   cloudron login my.example.com
   cloudron build
   cloudron install
   ```

## Development

No build step required. Serve the directory with any static file server:

```bash
cd fastpack-cloudron
python3 -m http.server 8080
```

Open `http://localhost:8080` in your browser.

Run tests: open `http://localhost:8080/test.html`.

## License

MIT
