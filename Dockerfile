# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.3-bookworm AS build
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

FROM alpine:3.24.3
LABEL org.opencontainers.image.source="https://github.com/sbolch/AssetGoblin" \
      org.opencontainers.image.description="Serve static files or dynamically manipulated images with ease" \
      org.opencontainers.image.licenses="MIT"

RUN apk add --no-cache ca-certificates python3 py3-pip imagemagick imagemagick-heic vips-tools vips-heif ffmpeg tzdata \
    && addgroup -S -g 10001 assetgoblin \
    && adduser -S -D -H -u 10001 -G assetgoblin -h /srv/assetgoblin assetgoblin \
    && mkdir -p /srv/assetgoblin/assets/img /srv/assetgoblin/public \
        /var/cache/assetgoblin /etc/assetgoblin /usr/local/lib/assetgoblin/plugins \
    && chown -R 10001:10001 /srv/assetgoblin /var/cache/assetgoblin
COPY --from=build /out/assetgoblin /usr/local/bin/assetgoblin
COPY LICENSE /usr/share/doc/assetgoblin/LICENSE

ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    HOME=/srv/assetgoblin \
    LANG=C.UTF-8 \
    XDG_CACHE_HOME=/var/cache
WORKDIR /srv/assetgoblin
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["assetgoblin"]
CMD ["-serve", "-plugins-dir", "/usr/local/lib/assetgoblin/plugins"]
