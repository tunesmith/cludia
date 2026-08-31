# Releasing Cludia

Cludia uses semantic versions, annotated Git tags, source-only GitHub releases,
and a source-building formula in
[`tunesmith/homebrew-tap`](https://github.com/tunesmith/homebrew-tap).

The application repository and Homebrew tap are released separately. Do not
publish a tag until the application pull request is merged and all checks pass.
The examples below use `v1.0.1`; substitute the intended version consistently.

## 1. Prepare the release candidate

Start from an up-to-date `main` and create a focused release branch:

```sh
release_version=v1.0.1
release_branch=codex/release-v1.0.1
git switch main
git pull --ff-only origin main
git switch -c "$release_branch"
```

Before committing, update the checked-in program version, dated changelog,
README installation examples, and user-visible documentation. The checked-in
and linker-injected builds must both report `cludia $release_version`.

Commit the complete candidate, then run the non-publishing preflight from the
clean commit:

```sh
scripts/release-check "$release_version"
```

The check verifies the clean tree, changelog date, local and remote tag
availability, GPL/SPDX declarations, current documentation, tests, race tests,
vet, builds, version output, `.arg` compatibility fixtures, profile migration,
truth evaluation, cycles, atomic failures, rooted export, and JSON/process
contracts. It does not modify Git, GitHub, Homebrew, or the working tree.

## 2. Review and merge

Push the release branch, open a ready pull request, and wait for CI:

```sh
git push -u origin "$release_branch"
gh pr create --base main --head "$release_branch" \
  --title "Release Cludia $release_version" --fill
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

## 3. Tag and create the source release

Create the annotated tag at verified `main` and push only that tag:

```sh
git tag -a "$release_version" -m "cludia $release_version"
git push origin "$release_version"
```

The tag-triggered workflow repeats tests, race tests, vet, command builds, and
version agreement. It extracts the matching changelog section and creates a
source-only GitHub release; no binary attachments are uploaded.

```sh
gh run list --workflow release.yml --limit 5
gh run watch --exit-status
gh release view "$release_version"
test "$(git rev-list -n 1 "$release_version")" = "$(git rev-parse main)"
```

## 4. Publish the Homebrew formula

Download the public tagged archive and calculate its checksum from that exact
artifact:

```sh
curl -fL \
  "https://github.com/tunesmith/cludia/archive/refs/tags/$release_version.tar.gz" \
  -o "/tmp/cludia-$release_version.tar.gz"
shasum -a 256 "/tmp/cludia-$release_version.tar.gz"
```

In `tunesmith/homebrew-tap`, update `Formula/cludia.rb` with that URL and
checksum. Keep license `GPL-3.0-or-later`, the build dependency on Go,
`std_go_args` for `./cmd/cludia` with `main.version=v#{version}`, and formula
tests for version output, workspace creation, JSON validation, derivation, and
grounded evaluation.

Run `brew style`, commit, and push the tap. Refresh the published tap before
the online audit:

```sh
brew style Formula/cludia.rb
git add Formula/cludia.rb
git commit -m "Update Cludia formula to ${release_version#v}"
git push origin main
brew update
brew audit --strict --online tunesmith/tap/cludia
```

## 5. Verify the installed formula

Upgrade and test through Homebrew:

```sh
brew upgrade tunesmith/tap/cludia
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
binary output. Every release identifier must match `release_version`.
