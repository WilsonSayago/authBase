# Release checklist (authBase v4.x)

Do **not** create tags, GitHub releases, or push from automation until every
item below is checked by a human maintainer.

## Repository identity

- [x] `git remote` points at `https://github.com/WilsonSayago/authBase.git`
      (not `middleware.git`).
- [x] Module path is `github.com/WilsonSayago/authBase/v4`.
- [ ] Working tree is clean on the release commit.

## Legal and security

- [x] LICENSE file added with the maintainer-chosen license text (MIT).
- [x] `docs/SECURITY.md` lists a confirmed private reporting channel (GitHub
      Security Advisories / private vulnerability reporting).
- [ ] In GitHub → Settings → Code security → enable **Private vulnerability
      reporting** so reporters can use the Security tab button.
- [ ] SECURITY / README do not claim a published tag prematurely.

## Quality gates

- [ ] CI green on Go 1.26.8 and 1.27.1.
- [ ] `go test ./...`
- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `govulncheck@v1.8.0 ./...` → no reachable vulnerabilities.
- [ ] `go build ./examples/...` and `go test ./examples/...` pass.
- [ ] External consumer module with
      `replace github.com/WilsonSayago/authBase/v4 => <local>` builds.
- [ ] `git diff --check` clean.
- [ ] Manual review of `docs/MIGRATION_V4.md` and retained v3 guidance.

## Publish (manual only)

- [ ] The intended `v4.x.y` version and changelog entry are finalized.
- [ ] Annotated `v4.x.y` tag created locally after checklist completion.
- [ ] Tag pushed explicitly; never move or reuse the tag.
- [ ] CHANGELOG date filled for the release.
- [ ] Proxy/`go list -m github.com/WilsonSayago/authBase/v4@v4.x.y` verified.
