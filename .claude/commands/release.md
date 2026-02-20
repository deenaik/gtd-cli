Release a new version of gtd-cli.

Ask the user for the version number (e.g. v0.2.0) if not provided as $ARGUMENTS.

Steps:
1. Run `make test` to ensure tests pass before releasing.
2. Run `go vet ./...` for lint checks.
3. Ensure the working tree is clean (`git status`). If there are uncommitted changes, ask the user whether to commit them first.
4. Create a git tag for the version: `git tag <version>`
5. Push the tag: `git push origin <version>`
6. Run GoReleaser locally: `GITHUB_TOKEN=$(gh auth token) goreleaser release --clean`
7. Verify the release was created: `gh release view <version> --repo deenaik/gtd-cli`
8. Verify the Homebrew formula was updated: `gh api repos/deenaik/homebrew-tap/contents/gtd-cli.rb --jq '.name'`
9. Report the release URL and confirm users can install via `brew install deenaik/tap/gtd-cli`.
