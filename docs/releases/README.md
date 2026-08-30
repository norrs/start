# Release notes

Releases are started manually from the GitHub Actions **Release** workflow on
`main`. Choose `auto` to infer the next stable semantic version from commits,
or explicitly choose `patch`, `minor`, or `major` when the history is
ambiguous.

Automatic selection follows Conventional Commits:

- `fix:`, documentation, chores, and other changes produce a patch release.
- `feat:` produces a minor release.
- A `!` before `:` or a `BREAKING CHANGE:` footer produces a major release.

To curate the notes for a release, add either `docs/releases/v1.2.3.md` or
`docs/releases/1.2.3.md` before running the workflow. The `v` form takes
precedence. If neither file exists, GitHub generates notes from commits and
merged pull requests.

The workflow runs the complete Bazel test suite, builds and publishes the
amd64/arm64 image, creates the GitHub release and tag, and then commits the
generated latest-release block in `README.md` back to `main`.
