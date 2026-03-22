# syntax=docker/dockerfile:1

# ---------- Frontend builder (admin-tools assets) ----------
FROM docker.io/node:24-bookworm-slim AS frontend-builder
WORKDIR /src

COPY package.json pnpm-lock.yaml tsconfig.json vite.config.ts ./
COPY cmd/admin-tools/resources ./cmd/admin-tools/resources

RUN corepack enable && pnpm install --frozen-lockfile
RUN pnpm exec vp build

# ---------- Go builder (all binaries) ----------
FROM docker.io/golang:1.25-bookworm AS go-builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/assets/public/build ./assets/public/build

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
ARG IMAGE_TAG=dev
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN mkdir -p /out && \
    for service in admin-tools elasticcli server worker; do \
      CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
      go build -trimpath \
        -ldflags "-s -w -X 'go-project-template/buildinfo.Version=${VERSION}' -X 'go-project-template/buildinfo.Commit=${COMMIT}' -X 'go-project-template/buildinfo.BuildDate=${BUILD_DATE}' -X 'go-project-template/buildinfo.ImageTag=${IMAGE_TAG}' -X 'go-project-template/buildinfo.Service=${service}'" \
        -o "/out/${service}" "./cmd/${service}"; \
    done

# ---------- Runtime image (single image with all binaries, distroless non-root) ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=go-builder /out/ /app/bin/

USER nonroot:nonroot

# Default binary. In Kubernetes/podman, override entrypoint/command to run another binary.
ENTRYPOINT ["/app/bin/server"]
