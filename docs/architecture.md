---
title: Voluspa architecture
status: draft
date: 2026-10-19
related: [https://github.com/ouroboric-dev/yggdrasil/blob/main/docs/2-engineering/specs/2026-10-11-yggdrasil-v0-spec.md]
style: reference
---

# Voluspa architecture

The graph library: one folder per graph, with its stage prompts.

## Responsibilities

- Holds the graphs Odin runs, one folder per graph, as graph packs. A graph pack holds a `pack.yaml`, a
  `graph.yaml` and the `prompts/` the graph names, and no recipe (no checks, leaf, sandbox or services).
- Holds `software`, 0.1.0: the graph of a software product (spec §4.2) and the prompts of its planning stages and
  its three reviewers. It moved here from `midgard-react` with its prompts byte for byte.
- A run names its graph with `--graph`; a project pins it in `umbrella.yaml`. A leaf's implementer prompt,
  template, stack and checks stay in the leaf's realm.
- Its files are MIT-licensed (`software/`).

## Depends on

- Gungnir's schemas: the pack and the graph (`graph.yaml` holds `schema: 2`).
- Havamal's `ponytail-review` skill, for the reuse review and the simplicity reviewer.
- The operator's `factory.yaml` defines the roles `planner`, `implementer` and `reviewer` the graph names.

## Tests

`go test ./...` checks each graph pack, its graph and its prompts against Gungnir's schemas.

## CI

`.github/workflows/ci.yml` has one job, `test`, the required check on `main`. It runs in the Go image of
the devcontainer, without `go.work`, and runs `gofmt`, `go vet`, `go test -race` as a non-root user, and
`govulncheck`. The pr-bot cannot change the workflow; the founder pushes changes to it.

## Source

The v0 engineering spec. When this page and the spec differ, the spec wins.
