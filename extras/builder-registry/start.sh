#!/bin/bash
set -eu -o pipefail

export DOCKER_HOST="${CLOUDRON_DOCKER_HOST}"

# Accept self-signed certificates on test/local Cloudron instances
export NODE_TLS_REJECT_UNAUTHORIZED=0

# Initialize data directories
mkdir -p /app/data/registry

if [[ ! -f /app/data/docker.json ]]; then
    cat > /app/data/docker.json <<DEOF
{
}
DEOF
fi

# Generate registry config with correct external hostname
cat > /tmp/registry-config.yml <<REOF
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /app/data/registry
  delete:
    enabled: true
http:
  addr: 127.0.0.1:5000
  headers:
    X-Content-Type-Options: [nosniff]
  relativeurls: true
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
REOF

chown -R cloudron:cloudron /app/data

# Forward signals to all children
trap 'kill $(jobs -p) 2>/dev/null; wait' TERM INT

# Start registry on port 5000 (internal only)
echo "=> Starting Docker Registry..."
/usr/local/bin/gosu cloudron:cloudron /usr/local/bin/registry serve /tmp/registry-config.yml &

# Start builder on port 3000
echo "=> Starting Build Service..."
/usr/local/bin/gosu cloudron:cloudron /app/code/app.js &

# Start nginx reverse proxy on port 8000
echo "=> Starting nginx reverse proxy..."
nginx -c /app/code/nginx.conf -g "daemon off;" &

wait
