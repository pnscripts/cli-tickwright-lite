# Changelog

## 1.0.2 (2026-11-08)

- `tickwright-lite version` shows the installed version for `go install` too (it showed 1.0.0).
- The help text names the command as installed, `tickwright-lite`.

## 1.0.1 (2026-11-08)

- `go install github.com/pnscripts/cli-tickwright-lite/cmd/tickwright-lite@latest` now works: the module path matches the repository, so the tool is also listed on pkg.go.dev.
- README: Docker instructions build the image from the repository until it is published on Docker Hub.

## 1.0.0 (2026-10-12)

- First release of the Lite edition: `validate`, `explain` (English and Bulgarian) and `next` (any IANA zone, DST-correct), JSON output, static binaries and a Docker image.
