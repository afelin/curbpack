# Install (source of truth)

One Curbpack product for **Windows, macOS, and Linux**. Same golden path after install:

```text
install → doctor → demo [--open] → init → check → share [--bundle] → (human) attest
# after OS update / PATH loss:
doctor → doctor --repair → full reinstall only if binary missing
```

Not conformity assessment. Not CE marking. Not a notified-body opinion.

**Pin:** `@v0.5.2` / install default from [`scripts/install-manifest.json`](../../scripts/install-manifest.json).  
**Action:** GitHub Action runners are **Linux/macOS only** in v0.5.2 — local Windows CLI is supported.  
**Sock:** optional Coreward Unix IPC — **Unix-only**; golden path never requires it.  
**Deferred:** winget, windows/arm64, pwsh completion.

Stuck? [Troubleshooting](troubleshooting.md).

---

## Three ladders × both OS families

### Ladder 0 — Read-only scan (no install)

Inside any git repository — **no init**, **no files written**:

```bash
npx curbpack@0.5.2 scan
```

Shows open gate signals against the CRA-shaped default pack (`cra-baseline`), Art 14 reporting clock, and product hint. Not conformity assessment. See also: [Art 14 reporting vs handling](art14-reporting-vs-handling.md) (campaign page ships separately).

### Ladder 1 — Download + run installer (recommended)

**macOS / Linux**

```bash
curl -fsSL -o /tmp/curbpack-install.sh https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.sh
sh /tmp/curbpack-install.sh
curbpack doctor
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.ps1 -OutFile $env:TEMP\curbpack-install.ps1
powershell -ExecutionPolicy Bypass -File $env:TEMP\curbpack-install.ps1
curbpack doctor
```

### Ladder 2 — One-liner pipe

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.sh | sh
```

**Windows (PowerShell)**

```powershell
irm https://raw.githubusercontent.com/afelin/curbpack/main/scripts/install.ps1 | iex
```

If ExecutionPolicy blocks scripts, use Ladder 1 with `-ExecutionPolicy Bypass`, or:

```powershell
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

### Ladder 3 — Manual binary + checksums

1. Open the release for pin **`v0.5.2`**: https://github.com/afelin/curbpack/releases/tag/v0.5.2  
2. Download the asset for your OS from the manifest (`curbpack_darwin_*`, `curbpack_linux_*`, or `curbpack_windows_amd64.exe`) **and** `checksums.txt`.  
3. Verify sha256 (fail closed if mismatch).  
4. Place the binary on PATH (`~/.local/bin` or `%LOCALAPPDATA%\Programs\Curbpack`).  
5. On Windows, copy `curbpack.exe` to `curb.exe` (alias). On Unix, `ln -s curbpack curb`.  
6. Write install marker (or run `curbpack doctor --repair` after a scripted install).

Checksum verify examples:

```bash
# macOS / Linux
grep -E 'curbpack_linux_amd64$' checksums.txt
sha256sum curbpack_linux_amd64   # or shasum -a 256
```

```powershell
# Windows
Get-FileHash -Algorithm SHA256 .\curbpack_windows_amd64.exe
Select-String -Path checksums.txt -Pattern 'curbpack_windows_amd64.exe'
```

---

## What installers guarantee

| Guarantee | Detail |
|-----------|--------|
| Fail-closed checksum | Half-download / missing checksums.txt → refuse |
| Atomic replace | Temp → verify → `.new` → replace (Windows access-denied → clear message) |
| Marker | Schema `curbpack-install-marker:1` — Unix `~/.local/share/curbpack/`; Windows `%LOCALAPPDATA%\Programs\Curbpack\` |
| Alias | `curb` / `curb.exe` beside binary |
| PATH | Unix: print export hint; Windows: persist User PATH via `[Environment]::SetEnvironmentVariable` |
| No silent auto-update | Repair never downloads |

Env overrides: `CURBPACK_VERSION`, `CURBPACK_INSTALL_DIR`, `CURBPACK_REPO`, optional `GITHUB_TOKEN`.

---

## Repair (local only)

```bash
curbpack doctor --repair
```

```powershell
curbpack doctor --repair
# or
.\install.ps1 -Repair
```

Re-asserts install dir on PATH + refreshes `curb` alias. **No network.** Exit **2** if binary missing → print install command. Defender quarantine or deleted exe → full reinstall (Ladder 1–3), not repair.

---

## SmartScreen / Unblock-File / Gatekeeper

| OS | Symptom | Ladder |
|----|---------|--------|
| Windows | SmartScreen blocks | Properties → Unblock, or `Unblock-File`, or Ladder 3 after checksum |
| Windows | ExecutionPolicy | Ladder 1 Bypass / CurrentUser RemoteSigned |
| macOS | Gatekeeper | System Settings → Privacy, or `xattr -d com.apple.quarantine` after checksum verify |
| Any | Binary missing after “AV” | Full reinstall; see [troubleshooting](troubleshooting.md) |

---

## Next

`curbpack doctor` → `curbpack demo` → product repo `init` → `check` → `share`.  
Audience paths: [60-second paths](60-second-paths.md).
