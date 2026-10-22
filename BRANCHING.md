# Branching Model

Branch structure for form.md development. The project has one owner, so the process is light: one protected long-lived branch, short-lived work branches, every change lands on `main` through a pull request that CI has built, and every release is a `compile_vX.Y.Z` branch cut from `main` that formcompiler builds.

## The branches

```
main ───●────●────●────●────●───────●──────── protected, always builds, releases cut as compile_vX.Y.Z
         \       /      \          /
          feature/[area]-[slug]    fix/[area]-[slug]      work branches, short-lived,
                                                          deleted after merge
```

`main` is the only long-lived branch. There is no `dev` or release branch.

| Branch | Role | Rules |
|---|---|---|
| `main` | The code, always buildable; every release is cut from it | Protected. Receives changes only through pull requests. CI (`.github/workflows/ci.yml`, job `build`) runs on every push to it and to `compile_v*` branches. |
| `feature/[area]-[slug]` | New capability | Branch from `main`, pull request back to `main`. |
| `fix/[area]-[slug]` | Bug fix | Branch from `main`, pull request back to `main`. |
| `docs/[slug]` | README, wiki (`.docs/wiki/`), HASHOUT logs | Branch from `main`, pull request back to `main`. |
| `chore/[slug]` | Tooling, scripts, dependencies, CI, releases | Branch from `main`, pull request back to `main`. |
| `experiment/[slug]` | Spikes and throwaway prototypes | Branch from `main`. Never merged directly; delete when concluded. |

## Area prefixes

Use the area the change mostly lives in, the same word as the commit scope:

`server` · `web` · `doc` · `editor` · `database` · `collection` · `auth` · `api` · `mcp` · `ops` · `offline` · `files` · `transfer` · `ai` · `automation` · `store` · `docs`

Examples: `feature/offline-start`, `fix/web-dev-proxy-origin`, `docs/wiki-operations`.

## Workflow

1. `git switch main && git pull`, then `git switch -c feature/[area]-[slug]`.
2. Commit in small steps. Every subject reads `type(scope): summary (x.y.z)`, where `x.y.z` is the version being built toward (see the wiki page Development). No emojis in commit messages.
3. Before pushing, run `make check` and `make build`.
4. Push and open a pull request into `main`. CI must pass. With a team of one, self-merge is expected; say in the pull request what was verified and how.
5. Merge, then delete the branch.

## Releases and tagging

A release is one version bump commit on `main` plus a `compile_vX.Y.Z` branch cut from main's head. This repository builds nothing: formcompiler (`nodemode-ai/formcompiler`) builds and drafts every form* release.

1. On a `chore/release-vX.Y.Z` branch, bump the version everywhere it is written:
   - `VERSION := vX.Y.Z` in `Makefile`
   - `var Version = "vX.Y.Z"` in `internal/app/version.go`
   - `"version"` in `web/package.json` and `web/package-lock.json`
   - the version badge in `README.md`
2. Commit it as `chore(release): vX.Y.Z (x.y.z)` and merge it into `main`.
3. Push a compile branch from main's head:

   ```sh
   git switch main && git pull
   git push origin main:refs/heads/compile_vX.Y.Z
   ```

4. CI runs on the branch. formcompiler picks it up on its next 8-hourly watch run, or at once when its watch is run by hand. It checks that the branch's version equals `VERSION` in the `Makefile`, runs `make check` and `make release`, builds the desktop app with `./build_app --yes` (`build_app.bat` on Windows), and drafts the release `form.md-vX.Y.Z` on its own release page with the five binaries, `SHA256SUMS`, `install.sh`, `form-md.service` and the desktop bundles. It also builds the container image for `linux/amd64` and `linux/arm64` (`ghcr.io/nodemode-ai/form.md:vX.Y.Z` and `:latest`) and keeps `form.md-updates/latest.json` current for the update check. The steps the old workflows ran are quoted in `src-tauri/HASHOUT_LOG.md`.
5. Check the draft, publish it, and run the installer against it (`sh install.sh --version=vX.Y.Z`) on a scratch machine.

## Versioning

Semver: patch for fixes, minor for new features or a finished phase, major for breaking changes to the data directory, the API or the command line. Tags read `v` then three dot-separated integers, for example `v0.11.0`, always with dots and never with underscores or hyphens inside the version. Compile branches read `compile_v` then the version, for example `compile_v0.46.1`. The `vX.Y.Z` tags up to v0.46.0 live on `main` and are never moved or reused; later release tags are formcompiler's `form.md-vX.Y.Z`.

## GitHub protection settings

One-time setup in the GitHub UI under Settings:

- Default branch: `main`.
- Branch rule for `main`: require a pull request before merging, require the status check `build` from the `ci` workflow, block force pushes, block deletions. With one owner, require 0 approvals; raise it to 1 when a second maintainer joins.
- Tag rule for `v*`: restrict creation to maintainers and block updates and deletions, so a published tag never changes under the release.

## Keeping branches healthy

- Rebase or merge `main` into your work branch when it drifts; resolve conflicts on the branch, never on `main`.
- Branches live days, not weeks. If a feature is too big, split it and land it behind inert seams.
- `experiment/*` never merges; promote a surviving idea by cherry-picking onto a fresh `feature/*` branch.
