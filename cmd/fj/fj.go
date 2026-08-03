// SPDX-License-Identifier: MIT
//
// cmd/fj is the Forgejo CLI binary. It is the `fj` analogue of `cmd/kubectl`:
// it lives in the server (root) module and wires together the in-tree staging
// libraries — the CLI command tree (forgejo.org/fj) and the API client
// (forgejo.org/client-go) it is generated against. See staging/README.md.
package main

import (
	"fmt"
	"os"

	fj "forgejo.org/fj/pkg/cmd"
)

func main() {
	if err := fj.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
