<div align="center">
  <a href="https://github.com/Hwatch/Hwatch-Server"><img width="100" alt="HWatch logo" src="https://raw.githubusercontent.com/Hwatch/Hwatch-Server/main/public/pwa-192.png"/></a>
  <p><em>🗂️ Self-hosted NAS media subscription, downloader and Plex TV library manager — powered by Go, SQLite, aria2 and Vue/TypeScript.</em></p>
  <p>English · <a href="README.zh-CN.md">简体中文</a></p>
  <div>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/Hwatch/Hwatch-Server" alt="License" /></a>
    <a href="https://github.com/Hwatch/Hwatch-Server/commits/main"><img src="https://img.shields.io/github/last-commit/Hwatch/Hwatch-Server" alt="Last commit" /></a>
    <a href="https://github.com/Hwatch/Hwatch-Server"><img src="https://img.shields.io/github/languages/top/Hwatch/Hwatch-Server" alt="Top language" /></a>
    <a href="https://github.com/Hwatch/Hwatch-Server"><img src="https://img.shields.io/github/repo-size/Hwatch/Hwatch-Server" alt="Repo size" /></a>
    <a href="https://github.com/Hwatch/Hwatch-Server/stargazers"><img src="https://img.shields.io/github/stars/Hwatch/Hwatch-Server?color=%23ED8936" alt="Stars" /></a>
    <a href="https://github.com/Hwatch/Hwatch-Server/issues"><img src="https://img.shields.io/github/issues/Hwatch/Hwatch-Server" alt="Issues" /></a>
  </div>
</div>

Docs: [HTTP API V1](API_V1.md) · [OpenAPI 3.1](openapi.yaml) · [Roadmap](TODO.md) · [Agent guide](AGENTS.md)

## Preview

![HWatch overview (demo mode)](docs/screenshots/dashboard.png)

## Features

- Full-pagination creator subscriptions; builds a complete baseline on first run and auto-downloads new works afterwards
- SQLite stores subscriptions, detail metadata, tasks, traffic and Plex season/episode numbering
- aria2 JSON-RPC and system wget downloaders; the engine can be switched for new tasks while existing tasks keep theirs
- Persistent download state machine, pause/resume, resumable transfers, 429 exponential backoff, live speed and connection counts
- Configurable global speed limit, per-task connections, concurrency and download directory
- Highest-quality source resolved on demand; multiple renditions are never stored in the database
- HTTP/HTTPS proxy applied to both scraping and aria2 downloads
- Daily scheduled checks (default Asia/Shanghai 04:00)
- Plex TV layout, tvshow.nfo, per-episode NFO, posters and episode thumbnails
- After download, ffprobe records H.264/H.265/AV1, audio codec, resolution, frame rate, bit rate, bit depth and container
- In-app library browsing, search, pagination, preview and playback
- Paste video-detail HTML to import
- NAS disk usage and cumulative download traffic stats
- Platform adapter registry; every source is integrated through an isolated adapter
- Safe WebAI translation JSON export, source-hash verification and re-import
- First launch creates an admin account; sessions are protected by a cookie and CSRF afterwards
- Scheduled SQLite backups, SHA-256 verification, and path/task reconciliation after restore
- Rule34Video creator and Hanime1 user/playlist subscriptions via a unified adapter layer
- Post-download quick hash plus full SHA-256 verification (pre-download dedupe removed; dedupe is led by TransRunner)
- Per-video delete: delete media only, or delete and add to "Excluded"; persistent exclusions block future auto-downloads by `video_id` and can be restored anytime
- Structured audit log, task status history, persistent notifications and downloader health checks
- Manifest, icons and safe-area handling for adding to the iPhone home screen

Video compute (dedupe analysis, transcoding) is led by a separately running TransRunner service. Responsibility boundary: **HWatch is the media authority** (acquisition, download, subscriptions, metadata, Plex/NFO, final media persistence and safe replacement, persisting dedupe/transcode results); **TransRunner is the compute authority** (batch scanning, H.264→HEVC, VDF dedupe, FFmpeg/VideoToolbox, task queue and progress). HWatch does not start or control TransRunner tasks, does not poll progress, and performs no pre-download dedupe; it only issues scoped, revocable Service Tokens so TransRunner can read media data and write results back. See [docs/TRANSRUNNER_INTEGRATION.md](docs/TRANSRUNNER_INTEGRATION.md).

## Platform adapters

The unified interface lives in `internal/adapters`. A platform implementation only recognizes subscription URLs, lists videos, reads details and resolves download sources; it never touches SQLite, the downloader or Plex directly. Internal video IDs use `platform:source_id`, so adding a new source needs no changes to the download center or the library.

Built in:

- Rule34Video creator subscriptions
- Hanime1 user-upload subscriptions
- Hanime1 playlist subscriptions

Source pages can change. Adapters ship snapshot and contract tests and never delete existing media when scraping anomalies are detected.

## WebAI translation

Open the "Translations" page, pick a target language and export JSON. The export contains only platform identifiers, titles, tags and source hashes — no proxy config, RPC secrets, cookies, download URLs or file paths. WebAI must keep `key`, `source_hash` and `fields` and only fill in `translations`. On import, entries whose source hash does not match are skipped; translations are stored in a separate table and never overwrite the originals.

## Prebuilt binaries

Every tagged release (`v*`) is built by GitHub Actions into static Linux binaries for `amd64` and `arm64` — see [Releases](https://github.com/Hwatch/Hwatch-Server/releases).

```bash
tar xzf hwatch-v1.0.0-linux-amd64.tar.gz
cd hwatch-v1.0.0-linux-amd64
sudo install -m755 r34watch /usr/local/bin/r34watch
sudo mkdir -p /opt/hwatch/web && sudo cp -r web/dist /opt/hwatch/web/dist
sudo cp hwatch.service /etc/systemd/system/   # edit paths inside first
sudo systemctl enable --now hwatch
```

The binary serves the frontend from `R34WATCH_WEB` (default `web/dist`), so keep `r34watch` and `web/dist` together (or point `R34WATCH_WEB` at the extracted directory). The host needs `aria2`, `wget` and `ffmpeg` (ffprobe).

## Docker Compose

First replace the NAS path in `compose.yaml` with your real Plex media directory:

    /path/on/nas/Plex-TV:/media

Then:

    docker compose up -d --build

Open:

    http://NAS-IP:8787

SQLite and the aria2 session live in the `data` directory. Plex mounts the same host media directory as a TV library, so HWatch and Plex read the same video files without a second copy.

## Local development

Requires Node.js, Go 1.24, aria2 and FFmpeg (for ffprobe):

    npm install
    npm run build
    aria2c --enable-rpc=true --rpc-listen-all=false --daemon=true
    go run ./cmd/r34watch

Before committing:

    go test ./...
    go vet ./...
    npm run build
    go build -o r34watch ./cmd/r34watch
    git diff --check

Environment variables:

- R34WATCH_ADDR: listen address, default :8787
- R34WATCH_DB: SQLite path, default r34watch.db
- R34WATCH_WEB: frontend build directory, default web/dist
- R34WATCH_MEDIA_ROOT: media root shared by downloads and Plex
- R34WATCH_ARIA_RPC: aria2 JSON-RPC endpoint
- R34WATCH_ARIA_SECRET: aria2 RPC secret

On first visit you must create an admin account. Even with auth enabled, if you need public access, put it behind an HTTPS reverse proxy or VPN and restrict source addresses.

## API & clients

- `/openapi.yaml`: machine-readable OpenAPI 3.1 contract
- `/docs`: built-in read-only API docs
- `src/api.generated.ts`: generated Vue/TypeScript types
- `clients/android-kotlin`: generated Android Kotlin client

After changing the API:

    ./scripts/generate-api-clients.sh

## Data & security

- Never commit SQLite files, media, aria2 sessions, proxy credentials, integration tokens or signed download URLs.
- Media and Plex share one directory; confirm NFS is mounted before deleting or replacing media.
- SQLite backups may contain proxy and RPC config; keep them in a restricted directory.
- The HTTP mode of any future integration API is only suitable for a trusted LAN; use HTTPS or a VPN across the internet.

## Project status

Frontend 2.0 is the default UI in this repository. HWatch is under active development; see [TODO.md](TODO.md) for open items and the long-term plan.

## Disclaimer

- HWatch is an open-source, self-hosted tool for **personal** media management. It does **not** host, bundle, mirror or distribute any media or third-party content.
- It talks to third-party platforms only to fetch metadata and download links for the content **you** choose. You are solely responsible for complying with all applicable laws and the terms of service of those platforms.
- This project is **not affiliated with, endorsed by, or sponsored by** any content source or platform.
- Intended for **adults only**. Do not use it to access, store or share content that is illegal in your jurisdiction or that infringes copyright.
- The software is provided **"as is"**, without warranty of any kind. The authors and contributors are not liable for any misuse, data loss or damages.
- Do not expose your instance to the public internet without authentication, HTTPS and network restrictions.

## License

MIT License. See [LICENSE](LICENSE).
