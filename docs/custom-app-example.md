# Custom App Example

Use this guide when your app has source files such as `app.py`, `requirements.txt`, or templates.

FastPack Deploy CLI is optimized for FastPack-generated packages and uploads a strict allow-list of package files. For custom source apps, use the official Cloudron CLI from your source directory so the full build context is included.

## Example App

```text
my-flask-app/
├── app.py
├── requirements.txt
└── templates/
    └── index.html
```

`app.py`:

```python
from flask import Flask, render_template

app = Flask(__name__)

@app.route("/")
def home():
    return render_template("index.html")

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000)
```

`requirements.txt`:

```text
flask==3.1.0
gunicorn==23.0.0
```

## Generate the Cloudron Files

In FastPackCloudron, use:

| Field | Value |
|-------|-------|
| Docker Image | `python:3.12-slim` |
| App Title | `My Flask App` |
| Stack | Python |
| Web Interface | Yes |
| HTTP Port | `8000` |
| Storage | localstorage |

Download the ZIP and copy its Cloudron files into your source directory.

## Dockerfile

Adjust the generated `Dockerfile` so it copies your source files:

```dockerfile
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends gosu \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r cloudron -g 808 \
    && useradd -r -g cloudron -u 808 -d /app cloudron

COPY requirements.txt /app/code/requirements.txt
RUN pip install --no-cache-dir -r /app/code/requirements.txt

COPY app.py /app/code/app.py
COPY templates/ /app/code/templates/
COPY start.sh /app/code/start.sh
RUN chmod +x /app/code/start.sh

EXPOSE 8000
CMD ["/app/code/start.sh"]
```

`start.sh` should start the app as the `cloudron` user:

```bash
#!/bin/bash
set -e

mkdir -p /app/data
chown -R cloudron:cloudron /app/data

cd /app/code
exec gosu cloudron:cloudron gunicorn --bind 0.0.0.0:8000 --workers 2 app:app
```

## Deploy

Run the Cloudron CLI from the source directory:

```bash
npm install -g cloudron
cloudron login my.example.com
cloudron build
cloudron install
```

For updates:

```bash
cloudron build
cloudron update --app my-flask-app.example.com
```

## Notes

- Use `/app/data` for persistent files.
- Keep the app listening on the manifest `httpPort`.
- For OIDC, Cloudron injects `CLOUDRON_OIDC_CLIENT_ID`, `CLOUDRON_OIDC_CLIENT_SECRET`, and `CLOUDRON_OIDC_ISSUER`.
