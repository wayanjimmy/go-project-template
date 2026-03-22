FROM docker.io/golang:1.22.2 AS builder

WORKDIR /app/blacklist
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o=./bin/ ./cmd/...

FROM docker.io/alpine:3.14
ARG service=dh

ENV SERVICE $service

COPY --from=builder /app/blacklist/bin/${SERVICE} /server

ENTRYPOINT ["/server"]
