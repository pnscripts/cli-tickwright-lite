# Tickwright Lite by PN Scripts: a static binary on an empty image (about 6 MB).
#   docker run --rm pnscripts/tickwright-lite explain "*/15 9-17 * * MON-FRI"
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cron ./cron
COPY cmd ./cmd
ARG VERSION=1.0.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/tickwright-lite ./cmd/tickwright-lite

FROM scratch
LABEL org.opencontainers.image.title="Tickwright Lite" \
      org.opencontainers.image.description="Validate and explain cron expressions (English and Bulgarian) and list their next runs in any time zone." \
      org.opencontainers.image.vendor="PN Scripts" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.url="https://pnscripts.com/marketplace/tickwright" \
      org.opencontainers.image.source="https://github.com/pnscripts/tickwright-lite"
COPY --from=build /out/tickwright-lite /tickwright-lite
COPY LICENSE /LICENSE
USER 65534:65534
ENTRYPOINT ["/tickwright-lite"]
CMD ["help"]
