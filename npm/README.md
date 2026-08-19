# curbpack (npm wrapper)

Thin Node wrapper around the [Curbpack](https://github.com/afelin/curbpack) release binary.
No Go required. **Lazy download on first exec** — no network `postinstall`.

Prepares evidence for human review — **not** a conformity assessment, CE mark, or certification.

## One-liner (read-only scan)

```bash
npx curbpack@0.5.2 scan
```

Run inside any git repository. Read-only — no files written, no hooks, no init.

## Install globally

```bash
npm install -g curbpack@0.5.2
curbpack scan
```

## Version pin

Default download is **`v0.5.2`** (from bundled `install-manifest.json`), not floating `latest`.

```bash
CURBPACK_VERSION=latest npx curbpack scan   # explicit opt-in to newest release
CURBPACK_VERSION=v0.5.2 npx curbpack scan   # explicit pin
```

## Trust

- Downloads from [afelin/curbpack](https://github.com/afelin/curbpack) GitHub Releases
- Verifies `checksums.txt` (sha256) **fail-closed** — mismatch refuses exec
- Cached binary: `~/.curbpack/bin` (macOS/Linux) or `%LOCALAPPDATA%\curbpack` (Windows)

## Claim safety

Green gates prepare structural evidence for **human review**. They are **not** certification,
CE marking, or notified-body approval.

Full install ladders: [docs/getting-started/install.md](https://github.com/afelin/curbpack/blob/main/docs/getting-started/install.md)
