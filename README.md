# Backend

> The vcassist backend.

## Project structure

- `docs/` - additional documentation
- `proto/` - protobuf definitions for services
- `services/` - gRPC services
   - `auth/` - handles the authentication flow, issuing of tokens, and verification codes
      - `verifier/` - exposes utilities to verify authentication tokens
   - `keychain/` - handles storing, retrieving, and refreshing user credentials
   - `linker/` - does data linking
   - `vcsis/` - implementation of a SIS service for Valley Christian Schools
   - `vcmoodle/` - stuff that power quick moodle
      - `server/` - the service that provides an API for reading moodle data
      - `scraper/` - a library that makes it easy to scrape moodle data
- `cmd/` - all official entrypoints/build targets
   - `vc-server/` - a single binary monolith for Valley Christian Schools that strings together all the services under `services/`
   - `vcmoodle-test/` - a testing utility to scrape all the moodle courses
   - `linker-cli/` - the CLI tool for viewing and editing data linker behavior
- `lib/` - shared libraries
   - `scrapers/` - scrapers for various platforms.
      - `moodle/` - [moodle](https://moodle.org/)
      - `powerschool/` - [powerschool](https://powerschool.com/)
      - `vcsnet/` - [vcs.net](https://vcs.net)
   - `gradestore/` - a simple time-series store for grade data.
   - `configutil/` - additional utilities for reading and resolving configuration.
   - `htmlutil/` - additional utilities for working with HTML.
   - `serviceutil/` - additional utilities that are commonly used in service entrypoints.
   - `restyutil/` - utilities for the `resty` HTTP client
   - `sqliteutil/` - utilities for opening up and migrating sqlite databases
   - `timezone/` - `time.Now()` always in the correct timezone, instead of system time. (because sometimes servers are hosted outside of PDT)
   - `telemetry/` - telemetry setup/teardown as well as misc. instrumentation utilities
- `dev/` - code for setting up the development environment
   - `local_stack/` - docker compose stuff for setting up grafana and other things locally
   - `.state/ (gitignore'd)` - internal state (like secrets, usernames, passwords, etc...) that are used by tests and other dev/local-only processes
- `buf.yaml` - [buf.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- `buf.gen-go.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for golang code
- `buf.gen-ts.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for typescript code
- `sqlc.yaml` - [sqlc.yaml](https://docs.sqlc.dev/en/latest/reference/config.html)
- `telemetry.json5` - configuration of telemetry for development

## Docker services

This project relies on grafana + opentelemetry for some of its debugging information, to run the appropriate docker services to make the integration work, you should execute

```sh
go run ./dev
```

in the root directory of the repository.

## Commands

Here are some commands relating to linting and code generation that will probably be useful.

- `go vet ./...` - typecheck all go packages
- `sqlc generate` - generate sql wrapper code with [sqlc](https://sqlc.dev/)
- `connectrpc-otel-gen .` - generates opentelemetry wrapper code for all gRPC services (you should run this every time you change `.proto` files)
- `buf lint` - lint protobuf files with [buf](https://buf.build/)
- `buf format -w` - format protobuf files
- `buf generate --template buf.gen-go.yaml` - generate golang protobuf code with [buf](https://buf.build/)
- `buf generate --template buf.gen-ts.yaml` - generate typescript protobuf code with [buf](https://buf.build/)
- `protogetter --fix ./...` - makes sure you use `Get` methods on protobufs to prevent nil pointer dereference when chaining stuff together

> [!NOTE]
> To use these commands you'll need to install their respective CLI binaries.

- `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `go install github.com/bufbuild/buf/cmd/buf@v1.33.0`
- `go install github.com/ghostiam/protogetter/cmd/protogetter@latest`
- `go install github.com/LQR471814/connectrpc-otel-gen@latest`
- [atlas](https://atlasgo.io/)

## Testing

- `go test ./lib/... ./services/auth` - runs all tests that don't require manual interaction
- `go test -v ./services/vcsis` - runs the tests for powerschool scraping, this is kept separately from the rest of the tests because it requires you to sign in with powerschool manually which doesn't work out that well if you're testing everything all at once
- `go clean -testcache` - cleans test cache, may be useful if telemetry isn't working

> [!NOTE]
> It is not a good idea to run tests from a directory other than root, this is because many temporary and local files are resolved relative to the current working directory so you may find unexpected issues!

