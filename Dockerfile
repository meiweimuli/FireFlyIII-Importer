# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Binary
FROM golang:1.23-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -o importer .

# Stage 3: Production Image
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/importer .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Config directory for volume mounting
RUN mkdir -p /app/data
VOLUME /app/data

ENV CONFIG_DIR=/app/data
ENV GIN_MODE=release
ENV PORT=8080

EXPOSE 8080

CMD ["./importer"]
