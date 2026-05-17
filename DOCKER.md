# Docker Deployment Guide

## Quick Start

### Build the Image

```bash
docker build -t kuons/fireflyiii-importer .
```

### Run with Docker

```bash
docker run -d \
  --name firefly-importer \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  kuons/fireflyiii-importer
```

Access the app at: **http://localhost:8080**

---

## Docker Compose

Create a `docker-compose.yml`:

```yaml
version: "3.8"

services:
  firefly-importer:
    build: .
    # Or use a pre-built image:
    # image: kuons/fireflyiii-importer:latest
    container_name: firefly-importer
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - CONFIG_DIR=/app/data
      - GIN_MODE=release
      - PORT=8080
      - TZ=Asia/Shanghai
    restart: unless-stopped
```

Then run:

```bash
docker-compose up -d
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CONFIG_DIR` | `/app/data` | Directory for config.json and backups |
| `PORT` | `8080` | Server listening port |
| `GIN_MODE` | `release` | Gin framework mode (`release` / `debug`) |
| `TZ` | `UTC` | Timezone (e.g., `Asia/Shanghai`) |

### Volume Mount

The `/app/data` directory contains:

- `config.json` — Main configuration file (Firefly URL, token, mapping rules, etc.)
- `config-backup-*.json` — Automatic backups created when importing configs

Mount this directory to persist your settings across container restarts:

```bash
-v /path/on/host/data:/app/data
```

---

## Network Configuration

If your Firefly III instance runs in Docker too, make sure they can communicate:

### Option 1: Docker Network

```bash
# Create a shared network
docker network create firefly-net

# Run Firefly III on this network
docker run --network firefly-net --name firefly-iii ...

# Run Importer on the same network
docker run --network firefly-net \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  kuons/fireflyiii-importer
```

Then use `http://firefly-iii:8080` as the Firefly URL in settings.

### Option 2: Host Network

```bash
docker run --network host \
  -v $(pwd)/data:/app/data \
  kuons/fireflyiii-importer
```

Then use `http://localhost:8080` (or your Firefly III's actual address).

---

## Updating

```bash
# Pull latest code
git pull

# Rebuild
docker-compose build

# Restart
docker-compose up -d
```

---

## Troubleshooting

### Cannot connect to Firefly III

- Check the Firefly URL in Settings (must be reachable from the container)
- If both are in Docker, use Docker network names instead of `localhost`
- Verify your Personal Access Token is valid

### Config not persisting

- Ensure the volume mount is correct: `-v /path/data:/app/data`
- Check file permissions: the container runs as root by default

### Import timeout

- The HTTP timeout is set to 60 seconds per transaction
- If your Firefly III is slow, consider upgrading its resources
