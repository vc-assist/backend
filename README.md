# Backend

> The vcassist backend.

## Project structure

- `docs/` - additional documentation
- `services/` - gRPC services
   - `auth/` - the service that handles the authentication flow, issuing of tokens and verification codes.
   - `keychain/` - the service that handles storing, retrieving, and keeping fresh user credentials
   - `powerservice/` - the service that fetches a student's powerschool data given a valid key in keychain.
   - `gradesnapshots/` - a service that can store and retrieve grade snapshots.
   - `linker/` - a service that does hybrid automatic and manual data linking
   - `studentdata/` - the interface all student data providers must fulfill to talk to the frontend
   - `vcsmoodle/` - the service for getting student data specific to vcs flavored moodle
   - `vcs/` - the student data provider for vcs
- `cmd/` - all official entrypoints/build targets
   - `auth/` - the entrypoint to the `auth` service
   - `vcs/` - the entrypoint to the `vcs` service
- `lib/` - shared libraries
   - `scrapers/` - scrapers for various platforms
- `dev/` - code for setting up the development environment
   - `local_stack/` - docker compose stuff for setting up grafana and other things locally
   - `.state/ (gitignore'd)` - internal state (like secrets, usernames, passwords, etc...) that are used by tests and other dev/local-only processes
- `buf.yaml` - [buf.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- `buf.gen-go.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for golang code
- `buf.gen-ts.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for typescript code
- `sqlc.yaml` - [sqlc.yaml](https://docs.sqlc.dev/en/latest/reference/config.html)
- `telemetry.json5` - configuration of telemetry for development

## Development environment

This project comes with its own custom development environment (which is basically just setup work that has to be done in order for a variety of things to work locally).

The code that initializes this environment is kept under `dev/`.

Here are a few things it sets up:

1. An empty sqlite database + migrations for all the services under `services/`.
2. A local setup of telemetry using Docker Compose under `dev/local_stack/`, you can access the grafana dashboard at `http://localhost:3000`.
3. Moodle credentials for use in testing in `lib/scrapers/moodle/...`

As such, before running tests or doing local debugging you should run one of the following commands.

- `go run ./dev` - setup development environment
- `go run ./dev -recreate` - recreate development environment (bypass cache effectively)

## Commands

Here are some commands relating to linting and code generation that will probably be useful.

- `go vet ./...` - typecheck all go packages
- `sqlc generate` - generate sql wrapper code with [sqlc](https://sqlc.dev/)
- `connectrpc-otel-gen .` - generates opentelemetry wrapper code for all gRPC services
- `buf lint` - lint protobuf files with [buf](https://buf.build/)
- `buf format -w` - format protobuf files
- `buf generate --template buf.gen-go.yaml` - generate golang protobuf code with [buf](https://buf.build/)
- `buf generate --template buf.gen-ts.yaml` - generate typescript protobuf code with [buf](https://buf.build/)
- `protogetter --fix ./...` - makes sure you use `Get` methods on protobufs to prevent segmentation faults when chaining stuff together
- `atlas schema apply -u "libsql://<db_url>?authToken=<auth_token>" --to file://path/to/schema.sql --dev-url "sqlite://dev?mode=memory"` - migrates a database, see [declarative migrations](https://atlasgo.io/getting-started/#declarative-migrations)

> [!NOTE]
> To use these commands you'll need to install their respective CLI binaries.

- `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `go install github.com/bufbuild/buf/cmd/buf@v1.33.0`
- `go install github.com/ghostiam/protogetter/cmd/protogetter@latest`
- `go install github.com/LQR471814/connectrpc-otel-gen@latest`
- [atlas](https://atlasgo.io/)

## Testing

- `go test -v ./lib/... ./services/auth ./services/vcsmoodle` - runs all tests that don't require manual interaction
- `go test -v ./services/powerservice` - runs the tests for the powerschool service, this is kept separately from the rest of the tests because it requires you to sign in with powerschool manually which doesn't work out that well if you're testing everything all at once
- `go clean -testcache` - cleans test cache, may be useful if telemetry isn't working

