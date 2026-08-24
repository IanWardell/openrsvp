# Git and Release Workflow

This file is the operational checklist for publishing OpenRSVP changes. Follow
it instead of improvising a release from recent conversation context.

## Repository model

- `main` is the protected, tested development branch.
- Normal changes reach `main` through a pull request with required CI passing.
- `release` is a protected promotion branch. Advance it only by fast-forwarding
  it to a commit already present on `main` with successful required checks.
- Pushing `release` publishes the mutable `candidate` container tag and an
  immutable `sha-*` container tag. It does **not** create a stable release.
- Pushing an annotated semantic-version tag publishes the stable or prerelease
  containers and creates the GitHub Release.
- Stable release tags and semantic container tags are immutable. Never reuse or
  overwrite a released version.

## Before any GitHub write

Confirm the repository, branch, scope, and authentication:

```bash
git status -sb
git remote -v
git branch --show-current
gh auth status
git fetch origin --prune --tags
```

Do not stage unrelated files. Use explicit paths when the worktree is mixed.
Do not push directly to `main` as a shortcut around the pull-request workflow.
The PR author cannot approve their own PR; approval must come from another
GitHub account when branch protection requires it.

## Normal change workflow

1. Start from current `main`:

   ```bash
   git switch main
   git pull --ff-only origin main
   git switch -c agent/<short-description>
   ```

2. Implement and test the change.
3. Review `git diff`, then stage only intended files.
4. Commit with a terse description and push the feature branch:

   ```bash
   git push -u origin "$(git branch --show-current)"
   ```

5. Open a draft PR against `main`, using `gh pr create --draft` if the GitHub
   app lacks write permission.
6. Wait for required PR checks and review. Merge only when the PR is ready.
7. After merge, synchronize locally and remove the merged local branch:

   ```bash
   git switch main
   git pull --ff-only origin main
   git branch -d agent/<short-description>
   ```

Merging a normal PR does not automatically make a stable release.

## Decide between candidate and stable release

Do not infer that "merge," "push," or "next steps" authorizes an immutable
stable release. Unless the user explicitly specifies the destination, ask them
to choose:

- **Candidate:** promote tested `main` to `release`; publishes `candidate` and
  `sha-*` images only.
- **Stable/prerelease:** prepare a new semantic version on `main`, promote it to
  `release`, then create and push the matching annotated tag.

For a backward-compatible bug fix, propose the next patch version. Never choose
a major or minor bump without confirming the intended release scope.

## Prepare a stable version

Before promotion, the exact release version must already be committed on
`main`. Prepare it through a normal PR:

1. Choose a new SemVer value such as `1.10.1`; the Git tag will be `v1.10.1`.
2. Update `VERSION` to the version without the `v` prefix.
3. Add `.github/releases/vX.Y.Z.md` with user-facing release notes.
4. Update the README changelog when appropriate.
5. Run the relevant full validation suite.
6. Commit, push, open a PR, obtain required review, and merge it.
7. Verify required CI on the resulting `main` commit has passed:

   ```bash
   gh run list --repo IanWardell/openrsvp --branch main --workflow CI --limit 5
   ```

The release workflow will reject a tag whose version does not exactly match
`VERSION`.

## Promote `main` to the candidate channel

Only promote a tested `main` commit, and only with a fast-forward:

```bash
git fetch origin --prune
git switch release
git pull --ff-only origin release
git merge --ff-only origin/main
git push origin release
```

The push starts the `Release` workflow's candidate job. Wait for it to finish
before tagging:

```bash
gh run list --repo IanWardell/openrsvp --branch release --workflow Release --limit 5
gh run watch <run-id> --repo IanWardell/openrsvp --exit-status
```

Confirm that the promoted `release` commit is the intended `main` commit. A
candidate failure must be fixed on `main` through a PR and promoted again; do
not patch the `release` branch directly.

## Publish a stable or prerelease version

After the candidate workflow succeeds, tag the promoted release commit. First
verify the version, release notes, and target commit:

```bash
git switch release
git pull --ff-only origin release
git rev-parse HEAD
git rev-parse origin/main
cat VERSION
test -f .github/releases/vX.Y.Z.md
git tag -l vX.Y.Z
gh release view vX.Y.Z --repo IanWardell/openrsvp
```

The final two lookup commands should show that the new tag and GitHub Release do
not already exist. Then create and push an **annotated** tag:

```bash
git tag -a vX.Y.Z -m "OpenRSVP vX.Y.Z"
git push origin vX.Y.Z
```

Never use a lightweight tag. Never move, delete, or recreate a published tag to
repair a release. If a published version is defective, fix it on `main` and
issue the next patch version.

Versions containing a SemVer prerelease suffix, such as `1.11.0-rc.1`, do not
update `latest`. Versions without a prerelease suffix do update `latest`.

## Verify publishing

The tag-triggered workflow must complete successfully before reporting the
release as done:

```bash
gh run list --repo IanWardell/openrsvp --workflow Release --limit 5
gh run watch <run-id> --repo IanWardell/openrsvp --exit-status
gh release view vX.Y.Z --repo IanWardell/openrsvp
docker buildx imagetools inspect ghcr.io/ianwardell/openrsvp:X.Y.Z
```

A successful stable release publishes:

- `X.Y.Z`
- `X.Y`
- `X`
- `latest` for stable versions only
- an immutable `sha-*` tag
- provenance and SBOM metadata
- a GitHub Release using `.github/releases/vX.Y.Z.md` when present

Finish by returning the local checkout to synchronized `main`:

```bash
git switch main
git pull --ff-only origin main
git status -sb
```

Report the version, tag, release URL, release workflow result, container tags,
and final local branch state.

## Release safety rules

- Never release from an unmerged feature branch.
- Never advance `release` with a merge commit or force push.
- Never tag before `VERSION` and release notes are on both `main` and `release`.
- Never tag a commit that has not passed required CI.
- Never overwrite an existing GitHub Release or semantic container tag.
- Never claim success until the remote release workflow finishes successfully.
- If authorization for candidate versus stable release is ambiguous, stop and
  ask before pushing `release` or a version tag.
