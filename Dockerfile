# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=development
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" \
    -o /out/assetgoblin .

FROM debian:bookworm-slim
LABEL org.opencontainers.image.source="https://github.com/sbolch/AssetGoblin" \
      org.opencontainers.image.description="Static assets, image transformations and portable Lua plugins" \
      org.opencontainers.image.licenses="MIT"
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates imagemagick libvips-tools ffmpeg \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 assetgoblin \
    && useradd --uid 10001 --gid assetgoblin --no-create-home --home-dir /app assetgoblin \
    && mkdir -p /app/assets/img /app/public /plugins /cache \
    && chown -R assetgoblin:assetgoblin /app /cache
COPY --from=build /out/assetgoblin /usr/local/bin/assetgoblin
COPY LICENSE /usr/share/doc/assetgoblin/LICENSE
ENV XDG_CACHE_HOME=/cache
WORKDIR /app
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["assetgoblin"]
CMD ["-serve", "-plugins-dir", "/plugins"]
