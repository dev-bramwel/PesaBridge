# PesaBrigde

A Kenyan payment library written in Go.

## Continuous integration

GitHub Actions runs on every push and pull request, and can also be run manually.
It uses the latest stable Go release to check `gofmt` formatting and run
`go build ./...`. Formatting failures list the files that need formatting.

While the repository has no Go source or module, the checks report that they are
skipped. Once Go files are added, a root `go.mod` is required for builds.

Before pushing Go code, run:

```sh
make fmt
make check
```

Available commands:

- `make fmt`: format Go source files.
- `make fmt-check`: check formatting without editing files.
- `make build`: build all Go packages.
- `make check`: run both CI checks locally.
- `make`: show available commands.

Requires Go, GNU Make, Bash, and Git. CI uses the same Makefile targets.
