# syntax=docker/dockerfile:1
# Багатоетапна збірка: окремий етап для тестів, мінімальний non-root образ для production.
ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-alpine AS base
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .

FROM base AS test
RUN CGO_ENABLED=0 go test ./...

FROM base AS builder
ARG VERSION=dev
ARG VCS_REF=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${VCS_REF}" \
    -o /out/roomflow ./cmd/roomflow \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/frontend ./cmd/frontend

FROM gcr.io/distroless/static-debian12:nonroot AS production
ARG VERSION=dev
ARG VCS_REF=unknown
ARG BUILD_DATE
LABEL org.opencontainers.image.title="roomflow" \
      org.opencontainers.image.source="https://github.com/sudo97/roomflow" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.created="${BUILD_DATE}"
COPY --from=builder /out/roomflow /out/frontend /
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=10s --timeout=2s CMD ["/roomflow", "-healthcheck"]
ENTRYPOINT ["/roomflow"]
