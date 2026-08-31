# Releasing Cludia

Cludia uses semantic versions, annotated Git tags, source-only GitHub releases,
and a source-building formula in
[`tunesmith/homebrew-tap`](https://github.com/tunesmith/homebrew-tap).

The v1.0.0 publication begins while `tunesmith/cludia` is private. The release
candidate is reviewed and tagged privately; only after the tag, workflow, and
source release agree is the repository made public.

## 1. Prepare the private release candidate

Start from current private `main` and create `codex/release-v1.0.0`. Before the
release commit, update the checked-in program version, changelog, specification,
license, compatibility contract, and release tooling. The checked-in and
linker-injected versions must both report `cludia v1.0.0`.

Commit the complete candidate, then run the non-publishing preflight from the
clean commit:

```sh
scripts/release-check v1.0.0
```

The check verifies the clean tree, changelog date, local and remote tag
availability, GPL/SPDX declarations, current documentation, tests, race tests,
vet, builds, version output, `.arg` compatibility fixtures, profile migration,
truth evaluation, cycles, atomic failures, rooted export, and JSON/process
contracts. It does not modify Git, GitHub, Homebrew, or the working tree.

## 2. Review and merge privately

Push the release branch, open a ready pull request in the private repository,
and wait for CI:

```sh
git push -u origin codex/release-v1.0.0
gh pr create --base main --head codex/release-v1.0.0 \
  --title "Release Cludia v1.0.0" --fill
gh pr checks --watch
```

Squash-merge the ready pull request. Then fast-forward local `main` and confirm
that it is clean and equals `origin/main`:

```sh
gh pr merge --squash --delete-branch
git switch main
git pull --ff-only origin main
git status -sb
test "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)"
```

## 3. Tag and create the source release while private

Create the annotated tag at verified `main` and push only that tag:

```sh
git tag -a v1.0.0 -m "cludia v1.0.0"
git push origin v1.0.0
```

The tag-triggered workflow repeats tests, race tests, vet, command builds, and
version agreement. It extracts the matching changelog section and creates a
source-only GitHub release; no binary attachments are uploaded.

```sh
gh run list --workflow release.yml --limit 5
gh run watch
gh release view v1.0.0
test "$(git rev-list -n 1 v1.0.0)" = "$(git rev-parse main)"
```

## 4. Make the repository public

After the private tag and release are verified, change visibility and confirm
anonymous access:

```sh
gh repo edit tunesmith/cludia --visibility public \
  --accept-visibility-change-consequences
curl -fsSL https://api.github.com/repos/tunesmith/cludia >/dev/null
```

Verify the public main page, tag, release notes, and source archives before
publishing a package that points at them.

## 5. Publish the Homebrew formula

Download the public tagged archive and calculate its checksum from that exact
artifact:

```sh
curl -fL \
  https://github.com/tunesmith/cludia/archive/refs/tags/v1.0.0.tar.gz \
  -o /tmp/cludia-v1.0.0.tar.gz
shasum -a 256 /tmp/cludia-v1.0.0.tar.gz
```

In `tunesmith/homebrew-tap`, add `Formula/cludia.rb` with that URL and checksum,
license `GPL-3.0-or-later`, a build dependency on Go, and `std_go_args` for
`./cmd/cludia` with `main.version=v#{version}`. Its test must cover version
output, workspace creation, JSON validation, an authored derivation, and
grounded evaluation.

Update the tap README, run `brew style`, commit, and push the tap. Refresh the
published tap before the online audit:

```sh
brew style Formula/cludia.rb
git add Formula/cludia.rb README.md
git commit -m "Add Cludia 1.0.0 formula"
git push origin main
brew update
brew audit --strict --online tunesmith/tap/cludia
```

## 6. Verify the installed formula

Install and test through Homebrew:

```sh
brew install tunesmith/tap/cludia
brew test tunesmith/tap/cludia
```

An older Go-installed `cludia` may appear earlier in `PATH`, so invoke the
formula explicitly:

```sh
cludia_brew_bin="$(brew --prefix cludia)/bin/cludia"
"$cludia_brew_bin" --version
"$cludia_brew_bin" validate --json examples/broken-window-workspace.arg
"$cludia_brew_bin" evaluate --json examples/broken-window-workspace.arg
```

Finish by recording and comparing the public `main` commit, annotated tag,
GitHub release, archive SHA-256, tap commit, formula version, and installed
binary output. Every release identifier must be `v1.0.0`.
