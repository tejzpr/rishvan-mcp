# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npx vite build

# Stage 2: Build Go binary
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o rishvan-mcp .

# Stage 3: Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/rishvan-mcp /usr/local/bin/rishvan-mcp
ENTRYPOINT ["rishvan-mcp"]
