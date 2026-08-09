# Changelog

Notable changes, newest first.

## 1.0.0 - 2026-08-09

- Chart resources are now scoped to the release name, so two installs can coexist in one namespace (previously everything was hardcoded).
- Fixed a port mismatch that could leave the health probe pointing at the wrong port.
- Changing config or secrets and running an upgrade now actually restarts the pods, instead of silently leaving them on stale values.
- Added resource limits, a liveness probe, and pod/container security hardening (non-root, read-only filesystem, dropped capabilities) - all configurable, sensible by default.
- The API token is now mandatory - the chart refuses to deploy without one instead of shipping a guessable default. The SD token stays optional, matching how the app itself treats it.
- Fixed a broken Prometheus job-label setting that silently did nothing.
- Documented that the bundled Redis sidecar only works with a single replica; scaling out needs an external Redis.
- Renamed a couple of typo'd template files.
- Fixed a batch of bugs found in a full code review: wrong HTTP status codes on delete/unsupported methods, Redis errors that were silently swallowed instead of surfaced, a UUID collision bug, a fragile config-loading mechanism, and a mislabeled Prometheus metric.
- Sped up target lookups by batching Redis calls instead of fetching one at a time.
- Docker image now runs as a non-root user.
- Added a full unit test suite - there wasn't one before.
- Bumped Go dependencies (Redis client, Prometheus client, and others) to their latest compatible versions.

## 0.4.0 and earlier

See git history.
