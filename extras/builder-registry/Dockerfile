FROM cloudron/io.cloudron.buildservice:202511060621210000

# Copy registry binary from official image
COPY --from=registry:2 /bin/registry /usr/local/bin/registry

# Install nginx for reverse proxy
RUN apt-get update && apt-get install -y --no-install-recommends nginx && rm -rf /var/lib/apt/lists/*

# Nginx reverse proxy config
COPY nginx.conf /app/code/nginx.conf

# Combined start script
COPY start.sh /app/code/start.sh
RUN chmod +x /app/code/start.sh

EXPOSE 8000

CMD ["/app/code/start.sh"]
