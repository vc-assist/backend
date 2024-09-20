# Backend

> The vcassist backend.

## Usage

1. Install [atlas](https://atlasgo.io/).
2. Run `go run ./dev` at the root of the repo, this starts grafana and some other telemetry stuff in a docker-compose.
3. `cd cmd/vc-server` and `go run . -v`.

## Project structure

- `docs/` - additional documentation
- `proto/` - protobuf definitions for services
- `cmd/` - gRPC services: entrypoint + configuration loading + telemetry init
   - `vc-server/` - a single binary monolith for Valley Christian Schools that strings together all the services under `services/`
   - `vcmoodle-test/` - a testing utility to scrape all the moodle courses
   - `linker-cli/` - the CLI tool for viewing and editing data linker behavior
- `services/` - gRPC services: actual logic
   - `auth/` - handles the authentication flow, issuing of tokens, and verification codes
      - `verifier/` - exposes utilities to verify authentication tokens
   - `keychain/` - handles storing, retrieving, and refreshing user credentials
   - `linker/` - does data linking
   - `vcsis/` - implementation of a SIS service (fancy name for PowerSchool) for Valley Christian Schools
   - `vcmoodle/` - stuff that powers quick moodle
      - `server/` - the service that provides an API for reading moodle data
      - `scraper/` - a library that makes it easy to scrape moodle data
- `lib/` - shared libraries
   - `scrapers/` - bootleg APIs for various platforms.
      - `moodle/` - [moodle](https://moodle.org/)
      - `powerschool/` - [powerschool](https://powerschool.com/)
      - `vcsnet/` - [vcs.net](https://vcs.net)
   - `configutil/` - additional utilities for reading and resolving configuration.
   - `gradestore/` - a simple time-series store for grade data.
   - `htmlutil/` - additional utilities for working with HTML.
   - `oauth/` - shared utils for working with oauth.
   - `restyutil/` - utilities for the `resty` HTTP client wrapper.
   - `serviceutil/` - additional utilities that are commonly used in service entrypoints.
   - `sqliteutil/` - utilities for opening up and migrating sqlite databases
   - `telemetry/` - telemetry setup/teardown as well as misc. instrumentation utilities
   - `textutil/` - utilities for cleaning and processing text.
   - `timezone/` - `time.Now()` always in the correct timezone, instead of system time. (because sometimes servers are hosted outside of PDT)
- `dev/` - go scripts for setting up the development environment
   - `local_stack/` - docker compose stuff for setting up grafana and other things locally
- `buf.yaml` - [buf.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- `buf.gen-go.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for golang code
- `buf.gen-ts.yaml` - [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml) for typescript code
- `sqlc.yaml` - [sqlc.yaml](https://docs.sqlc.dev/en/latest/reference/config.html)
- `telemetry.json5` - configuration of telemetry for development

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

## Setting up a database schema + queries

We use `sqlc` as a code generation tool to generate the "data layer" (the part that's only concerned with interfacing with the database which provides a high-level api to insert/delete/create for the rest of the code) of the backend.

As such, there's a specific way you must setup your database schema/operations so that sqlc can generate the code for it properly, that is as follows.

1. Create a `db` directory (usually in the directory of the service/package using it).
2. Create a file called `schema.sql` and a file called `query.sql` inside the `db` directory.
3. The `schema.sql` file will contain all your `create table ...` statements forming your database schema while the `query.sql` file will contain all the operations that need to have wrappers code-generated for them.
4. Then, in the `sqlc.yaml` directory, add another entry alongside the existing entries where `engine: "sqlite"` with the `queries` and `schema` field changed to the correct paths.
5. When you have added at least 1 table to `schema.sql` and 1 operation to `query.sql`, you can run `sqlc generate` and obtain your generated wrapper code.

## Creating a new service

Creating a new service at a high level, involves 3 steps.

1. Defining gRPC shape.
   1. Creating a protobuf definition of the service under `/proto/vcassist/services/<service_name>/v1/<...>.proto`.
   2. Running `buf generate --template buf.gen-go.yaml`.
   3. Running `buf generate --template buf.gen-ts.yaml` (you don't need to do this if the service is internal, that is, if the frontend will not directly call your service).
2. Setting up service logic.
   1. Creating a directory for your service logic under `/services/<service_name>`.
   2. Adding a `/services/<service_name>/db` subdirectory to hold your `schema.sql` and `query.sql` (don't do this if you do not need a DB).

