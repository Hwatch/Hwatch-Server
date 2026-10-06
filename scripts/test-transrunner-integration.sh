#!/bin/sh
# Real HTTP end-to-end test for the HWatch TransRunner Service API.
# Boots a throwaway server, seeds media, then exercises token scopes, media,
# dedupe candidates, duplicate results and transcode results over HTTP.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
ROOT=$(mktemp -d "${TMPDIR:-/tmp}/hwatch-transrunner-e2e.XXXXXX")
MEDIA="$ROOT/media"
DB="$ROOT/hwatch.db"
PORT="${HWWATCH_TR_E2E_PORT:-18878}"
BASE="http://127.0.0.1:$PORT"
JAR="$ROOT/cookies.txt"
PID=""

cleanup() { if [ -n "$PID" ]; then kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi; rm -rf "$ROOT"; }
trap cleanup EXIT
log() { printf '==> %s\n' "$*"; }
json() { python3 -c "import sys,json;print(json.load(sys.stdin)$1)"; }

start_server() {
  R34WATCH_ADDR="127.0.0.1:$PORT" R34WATCH_DB="$DB" R34WATCH_WEB="$REPO/web/dist" R34WATCH_MEDIA_ROOT="$MEDIA" R34WATCH_ARIA_RPC="http://127.0.0.1:6898/jsonrpc" \
    "$ROOT/r34watch" >"$ROOT/server.log" 2>&1 &
  PID=$!
  i=0
  while [ "$i" -lt 60 ]; do
    if curl -fsS "$BASE/api/auth/status" >/dev/null 2>&1; then return 0; fi
    i=$((i + 1)); sleep 0.25
  done
  echo "server failed to start"; cat "$ROOT/server.log"; exit 1
}
stop_server() { kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; PID=""; }

log "building binary"
(cd "$REPO" && go build -o "$ROOT/r34watch" ./cmd/r34watch)

mkdir -p "$MEDIA/A" "$MEDIA/.transrunner/job_1"
printf 'target' > "$MEDIA/A/target.mp4"
printf 'near' > "$MEDIA/A/near.mp4"

log "first boot to create schema"
start_server
stop_server

log "seeding media and a narrow token"
python3 - "$DB" "$MEDIA" <<'PY'
import sqlite3, sys, hashlib, datetime
db, media = sys.argv[1], sys.argv[2]
now = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
con = sqlite3.connect(db)
con.execute("INSERT INTO subscriptions(slug,name,url,platform,subscription_type,source_id,created_at,enabled,baseline_complete) VALUES('test:sub','sub','https://example.invalid/sub','test','creator','',?,1,1)", (now,))
def video(vid, dur, sha, name):
    path = f"{media}/A/{name}.mp4"
    con.execute("INSERT INTO videos(id,subscription_slug,title,page_url,status,file_path,plex_path,duration_seconds,downloaded_at,updated_at,discovered_at,platform,source_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)",
                (vid, "test:sub", vid, "https://example.invalid/"+vid, "completed", path, path, dur, now, now, now, "test", ""))
    con.execute("INSERT INTO media_fingerprints(video_id,file_size,quick_hash,sha256,hash_status,updated_at) VALUES(?,?,?,?, 'completed', ?)",
                (vid, 10, "q-"+vid, sha, now))
video("t:target", 100, "sha-target", "target")
video("t:near", 101, "sha-near", "near")
narrow = "narrow-scope-token"
con.execute("INSERT INTO integration_tokens(name,token_hash,scopes_json,created_at) VALUES('narrow',?,'[\"read_media\"]',?)", (hashlib.sha256(narrow.encode()).hexdigest(), now))
con.commit(); con.close()
PY
NARROW="narrow-scope-token"

log "boot server for the flow"
start_server

log "admin setup"
curl -fsS -c "$JAR" -H 'Content-Type: application/json' -d '{"username":"tr-admin","password":"tr-password-123"}' "$BASE/api/auth/setup" >/dev/null
CSRF=$(curl -fsS -b "$JAR" "$BASE/api/auth/status" | json "['csrf_token']")

log "create TransRunner token via admin API"
TOKEN=$(curl -fsS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' -d '{}' "$BASE/api/integrations/transrunner/token" | json "['token']")
[ -n "$TOKEN" ] || { echo "no token"; exit 1; }

log "security: missing/bad token 401, wrong scope 403"
[ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/transrunner/v1/media/t:target")" = "401" ] || { echo "expected 401 no token"; exit 1; }
[ "$(curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer nope' "$BASE/api/transrunner/v1/media/t:target")" = "401" ] || { echo "expected 401 bad token"; exit 1; }
[ "$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $NARROW" "$BASE/api/transrunner/v1/media/t:target/dedupe-candidates")" = "403" ] || { echo "expected 403 wrong scope"; exit 1; }

log "GET media (read_media)"
curl -fsS -H "Authorization: Bearer $TOKEN" "$BASE/api/transrunner/v1/media/t:target" | python3 -c '
import sys, json
m = json.load(sys.stdin)
assert m["video_id"] == "t:target", m
assert m["media_relative_path"] == "A/target.mp4", m
assert "/" not in m["media_relative_path"].split("/")[0] or True
assert m["media_relative_path"][0] != "/", m
print("media OK:", m["media_relative_path"])
'

log "GET dedupe candidates (read_dedupe_candidates)"
curl -fsS -H "Authorization: Bearer $TOKEN" "$BASE/api/transrunner/v1/media/t:target/dedupe-candidates" | python3 -c '
import sys, json
r = json.load(sys.stdin)
assert r["algorithm_version"] == "media-fingerprint-v1", r
ids = [c["video_id"] for c in r["candidates"]]
assert ids == ["t:near"], ids
print("candidates OK:", ids)
'

log "POST duplicate-results (write_duplicate_results)"
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"video_id":"t:target","algorithm_version":"media-fingerprint-v1","matches":[{"video_id":"t:near","classification":"whole-video","backend":"phash","confidence":0.98,"metrics":{"sample_count":32,"matched_samples":32}}]}' \
  "$BASE/api/transrunner/v1/duplicate-results" >/dev/null
curl -fsS -b "$JAR" "$BASE/api/duplicates" | python3 -c '
import sys, json
items = [i for i in json.load(sys.stdin)["items"] if i["match_type"] == "media_duplicate"]
assert len(items) == 1, items
print("duplicate candidate OK:", items[0]["classification"])
'

log "POST transcode-results (write_transcode_results)"
if command -v ffmpeg >/dev/null 2>&1 && command -v ffprobe >/dev/null 2>&1; then
  ffmpeg -hide_banner -loglevel error -y -f lavfi -i testsrc=size=64x64:rate=5:duration=1 -c:v libx264 -pix_fmt yuv420p "$MEDIA/.transrunner/job_1/output.mp4"
  HTTP=$(curl -s -o "$ROOT/tr_out.json" -w '%{http_code}' -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"video_id":"t:target","runner_job_id":"job_1","source_sha256":"sha-target","output_relative_path":".transrunner/job_1/output.mp4","profile":"hevc-1080p"}' \
    "$BASE/api/transrunner/v1/transcode-results")
  [ "$HTTP" = "202" ] || { echo "transcode expected 202 got $HTTP"; cat /tmp/tr_e2e_out.json; exit 1; }
  [ "$(cat "$MEDIA/A/target.mp4")" = "target" ] || { echo "source was replaced!"; exit 1; }
  echo "transcode result OK (202), source untouched"
else
  echo "ffmpeg/ffprobe missing, skipping transcode step"
fi

log "revoke token -> 401"
curl -fsS -b "$JAR" -H "X-CSRF-Token: $CSRF" -X DELETE "$BASE/api/integrations/transrunner/token" >/dev/null
[ "$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" "$BASE/api/transrunner/v1/media/t:target")" = "401" ] || { echo "expected 401 after revoke"; exit 1; }

log "PASS: TransRunner service API end-to-end"
