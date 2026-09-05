# envdrift

**Catch `.env` drift before your deployment does.**

`envdrift` is a small, dependency-free Go CLI that treats `.env.example` as a contract and checks every environment file against it. It spots missing variables, unexpected variables, duplicate declarations, and—when asked—different values without printing secrets.

> Your staging deploy should not be the first test of your configuration.

## Why it exists

Most teams have more than one dotenv file: `.env.example`, `.env.local`, CI variables, staging, production. They drift quietly. A feature adds `STRIPE_WEBHOOK_SECRET`, production does not get it, and the error arrives much later and in the least convenient place.

`envdrift` makes that contract executable in a pre-commit hook or CI job.

## Install

```sh
go install github.com/qvoo/Hello-World/cmd/envdrift@latest
```

Or run it without installing:

```sh
go run github.com/qvoo/Hello-World/cmd/envdrift@latest .env.example .env.local
```

## Quick start

```dotenv
# .env.example
PORT=8080
DATABASE_URL=
STRIPE_WEBHOOK_SECRET=
```

```dotenv
# .env.staging
PORT=8080
DATABASE_URL=postgres://…
DEBUG=true
```

```sh
envdrift .env.example .env.staging
```

```text
WARNING: extra DEBUG in .env.staging — not declared in .env.example
ERROR: missing STRIPE_WEBHOOK_SECRET in .env.staging — required by .env.example

Summary: 1 error(s), 1 warning(s)
```

No paths uses the familiar `.env.example` → `.env` default.

## What it checks

| Finding | Default level | Meaning |
| --- | --- | --- |
| `missing` | error | A contract key is absent from a target file. |
| `extra` | warning | A target key has not been documented in the contract. |
| `value` | warning | A key value differs; available with `--values`. |
| duplicate key | input error | A dotenv file defines the same key twice. |

Values are **never emitted**. `--values` uses a short SHA-256 fingerprint, which makes comparison possible without copying credentials into CI logs.

## Useful commands

```sh
# Check several targets against one contract.
envdrift .env.example .env.local .env.staging

# Also detect differing values (safe fingerprints only).
envdrift --values .env.example .env.ci

# Treat undocumented variables as build failures.
envdrift --fail-on warning .env.example .env.production

# Ignore values owned outside this repository.
envdrift --ignore SENTRY_DSN --ignore CI_JOB_TOKEN .env.example .env.ci

# Send structured data to another CI step.
envdrift --format json .env.example .env.staging
```

## GitHub Actions

Add this job to your workflow:

```yaml
env-contract:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: stable
    - run: go run ./cmd/envdrift .env.example .env.ci.example
```

Do not commit real secrets just to satisfy the tool. Check safely committed contract files, or materialize short-lived CI files from your secret store first.

## Supported dotenv syntax

- Blank lines and full-line comments
- `KEY=value` assignments
- `export KEY=value` assignments
- Single- and double-quoted single-line values
- Inline comments after whitespace (`KEY=value # note`)

It intentionally rejects multiline shell expressions: deterministic configuration checks are more useful in CI than trying to interpret a shell.

## Development

```sh
go test ./...
go vet ./...
go build ./cmd/envdrift
```

The project uses only the Go standard library. Issues and small focused pull requests are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md).

## Roadmap

- [ ] GitHub Action wrapper with PR annotations
- [ ] Optional `.envdrift.yml` policy file
- [ ] Shell completion scripts

## License

MIT. See [LICENSE](LICENSE).

