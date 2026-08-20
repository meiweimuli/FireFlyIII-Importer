# AGENTS.md

- **Stack**: Go 1.26+ (Gin backend) + Vue 3 + TypeScript + Element Plus (Frontend).
- **Entrypoints**:
  - Backend: `main.go`
  - Frontend: `frontend/src/main.ts` (root component `frontend/src/App.vue`)

## Key Architecture & Directories
- `api/`: HTTP handlers and routing (`handlers.go`, `import.go`, `upload.go`).
- `parser/`: Alipay (`alipay.go`) and WeChat Pay (`wechat.go`) transaction statement parsers.
- `firefly/`: Firefly III API client integration (`client.go`).
- `models/`: Domain models (`transaction.go`).
- `config/`: Configuration management and preset mapping rules (`config.go`).
- `frontend/`: Vue SPA.

## Build & Run Commands

1. **Build Frontend First** (Required because the Go server serves static assets from `./frontend/dist`):
   ```bash
   cd frontend && npm install && npm run build
   ```

2. **Build & Run Backend**:
   ```bash
   go build -o importer .
   ./importer
   ```
   - Default port: `8080` (override via `PORT` env var).
   - Config directory: defaults to `./data` (override via `CONFIG_DIR` env var).

3. **Docker**:
   ```bash
   docker build -t firefly-importer .
   docker-compose up -d
   ```
