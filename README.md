# install-update-workflow-test

A small Go TUI (built with [Bubble Tea](https://github.com/charmbracelet/bubbletea))
used to work through three patterns that come up in almost every self-contained
CLI tool:

1. A one-command install script that fetches the right binary and puts it on `PATH`.
2. Checking for and applying updates from *inside* the running app.
3. Publishing releases in a shape both of the above can consume.

## Install

macOS/Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/xSaVageAU/install-update-workflow-test/main/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr https://raw.githubusercontent.com/xSaVageAU/install-update-workflow-test/main/scripts/install.ps1 -useb | iex
```

Both scripts: detect OS/arch, fetch the latest GitHub release, download the
matching binary, verify its SHA-256 checksum against `checksums.txt`, install
it to a user-owned directory (`~/.local/bin`, or `%LOCALAPPDATA%\Programs\iuw`
on Windows), and add that directory to `PATH` if it isn't already there.

## Run

```sh
iuw
```

`↑`/`↓` to move, `enter` to select, `q` to quit. The menu has "Check for
updates" (queries the GitHub Releases API and offers to install if a newer
version is published) and "About" (shows the running version/commit/build date).

## How the pieces fit together

- **`internal/version`** — `Version`/`Commit`/`Date` are `dev`/`none`/`unknown`
  by default and get overwritten at build time via `-ldflags`, wired up in
  `.goreleaser.yaml`.
- **`internal/update`** — talks to the GitHub Releases API (`LatestRelease`),
  compares semver against the running version (`IsNewer`), and applies an
  update in place (`Apply`). Releases are published as raw binaries (no
  tar/zip), named `iuw_<os>_<arch>` (`.exe` on Windows) plus a `checksums.txt`,
  so both the install scripts and the in-app updater can fetch and use them
  directly.
- **`internal/tui`** — the Bubble Tea model that drives the menu and the
  check/confirm/apply update flow.
- **`scripts/install.sh` / `scripts/install.ps1`** — the one-command installers
  described above.
- **`.goreleaser.yaml`** + **`.github/workflows/release.yml`** — pushing a tag
  like `v0.1.0` builds binaries for linux/darwin/windows × amd64/arm64, injects
  version info, and publishes them as a GitHub Release with a `checksums.txt`.

### The tricky part: replacing a running binary

You can't overwrite or delete a file that's currently mapped into a running
process on Windows. The workaround (also used by tools like rustup and scoop)
is a rename dance: rename the running binary aside (`iuw` -> `iuw.old`), move
the newly downloaded binary into its place, then best-effort delete the old
one — Windows allows renaming an in-use file even though it disallows
deleting or overwriting it directly. See `internal/update/apply.go`.

## Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow picks up the tag and runs
[GoReleaser](https://goreleaser.com) to build and publish everything.

To test the release pipeline locally without publishing:

```sh
goreleaser release --snapshot --clean
```
