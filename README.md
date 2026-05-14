<h1 align="center">userhunt</h1>

<p align="center">
  <b>Fast OSINT username enumerator across 100+ platforms.</b><br>
  Single Go binary. Install in 10 seconds. Beautiful colored output. <i>Sherlock, in Go.</i>
</p>

<p align="center">
  <a href="https://github.com/nodirsafarov/userhunt/releases"><img src="https://img.shields.io/github/v/release/nodirsafarov/userhunt?style=flat-square&color=cyan" alt="Release"></a>
  <a href="https://github.com/nodirsafarov/userhunt/actions"><img src="https://img.shields.io/github/actions/workflow/status/nodirsafarov/userhunt/release.yml?branch=main&style=flat-square" alt="Build"></a>
  <a href="https://goreportcard.com/report/github.com/nodirsafarov/userhunt"><img src="https://goreportcard.com/badge/github.com/nodirsafarov/userhunt?style=flat-square" alt="Go Report"></a>
  <a href="https://pkg.go.dev/github.com/nodirsafarov/userhunt"><img src="https://pkg.go.dev/badge/github.com/nodirsafarov/userhunt.svg" alt="Go Reference"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/nodirsafarov/userhunt?style=flat-square" alt="License"></a>
  <a href="https://github.com/nodirsafarov/userhunt/stargazers"><img src="https://img.shields.io/github/stars/nodirsafarov/userhunt?style=flat-square" alt="Stars"></a>
</p>

<p align="center">
  <code>
    $ userhunt torvalds --only-found<br>
    [+] GitHub        https://github.com/torvalds<br>
    [+] GitLab        https://gitlab.com/torvalds<br>
    [+] HackerNews    https://news.ycombinator.com/user?id=torvalds<br>
    [+] Reddit        https://www.reddit.com/user/torvalds<br>
    [+] Keybase       https://keybase.io/torvalds<br>
    ...<br>
    Found: 55 / 103 platforms — 17.0s
  </code>
</p>

---

## Why userhunt?

| | [Sherlock](https://github.com/sherlock-project/sherlock) | **userhunt** |
|---|---|---|
| Language | Python | **Go** |
| Install | `pip install` + Python runtime | `go install` or 1 binary, **zero deps** |
| Concurrency | Threads (slow) | **Goroutines, 500+ parallel** |
| Speed (100 sites) | ~4–6 min | **~15–25 sec** (10×+ faster) |
| Detection | Status code only | **Status + content markers + smart heuristics** |
| Output | Plain text | **Colored TUI + progress bar** |
| Export | TXT | **JSON, CSV, Markdown** |
| Multi-target | One at a time | **Batch usernames in one run** |
| Filters | None | **Category, NSFW toggle** |
| Retries | None | **Exponential backoff** |
| Proxy / UA | Limited | **Full HTTP(S) proxy + UA rotation** |

## Install

### Go (recommended)

```bash
go install github.com/nodirsafarov/userhunt/cmd/userhunt@latest
```

### Pre-built binaries

Grab the binary for your OS/arch from the [releases page](https://github.com/nodirsafarov/userhunt/releases).

```bash
# Linux x86_64
curl -L https://github.com/nodirsafarov/userhunt/releases/latest/download/userhunt_Linux_x86_64.tar.gz | tar xz
sudo mv userhunt /usr/local/bin/
```

### From source

```bash
git clone https://github.com/nodirsafarov/userhunt
cd userhunt
make build
./bin/userhunt --help
```

## Quick start

```bash
# Hunt a single username across all 100+ platforms
userhunt torvalds

# Only show found accounts (clean output for reports)
userhunt torvalds --only-found

# Multiple usernames in one run
userhunt torvalds linus linus_torvalds --only-found

# Export to JSON / CSV / Markdown
userhunt torvalds -o report.json
userhunt torvalds -o report.csv
userhunt torvalds -o report.md
userhunt torvalds -o "{user}_report.json"   # template: {user} -> username

# Filter by category (tech, social, gaming, security, music, video, ...)
userhunt torvalds --category security

# Crank up concurrency on a fast network
userhunt torvalds --concurrency 200 --timeout 5s

# Pipe through a proxy (Burp / mitmproxy)
userhunt torvalds --proxy http://127.0.0.1:8080
```

### List all platforms and categories

```bash
userhunt --list
```

## Example output

```
  Hunting @torvalds across 103 platforms...

  [+]  GitHub                    https://github.com/torvalds
  [+]  GitLab                    https://gitlab.com/torvalds
  [+]  HackerNews                https://news.ycombinator.com/user?id=torvalds
  [+]  Reddit                    https://www.reddit.com/user/torvalds
  [+]  Keybase                   https://keybase.io/torvalds
  [+]  Medium                    https://medium.com/@torvalds
  [+]  Last.fm                   https://www.last.fm/user/torvalds
  [+]  Steam                     https://steamcommunity.com/id/torvalds
  ...

──────────────────────────────────────────────
  Target:  torvalds
  Found:   55 / 103 platforms
  Errors:   15
  Time:    17.0s
──────────────────────────────────────────────
```

## Flag reference

| Flag | Short | Default | Description |
|---|---|---|---|
| `--timeout` | `-t` | `15s` | Per-request timeout |
| `--concurrency` | `-c` | `50` | Number of parallel workers |
| `--retries` | | `1` | Retry attempts per platform |
| `--user-agent` | | rotating | Custom User-Agent |
| `--proxy` | | | HTTP(S) proxy URL |
| `--output` | `-o` | | Write report to file |
| `--format` | `-f` | inferred | `json`, `csv`, or `md` |
| `--only-found` | | `false` | Suppress not-found / error lines |
| `--category` | | | Restrict to one category |
| `--list` | | | List platforms / categories |
| `--include-nsfw` | | `false` | Include NSFW platforms |
| `--no-color` | | `false` | Disable colored output |
| `--no-banner` | | `false` | Hide the ASCII banner |
| `--silent` | `-s` | `false` | No banner, no progress, no live lines |
| `--fail-if-not-found` | | `false` | Exit code `2` if zero accounts found |

## Use cases

- **OSINT** — investigate digital footprints, map a target's online presence.
- **Bug bounty / pentest recon** — fast attack-surface discovery during engagements.
- **CTF** — find pivot points across platforms during an OSINT-flavored challenge.
- **Personal audit** — see where your own username is registered before launching a brand.

## How detection works

For every platform, userhunt picks one of two strategies (declared in [`internal/platforms/platforms.json`](internal/platforms/platforms.json)):

- **`status`** — `200 OK` ⇒ exists, `404` / `410` ⇒ does not exist, anything else ⇒ error.
- **`content`** — fetch the page and look for marker substrings. If an `exists_content` marker is present ⇒ exists. If a `not_exists_content` marker is present ⇒ does not exist. Useful for SPAs that always return `200`.

The HTTP client uses HTTP/2, keep-alive, connection pooling, retries with exponential backoff and a rotating set of real-browser User-Agents. Each probe reads at most 256 KiB of body, so even SPA-heavy sites stay fast.

> [!IMPORTANT]
> A "found" result means the URL is **registered**, not that it belongs to your target. Always corroborate findings — usernames are shared and squatted across the web.

## Adding new platforms

Edit [`internal/platforms/platforms.json`](internal/platforms/platforms.json) and add a new object:

```json
{
  "name": "Example",
  "url": "https://example.com/u/{}",
  "category": "social",
  "check_type": "status"
}
```

For SPA / JS-heavy sites, prefer `content` mode:

```json
{
  "name": "MySPA",
  "url": "https://myspa.com/{}",
  "category": "social",
  "check_type": "content",
  "not_exists_content": ["User not found", "404"]
}
```

PRs welcome — see [CONTRIBUTING](#contributing).

## Library usage

userhunt is also importable from Go programs:

```go
import (
    "context"

    "github.com/nodirsafarov/userhunt/internal/checker"
    "github.com/nodirsafarov/userhunt/internal/platforms"
)

func scan(username string) error {
    list, err := platforms.Load()
    if err != nil {
        return err
    }
    chk, err := checker.New(checker.Options{Concurrency: 100})
    if err != nil {
        return err
    }
    for r := range chk.Run(context.Background(), username, list) {
        if r.Status == checker.StatusFound {
            // r.URL holds the matched profile URL
        }
    }
    return nil
}
```

## Roadmap

- [ ] `--watch` continuous monitoring with diff reports
- [ ] Avatar / profile-photo extraction
- [ ] HTML report with screenshots
- [ ] More platforms (target: 250+)
- [ ] Domain / email modes (`userhunt-domain`, `userhunt-email`)
- [ ] Plugin API for custom checkers

## Contributing

Issues and PRs welcome. Quick checklist:

```bash
make test
make vet
make build
./bin/userhunt yourname --only-found
```

When adding a platform, please verify both **found** and **not found** behavior locally and include an entry in the PR description.

## Legal & ethics

userhunt is intended for **lawful OSINT, security research, and personal recon**. You are responsible for complying with each platform's terms of service and your local laws. Do not use userhunt for harassment, stalking, or any activity targeting individuals without authorization.

## Credits

- Inspired by [Sherlock](https://github.com/sherlock-project/sherlock).
- Built with [cobra](https://github.com/spf13/cobra), [fatih/color](https://github.com/fatih/color) and [progressbar](https://github.com/schollz/progressbar).

## License

[MIT](LICENSE) © Nodir Safarov
