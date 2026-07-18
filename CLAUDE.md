# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

DDNS is a small Go service that watches the local machine's network interfaces and keeps a
Cloudflare DNS A record pointed at the current public IP. It's built with `go.uber.org/fx` for
dependency injection/lifecycle management.

## Platform constraint: Linux only

`pkg/netlink` talks to the kernel directly via `AF_NETLINK` sockets (`golang.org/x/sys/unix` +
the standard `syscall` package's netlink helpers). This only compiles and runs on Linux.

- On a non-Linux dev machine (e.g. macOS), `go build`/`go vet` still work if you cross-compile:
  `GOOS=linux GOARCH=amd64 go build ./...` — but the resulting binary/test can't be *executed*
  locally (`exec format error`).
- To actually run tests on a non-Linux host, build and run inside a Linux container. If `docker`
  resolves to a remote (non-local) context — check with `docker context ls` — bind mounts of the
  local working directory won't work because the path doesn't exist on the remote host; instead
  build an image that `COPY`s the source in:
  ```
  FROM golang:1.26      # match the `go` directive in go.mod
  WORKDIR /src
  COPY . .
  CMD ["go", "test", "./...", "-v"]
  ```
  then `docker build` + `docker run` that image.

## Commands

- Build: `go build ./...` (or `go build -o ddns ./cmd` for the binary)
- Run all tests: `go test ./...`
- Run one package's tests: `go test ./pkg/ddns/...`
- Run a single test: `go test ./pkg/ddns/... -run TestRun -v`
- Match CI exactly (race + coverage): `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
- Vet: `go vet ./...`
- Regenerate mocks after changing an interface: `mockery` (config in `.mockery.yml`)

## Architecture

Three packages, each following the same `New`/`Invoke` shape so `cmd/main.go` can wire them
uniformly with fx:

- `New(...)` — a constructor registered via `fx.Provide`, returning an interface (not the
  concrete type).
- `Invoke(lifecycle fx.Lifecycle, ...)` — registered via `fx.Invoke`, appends `fx.StartHook` /
  `fx.StartStopHook` callbacks to wire the component into the app's start/stop lifecycle.

Data flow: `pkg/netlink` → `pkg/ddns` → `pkg/cloudflare`.

- **`pkg/netlink`**: opens a raw `NETLINK_ROUTE` socket, subscribes to address/link change
  groups, and does an initial `RTM_GETADDR` dump. The `Watcher` interface exposes a
  `Watch() <-chan net.IP` channel that emits an IP every time an `RTM_NEWADDR` message arrives
  (address removals and link up/down events are intentionally not forwarded — only new address
  assignments). The channel is closed when the watcher stops.
- **`pkg/ddns`**: owns the `Client` interface (the DNS-provider abstraction) and `run()`, which
  consumes `netlink.Watcher.Watch()`, filters out private/loopback/link-local addresses, prefers
  IPv4 over IPv6, and — once a candidate IP is found — calls `updateIP`. `updateIP` dedupes
  against the last-seen IP (via an atomic pointer) before querying/updating the DNS record, so
  the Cloudflare API is only hit when the address actually changes.
- **`pkg/cloudflare`**: implements `ddns.Client` on top of the official Cloudflare Go SDK v6. Its
  `Invoke` registers an `fx.StartHook` that resolves the zone ID (`ddns.Config.ZoneName` →
  Cloudflare zone ID) once at startup, before `ddns.Invoke`'s consumer loop begins.

Config (`pkg/ddns/config.go`) is parsed from environment variables via `caarlos0/env`:
`CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_EMAIL`, `DNS_ZONE` (`CLOUDFLARE_API_TOKEN`/`_EMAIL` are read
directly by the Cloudflare SDK, not by `Config`). `pkg/logger` additionally reads `LOG_LEVEL`
(`debug`/`info`/`warn`/`error`, default `info`). Detection is event-driven off netlink — there is
no polling interval to configure.

## Mocks

Mocks are generated per-package by `mockery` (testify template) into a `mocks_test.go` file
**in the same package as the interface**, e.g. `pkg/netlink/mocks_test.go` mocks
`netlink.Watcher`. Because these are `_test.go` files, **they are not importable from another
package's tests** — e.g. `pkg/ddns`'s tests can't import `netlink`'s generated `MockWatcher`.
When a test in one package needs to satisfy an interface defined in another package, write a
small hand-rolled fake locally instead of trying to reuse the other package's mock.

## CI

- `.github/workflows/test.yml`: runs on every push/PR, `go mod download` + `verify`, then
  `go test -race` with coverage.
- `.github/workflows/release.yml`: triggered on `v*` tags, cross-compiles `CGO_ENABLED=0`
  binaries for linux/amd64 and linux/arm64 (Darwin isn't buildable — see the platform constraint
  above), generates checksums, and creates a GitHub release.
- Both use `actions/setup-go` with `go-version-file: 'go.mod'`, so the Go version follows the
  `go` directive in `go.mod` — bump that instead of editing the workflows.
