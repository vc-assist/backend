## vcassist-backend

> WIP rewrite of the vcassist backend in golang.

### project structure

- `docs/` - documentation
- `cmd/` - all official entrypoints/build targets
- `lib/` - shared libraries
   - `platforms/` - bootleg APIs for various platforms
- `dev/` - code for setting up the development environment
   - `local_stack/` - docker compose stuff for setting up grafana and other things locally
   - `.state/ (gitignore'd)` - internal state (like secrets, usernames, passwords, etc...) that are used by tests and other dev/local-only processes
- `buf.yaml` - [buf.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- `buf.gen.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- `sqlc.yaml` - [sqlc.yaml](https://docs.sqlc.dev/en/latest/reference/config.html)
- `telemetry.json5` - configuration of telemetry for development

### development environment

this project comes with its own custom development environment (which is basically just setup work that has to be done in order for a variety of things to work locally).

the code that initializes this environment is kept under `dev/`.

here are a few things it sets up:

1. an empty sqlite database + migrations for `cmd/powerschool_api/`
2. a local setup of telemetry using Docker Compose under `dev/local_stack/`
3. moodle credentials for use in testing in `lib/platforms/moodle/...`

as such, before running tests or doing local debugging you should run one of the following commands.

- `go run ./dev` - setup development environment
- `go run ./dev -recreate` - recreate development environment (bypass cache effectively)

### commands

here are some commands relating to linting and code generation that will probably be useful.

- `sqlc generate` - generate sql wrapper code with [sqlc](https://sqlc.dev/)
- `buf generate` - generate protobuf files with [buf](https://buf.build/)
- `buf lint` - lint protobuf files with [buf](https://buf.build/)
- `protogetter --fix ./...` - makes sure you use `Get` methods on protobufs to prevent segmentation faults when chaining stuff together

> [!NOTE]
> to use these commands you'll need to install their respective dependencies

- `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `go install github.com/bufbuild/buf/cmd/buf@v1.33.0`
- `go install github.com/ghostiam/protogetter/cmd/protogetter@latest`
- [altas](https://atlasgo.io/getting-started#installation)

### testing

- `go test ./lib/...` - runs all tests in shared packages
- `go test ./cmd/powerschool_api` - runs the tests for the Powerschool API, this is kept separately from `go test ./lib/...` because it requires you to sign in with powerschool manually which doesn't work out that well if you're testing everything all at once

