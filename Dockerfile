# syntax=docker/dockerfile:1

# ---------- Frontend builder (admin-tools assets) ----------
FROM docker.io/node:24-bookworm-slim AS frontend-builder
WORKDIR /src

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml tsconfig.json vite.config.ts ./
COPY cmd/admin-tools/resources ./cmd/admin-tools/resources

RUN corepack enable && pnpm install --frozen-lockfile
RUN pnpm exec vp build

# ---------- Go builder (all binaries) ----------
FROM docker.io/golang:1.27-bookworm AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/assets/public/build ./assets/public/build

ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN mkdir -p /out && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
      -ldflags "-s -w -X 'go-project-template/buildinfo.Version=${VERSION}'" \
      -o /out/ ./cmd/...

# ---------- Runtime image (distroless non-root, one binary per image) ----------
# Build a role-specific image with:
#   podman build --format=docker --build-arg service=worker -t app-worker .
# Binaries available: server, worker, admin-tools, elasticcli.
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

ARG service=server

COPY --from=go-builder /out/${service} /app/bin/server

USER nonroot:nonroot

ENTRYPOINT ["/app/bin/server"]
