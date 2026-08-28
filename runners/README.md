# Runner fleet

Forgejo Actions runners for `git.rezus.cloud`. The **server needs no changes for
non-linux runners** — labels are free-form and the `host` label schema (jobs run
directly on the machine) is even the protocol default. What upstream *doesn't*
ship is darwin runner **binaries**: `code.forgejo.org/forgejo/runner`'s
build-release.yml passes `platforms: linux/amd64,linux/arm64` to the shared
build action, so no macOS assets exist at any upstream version.

## Platform matrix

| Platform | Deployment | Labels (schema) | Owner |
|---|---|---|---|
| linux (k8s) | `k8s-iac/modules/codeberg-runner` — wrenix chart + DinD | `<name>:docker://<image>` | k8s-iac repo |
| **macOS (host)** | **`runners/macos/`** — this dir, user LaunchAgent | `<os>-<arch>:host` | this monorepo |

## Binary source

[`rezuscloud/forgejo-runner`](https://github.com/rezuscloud/forgejo-runner) —
fork of upstream adding darwin-amd64/arm64 tarballs + checksums on fork tags
`v*-rezus.*` (Gitea publishes darwin in every runner release; this restores
that coverage for the Forgejo runner). Pin the version in `install.sh` to the
same runner version as the k8s fleet (protocol parity).

## Labels contract

- Format `name[:schema[:args]]`; schema `docker` (container from image) or
  `host` (steps run directly on the machine), **`host` is the default**.
- Host runners use distinct names — `macos-arm64:host`, never `ubuntu-latest` —
  so GitHub-flavored workflows cannot accidentally run unsandboxed on a laptop.
- Jobs land via `runs-on: macos-arm64` matching by name.

## Security model

Host mode executes workflow code **with the privileges of the LaunchAgent
user**, no isolation. Treat any repo whose workflow can target the label as
trusted on that machine. Future isolation tiers if untrusted jobs ever need
macOS:

- **Tart / Cirrus-style** — ephemeral macOS VMs on Apple Silicon
  (Virtualization.framework), per-job throwaway instances.
- **Colima/Lima** — Linux containers on the Mac via a Lima VM; lets the same
  host additionally serve `docker://` labels (Woodpecker community pattern).

## macOS host layout (per `runners/macos/`)

| Path | Contents |
|---|---|
| `~/.local/bin/forgejo-runner` | binary (version-pinned, sha256-verified) |
| `~/.local/share/forgejo-runner/` | `config.yml`, `.runner` (registration), cache, logs |
| `~/Library/LaunchAgents/com.rezuscloud.forgejo-runner.plist` | user agent: RunAtLoad + KeepAlive |

Registration uses a one-shot instance-level registration token
(`fj api admin actions get-registration-token`); the resulting `.runner` file
holds the per-runner secret. The registration token is never stored.

## References

- Gitea labels & host mode: https://docs.gitea.com/runner/labels/
- Gitea runner releases (darwin assets): https://gitea.com/gitea/runner/releases
- GitHub self-hosted macOS runners: https://github.com/actions/runner/releases
  (osx-x64/osx-arm64 binaries; host execution; launchd service)
- Upstream linux-only gap: `build-release.yml` →
  `forgejo-build-publish/build` action, `platforms: linux/amd64,linux/arm64`
