# Contributing to envdrift

Thanks for considering a contribution.

## Before opening a pull request

1. Search existing issues so the work is not duplicated.
2. Keep behavior unsurprising and dependency-free.
3. Add a test for every bug fix or parsing edge case.
4. Run `go test ./...` and `go vet ./...`.

## Design principles

- **Secrets never reach output.** Findings may contain a fingerprint, never a raw dotenv value.
- **CI output is stable.** Findings are sorted and JSON fields are explicit.
- **The contract is simple.** The first file is the baseline; every following file is a target.

For a larger change, open an issue first with an example input, expected result, and why the current behavior is insufficient.

