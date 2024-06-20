## vcassist-backend

> WIP rewrite of the vcassist backend in golang.

### project structure

- `cmd/` - all official entrypoints/build targets
- `lib/` - shared libraries
   - `platforms/` - bootleg APIs for various platforms
- `dev/` - stuff only relevant to development
   - `docs/` - documentation
   - `local-stack/` - docker compose stuff for setting up grafana and other things locally
   - `scripts/` - golang scripts
- [buf.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- [buf.gen.yaml](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- [sqlc.yaml](https://docs.sqlc.dev/en/latest/reference/config.html)
- `telemetry.json5` - configuration of telemetry for development

### common commands

- `sqlc generate` - generate sql wrapper code
- `buf generate` - generate protobuf files
- `buf lint` - lint protobuf files
- `cd local-stack && docker compose up -d` - starts up a local grafana instance at `http://localhost:3000`
- `go run dev/scripts/main.go dev:apply_db_schema` - apply db schema for local dev databases

### dependencies

- `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `go install github.com/bufbuild/buf/cmd/buf@v1.33.0`
- [altas](https://atlasgo.io/getting-started#installation)

### may add?

- [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck)

