## vcassist-backend

> WIP rewrite of the vcassist backend in golang.

### common commands

- `cd local-stack && docker compose up -d` - starts up a local grafana instance at `http://localhost:3000`
- `sqlc generate` - generate sql wrapper code
- `buf generate` - generate protobuf files
- `buf lint` - lint protobuf files
- `go run dev/scripts/main.go dev:apply_db_schema` - apply db schema for local dev databases

### dependencies

- `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- `go install github.com/bufbuild/buf/cmd/buf@v1.33.0`
- [altas](https://atlasgo.io/getting-started#installation)

### may add?

- [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck)

