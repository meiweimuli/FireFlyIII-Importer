# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Binary
FROM golang:1.26-alpine AS go-builder
# Install ca-certificates and tzdata to copy them to the scratch image
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build a fully static Go binary with stripped debug info
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o importer .
# Create the data directory in builder so we can copy it to scratch
RUN mkdir -p /app/data

# Stage 3: Pure Static Scratch Image
FROM scratch

# 1. Copy CA certificates for HTTPS requests (otherwise Go cannot connect to the FireFly API)
COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 2. Copy timezone data
COPY --from=go-builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app

# 3. Copy the compiled static binary
COPY --from=go-builder /app/importer .

# 4. Copy the frontend files
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# 5. Copy the empty data directory for configuration persistence
COPY --from=go-builder /app/data ./data
VOLUME /app/data

# 6. Set Environment Variables
ENV CONFIG_DIR=/app/data
ENV GIN_MODE=release
ENV PORT=8080
ENV TZ=Asia/Shanghai

EXPOSE 8080

CMD ["./importer"]
