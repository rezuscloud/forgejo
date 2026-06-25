// This is a staging module, developed in-tree alongside the Forgejo server
// (module forgejo.org). It is the `fj` CLI command library, the analogue of
// k8s.io/kubectl. The binary entrypoint lives at the repo root in cmd/fj
// (the fj analogue of cmd/kubectl). See staging/README.md.

module forgejo.org/fj

go 1.26.0

require (
	forgejo.org/client-go v0.0.0
	github.com/fatih/color v1.19.0
	github.com/spf13/cobra v1.9.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/sys v0.42.0 // indirect
)

replace forgejo.org/client-go => ../client-go
