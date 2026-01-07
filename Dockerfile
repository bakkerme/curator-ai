ARG GO_VERSION=1.25.5

FROM golang:${GO_VERSION} AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Build a small, static-ish binary (CGO disabled) suitable for distroless.
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/curator ./cmd/curator

FROM debian:bookworm-slim AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --uid 10001 curator

WORKDIR /app

COPY --from=build /out/curator /usr/local/bin/curator

USER 10001:10001

# Default config location (can be overridden by flag or env).
ENV CURATOR_CONFIG=/app/curator.yaml

ENTRYPOINT ["/usr/local/bin/curator"]
CMD ["-config", "/app/curator.yaml"]

FROM golang:${GO_VERSION} AS dev
WORKDIR /app

RUN useradd --create-home --uid 10001 curator

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

CMD ["/go/bin/air", "-c", "/app/air.toml"]
