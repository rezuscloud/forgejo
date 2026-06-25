# Staging Repositories

This directory follows the same convention as Kubernetes
([`k8s.io/kubernetes/staging`](https://github.com/kubernetes/kubernetes/tree/master/staging)):
it holds Go modules that are developed **in-tree** alongside the Forgejo server
but published/consumed as **independent** modules. The server module
(`forgejo.org`, at the repo root) references them through `replace` directives
so a single checkout builds everything, while external consumers can import them
as standalone modules.

## Why staging

Putting the API client and the CLI in the server repo (rather than separate
repos) keeps them versioned with the exact Forgejo release they target, exactly
like `client-go` and `kubectl` track Kubernetes releases. Keeping them as their
own modules means the server binary (`cmd/gitea`, the Forgejo server) never
compiles them in — they are separate import trees.

## Layout

```
staging/src/forgejo.org/
├── client-go/   module forgejo.org/client-go   — Go API client (analogue of k8s.io/client-go)
└── fj/          module forgejo.org/fj          — fj CLI command library (analogue of k8s.io/kubectl)
```

The corresponding **binary** lives in the root module, mirroring `cmd/kubectl`:

```
cmd/fj/fj.go   package main, module forgejo.org   — the fj binary
```

`cmd/fj` imports `forgejo.org/fj/pkg/cmd` (the command tree) and
`forgejo.org/client-go` (the API client).

## Wiring (root go.mod)

The root module declares both `require` and `replace` for each staging module,
exactly like Kubernetes:

```go
require (
	forgejo.org/client-go v0.0.0
	forgejo.org/fj v0.0.0
)

replace (
	forgejo.org/client-go => ./staging/src/forgejo.org/client-go
	forgejo.org/fj => ./staging/src/forgejo.org/fj
)
```

Staging modules that depend on each other use `v0.0.0` plus a relative replace
within staging (e.g. `fj` requires `forgejo.org/client-go v0.0.0` and
`replace forgejo.org/client-go => ../client-go`).

## Rules (mirroring Kubernetes staging rules)

1. **No new dependencies leak into the server.** The staging modules are
   independent import trees; the server never imports them. Their third-party
   dependencies live in their own `go.mod`, not in the root module's build of
   the server (only `cmd/fj` pulls them in at build time via the `replace`).
2. **Import paths must match the module path.** Code under
   `staging/src/forgejo.org/client-go` imports as `forgejo.org/client-go`, never
   a path relative to the staging tree.
3. **The layout is machine-checked.** `cmd/fj` building from the root module
   proves the `replace` graph is consistent. CI builds all three artifacts:
   the server image, the `fj` binary, and the Helm chart.
4. **Re-basing upstream is clean.** Upstream Forgejo has no `staging/`,
   `cmd/fj`, `charts/`, or `distributions/` — these directories never conflict
   with a `git merge` from `codeberg.org/forgejo/forgejo`.
