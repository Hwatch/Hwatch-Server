# TransRunner Integration

HWatch and TransRunner have separate authorities:

- **HWatch = media library authority** — content acquisition, downloads,
  subscriptions, metadata, Plex/NFO, final file placement and safe replacement,
  duplicate/transcode result persistence, OpenAPI.
- **TransRunner = compute authority** — runs on the Mac: batch library scan,
  H.264 → HEVC, VDF duplicate detection, FFmpeg/VideoToolbox, its own job queue,
  progress, cancel and retry.

HWatch does **not** start or control TransRunner jobs and does **not** poll
progress or mirror runner state. There is no compute scheduler and no
pre-download dedupe in HWatch. TransRunner reads HWatch data and pushes results
back through the scoped API below.

## Service token

Create a scoped token in **设置 → TransRunner 集成** (or the admin API). The
plaintext is shown exactly once; only the SHA-256 hash is stored.

Fixed scope set:

```
read_media
read_dedupe_candidates
write_duplicate_results
write_transcode_results
```

Admin API (session cookie + CSRF):

```
GET    /api/integrations/transrunner/token   # status only, never the token
POST   /api/integrations/transrunner/token   # {"rotate": true} to replace
DELETE /api/integrations/transrunner/token   # revoke
```

Authenticate service calls with `Authorization: Bearer <service-token>`.
Missing/invalid/revoked → `401`; valid token without the required scope → `403`.
The OpenAPI describes every endpoint, but the Service Token is limited by scope:
it can never delete media, change settings, manage users/subscriptions, restore
backups or perform admin operations.

## Reading the library

### `GET /api/videos` — scope `read_media`

Standard paginated media list (`limit`/`offset`/`status`/`codec`/`sort`… see
OpenAPI). A Service Token may call it read-only to enumerate the library; in that
case `file_path` and `plex_path` are emptied. Use the media endpoint below for the
relative path and full metadata.

### `GET /api/transrunner/v1/media/{video_id}` — scope `read_media`

```json
{
  "video_id": "rule34video:123",
  "media_relative_path": "Rule34Video/Creator/file.mp4",
  "file_size": 123456789,
  "duration": 1030,
  "container": "mp4",
  "video_codec": "h264",
  "audio_codec": "aac",
  "width": 1920,
  "height": 1080,
  "fps": 30,
  "bit_depth": 8,
  "bitrate": 8000000,
  "sha256": "..."
}
```

`media_relative_path` is relative to HWatch's media root; TransRunner maps it with
its own media root. If the record exists but storage is unavailable, HWatch
returns `503 media_unavailable` (not `404`).

### `GET /api/transrunner/v1/media/{video_id}/dedupe-candidates` — scope `read_dedupe_candidates`

HWatch filters candidates server-side; TransRunner never scans the library.

Query: `?limit=` (default 20, max 100). Candidate rule: completed media, not
self, media file present, duration within `max(2s, duration × 1%)`, not an exact
SHA-256 duplicate, not already confirmed/dismissed.

```json
{ "video_id": "target-id", "algorithm_version": "media-fingerprint-v1",
  "candidates": [ { "video_id": "...", "media_relative_path": "...", "duration": 1028, "sha256": "...", "file_size": 123 } ] }
```

## Writing results

### `POST /api/transrunner/v1/duplicate-results` — scope `write_duplicate_results`

```json
{ "video_id": "A", "algorithm_version": "vdf-4.1.1",
  "matches": [ { "video_id": "B", "classification": "whole-video", "backend": "vdf", "confidence": 0.97, "metrics": {"vdf_similarity": 99.99} } ] }
```

`classification` ∈ `whole-video`, `trimmed-video`, `possible-match`, `different`.
The first three create a reviewable duplicate candidate; `different` creates none.
Unknown classification/unknown match → `422`; unknown source → `404`. Pairs are
canonicalised (`min/max(video_id)`). This endpoint only writes
`duplicate_candidates`; it never deletes, renames or replaces media.

### `POST /api/transrunner/v1/transcode-results` — scope `write_transcode_results`

```json
{
  "video_id": "rule34video:123",
  "runner_job_id": "job_xxx",
  "source_sha256": "...",
  "output_relative_path": ".transrunner/job_xxx/output.mp4",
  "profile": "hevc-preserve",
  "apply_mode": "replace_source"
}
```

- `output_relative_path` must resolve inside `.transrunner/` under the media root;
  traversal, absolute paths and symlink escapes are rejected (`422 invalid_path`).
- HWatch runs `ffprobe` itself; probe failure → `422 probe_failed`.
- If `source_sha256` is provided and differs from the current source hash →
  `409 source_changed` (never replaced).
- **Without `apply_mode`** (backward compatible): the result is recorded as
  `pending` and the source is not touched; response `202 {"status":"validated"}`.
- **With `apply_mode: "replace_source"`**: after extra `hevc-preserve` validation
  HWatch atomically replaces the source and updates DB metadata from its own
  ffprobe result; response `200 {"status":"applied"}`.

`hevc-preserve` validation (HWatch-probed, never trusts the callback):

- output is HEVC; same width/height; frame-rate within tolerance; duration within
  `max(2s, 1%)`; bit depth not reduced; color primaries/transfer preserved when
  both declare them.

Replacement is transactional: source → `.hwatch-pre-transcode` backup, staged
output → original path (rename, with an EXDEV-safe same-directory copy fallback),
then DB metadata + file hash refresh; on failure the original source is restored.
`transcode_results.status` becomes `applied` or `failed`.

## What HWatch does not do

- No HWatch → TransRunner job submission (submit/test endpoints removed).
- No task progress polling, lease, heartbeat or claim.
- No pre-download dedupe: the download flow is discover → queued → download →
  media import, without waiting on TransRunner/VDF.
- Content dedupe is driven entirely by TransRunner's own batch processing.
