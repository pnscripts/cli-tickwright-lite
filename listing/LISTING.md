# Tickwright Lite: GitHub Releases and Docker Hub

Nothing has been published.

## GitHub Releases (free, ready)

1. Owner: create the public repo `github.com/pnscripts/tickwright-lite`, push `main`, then push the tag `v1.0.0`.
2. The Release workflow runs the tests, builds six archives + SHA256SUMS (`make dist`) and creates the release. Public repos do not use the org's $0 Actions budget. Local build verified: `dist/` holds the same files.
3. Repo settings: description from README, topics `cron crontab cli golang timezone`, social preview `listing/social-preview-1280x640.png`.

## Docker Hub (free Personal plan, public repositories unlimited)

1. Owner: create a Docker Hub account/namespace `pnscripts` (Personal plan; an organization is a paid Team plan, so use the personal namespace named pnscripts). Create the public repository `pnscripts/tickwright-lite`, short description (≤ 100 chars): "Validate and explain cron expressions and list their next runs in any time zone."
2. Create an access token (Read & Write) and add it to the GitHub repo: variable `DOCKERHUB_USERNAME=pnscripts`, secret `DOCKERHUB_TOKEN`. The next tag push builds linux/amd64 + linux/arm64 and pushes `latest` and the version tag.
3. Docker Hub overview: paste README.md.

The Dockerfile was not built on this machine (no Docker installed); the binary it builds is the same `go build` that `make dist` ran and tested.
