# syntax=docker/dockerfile:1

# --- Stage: frontend-webui — builds the Web UI SPA embedded by internal/webui ---
# vite.config.ts's outDir ('../internal/webui/dist') is relative to the web/
# source dir, so both dirs are placed as siblings here to match that layout.
FROM node:22-alpine AS frontend-webui
WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN cd web && npm ci
COPY web/ ./web/
RUN cd web && npm run build

# --- Stage: frontend-compose — builds the `maelsink compose` SPA embedded by internal/compose ---
FROM node:22-alpine AS frontend-compose
WORKDIR /src
COPY web-compose/package.json web-compose/package-lock.json ./web-compose/
RUN cd web-compose && npm ci
COPY web-compose/ ./web-compose/
RUN cd web-compose && npm run build

# --- Stage: builder — compiles the static Go binary with both SPAs embedded ---
FROM golang:1.26-alpine AS builder
WORKDIR /src

ARG VERSION=0.0.0
ARG COMMIT=none
ARG BUILD_DATE=unknown

COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-webui /src/internal/webui/dist ./internal/webui/dist
COPY --from=frontend-compose /src/internal/compose/dist ./internal/compose/dist

RUN CGO_ENABLED=0 go build \
    -ldflags "-X github.com/0funct0ry/maelsink/internal/version.Version=${VERSION} \
              -X github.com/0funct0ry/maelsink/internal/version.Commit=${COMMIT} \
              -X github.com/0funct0ry/maelsink/internal/version.BuildDate=${BUILD_DATE} \
              -s -w" \
    -o /maelsink .

RUN mkdir /data

# --- Stage: runtime — minimal distroless image, binary only ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=builder /maelsink /maelsink
COPY --from=builder --chown=nonroot:nonroot /data /data

# Hosts default to 127.0.0.1 (loopback) in maelsink.yaml/CLI defaults (SPEC.md
# §3.1, §12) — override to 0.0.0.0 here so the EXPOSEd ports are reachable
# through Docker's -p host:container mapping. storage.path defaults to a
# relative ./maelsink.db, which would otherwise land outside the /data volume.
ENV MAELSINK_SMTP_HOST=0.0.0.0
ENV MAELSINK_WEB_HOST=0.0.0.0
ENV MAELSINK_API_HOST=0.0.0.0
ENV MAELSINK_STORAGE_PATH=/data/maelsink.db

EXPOSE 1025 8080 9090
VOLUME ["/data"]

ENTRYPOINT ["/maelsink", "serve"]
