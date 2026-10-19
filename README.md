# voluspa

The graph library of Yggdrasil: one folder per graph, with its stage prompts.

Part of [Yggdrasil](https://github.com/ouroboric-dev/yggdrasil), the agentic factory by
[Ouroboric](https://github.com/ouroboric-dev/ouroboric). Work on it from the Yggdrasil umbrella,
which holds every component as a submodule and builds them together through `go.work`.

## Build and test

```bash
go test ./...
go vet ./...
```

The required check `test` runs `gofmt`, `go vet`, `go test -race` as a non-root user, and `govulncheck`.

## Docs

- Architecture: [docs/architecture.md](docs/architecture.md)
- Engineering spec: <https://github.com/ouroboric-dev/yggdrasil/blob/main/docs/2-engineering/specs/2026-10-11-yggdrasil-v0-spec.md>

## Contributing

Run `uvx pre-commit install` once after cloning, so `gitleaks` checks each commit for secrets.

## Licence

Code is AGPL-3.0-only, except the graphs (`software/`), which are MIT. `docs/`, this README and `AGENTS.md` are CC-BY-4.0. See `REUSE.toml`.
