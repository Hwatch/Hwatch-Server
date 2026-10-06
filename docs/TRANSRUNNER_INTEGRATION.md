# TransRunner Integration

HWatch and TransRunner have separate authorities:

- **HWatch = media library authority** — subscriptions, downloads, Plex/NFO,
  metadata, duplicate review.
- **TransRunner = computation authority** — it owns its own task queue,
  scheduling, progress and runner state.

HWatch does **not** manage TransRunner jobs, runner nodes, progress, heartbeat,
leases or capabilities. There is no compute scheduler in HWatch. This document
describes only the callback surface HWatch exposes for TransRunner to read
media/library data and write results back.

## Service token

Create a scoped token in **设置 → TransRunner 集成** (or via the admin API). The
plaintext token is shown exactly once; only its SHA-256 hash is stored.

The default scope set is fixed to:

```
read_media
read_dedupe_candidates
write_duplicate_results
write_transcode_results
```

There is intentionally no scope for deleting/moving media, managing users or
subscriptions, changing settings, or backup/restore.

Admin API (session cookie + CSRF):

```
GET    /api/integrations/transrunner/token   # status only, never the token
POST   /api/integrations/transrunner/token   # {"rotate": true} to replace an existing token
DELETE /api/integrations/transrunner/token   # revoke
```

Authenticate service calls with:

```
Authorization: Bearer <service-token>
```

Missing/invalid/revoked token → `401`. Valid token without the required scope →
`403`. `last_used_at` is updated at most once per minute.

## Endpoints

Base prefix: `/api/transrunner/v1`. All responses never include absolute paths,
proxy settings, download URLs, cookies or secrets.

### `GET /api/transrunner/v1/media/{video_id}` — scope `read_media`

Returns the metadata TransRunner needs to locate and inspect the media.

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

`media_relative_path` is relative to HWatch's media root; TransRunner maps it
using its own media root. If the DB record exists but storage is unavailable,
HWatch returns `503 media_unavailable` (not `404`).

### `GET /api/transrunner/v1/media/{video_id}/dedupe-candidates` — scope `read_dedupe_candidates`

HWatch filters candidates server-side; TransRunner never scans the library.

Query: `?limit=` (default 20, max 100).

Candidate rule: completed media, not self, media file present, duration within
`max(2s, duration × 1%)`, not an exact SHA-256 duplicate, and not already
confirmed/dismissed.

```json
{
  "video_id": "target-id",
  "algorithm_version": "media-fingerprint-v1",
  "candidates": [
    { "video_id": "...", "media_relative_path": "...", "duration": 1028, "sha256": "...", "file_size": 123 }
  ]
}
```

### `POST /api/transrunner/v1/duplicate-results` — scope `write_duplicate_results`

```json
{
  "video_id": "A",
  "algorithm_version": "media-fingerprint-v1",
  "matches": [
    { "video_id": "B", "classification": "whole-video", "backend": "phash", "confidence": 0.97, "metrics": {"matched_samples": 31, "sample_count": 32} }
  ]
}
```

`classification` ∈ `whole-video`, `trimmed-video`, `possible-match`, `different`.
The first three create a reviewable duplicate candidate; `different` creates
none. Unknown classification/unknown match → `422`; unknown source → `404`.

The server validates both videos exist and canonicalises the pair
(`min/max(video_id)`) so A→B and B→A never create two rows. This endpoint only
inserts/updates `duplicate_candidates`; it never deletes, renames or replaces
media, and never auto-confirms.

### `POST /api/transrunner/v1/transcode-results` — scope `write_transcode_results`

```json
{
  "video_id": "rule34video:123",
  "runner_job_id": "job_xxx",
  "source_sha256": "...",
  "output_relative_path": ".transrunner/job_xxx/output.mp4",
  "profile": "hevc-1080p"
}
```

- `output_relative_path` must resolve inside the staging root `.transrunner/`
  under the media root; traversal, absolute paths and symlink escapes are
  rejected (`422 invalid_path`).
- HWatch runs `ffprobe` itself on the staged output; probe failure →
  `422 probe_failed`.
- If `source_sha256` is provided and differs from the current source hash →
  `409 source_changed`.
- On success the result is recorded as `pending` and the endpoint returns
  `202 {"status":"validated"}`.

This version **does not** replace the source, delete the original, or refresh
Plex. Commit/replace is a later phase.

## HWatch → TransRunner (submission)

HWatch can submit a **transcode** job to TransRunner's own task system.

Settings (`GET/PUT /api/settings`):

- `enable_transrunner` — enable submission.
- `transrunner_url` — TransRunner base URL.
- `transrunner_api_token` — write-only secret; **never** returned by `GET /api/settings`
  (only `transrunner_token_configured` is exposed). An empty value preserves the
  existing token; clear it with `transrunner_api_token_clear: true`.
- `transrunner_default_profile` — profile ID used when the request omits one.

Admin endpoints (session cookie + CSRF):

```
POST /api/integrations/transrunner/test
POST /api/integrations/transrunner/submit/transcode   {"video_id":"...","profile":"...","force":false}
```

`test` calls TransRunner `/health` then the authenticated `/api/v1/capabilities`
and returns `{ok, version, platform, capabilities}`. Result is audited as
`transrunner_tested`; submission is audited as `transrunner_job_submitted`.

Submission sends to TransRunner `POST /api/v1/jobs`:

```json
{ "input": { "type": "hwatch", "video_id": "rule34video:123", "media_relative_path": "Rule34Video/A/file.mp4" },
  "profile": "hevc-1080p" }
```

HWatch sends only `video_id` and `media_relative_path` (never an absolute path,
ffmpeg arguments, callback URL or callback token — TransRunner already holds the
HWatch URL, HWatch Service Token and media root). The returned `runner_job_id` is
stored in `transrunner_submissions`; a repeat submission of the same
video/operation/profile returns `409 already_submitted` unless `force: true`.

HWatch does **not** poll TransRunner, mirror progress/leases, or run a local
runner state machine. The result comes back through the TransRunner → HWatch
callback API above.

### Required TransRunner contract additions

The current TransRunner HTTP API (v0.1.x) accepts only `input.type` `url`/`path`
and has no content-duplicate job. For this integration to work end-to-end,
TransRunner must add:

1. `input.type = "hwatch"` accepting `video_id` + `media_relative_path`
   (resolved against TransRunner's configured HWatch media root). HWatch will
   never send an absolute path.
2. An HTTP content-duplicate job (e.g. `kind = "dedupe"` / `media_duplicate`) that
   accepts the same `hwatch` input plus candidates, and posts results back to
   HWatch's `/api/transrunner/v1/duplicate-results`.

Until (2) exists, HWatch does not expose a duplicate-submission endpoint.
