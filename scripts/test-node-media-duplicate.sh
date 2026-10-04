#!/bin/sh
# Real HTTP end-to-end test for the HWatch media-duplicate node flow.
#
# Boots a throwaway HWatch server with its own SQLite DB, media root and port,
# seeds completed media deterministically, then acts as an external node:
# heartbeat -> claim (assert candidates) -> progress/lease -> complete.
# A "whole-video" result must create a duplicate candidate; "different" must not.
#
# Requires: go, curl, python3 (stdlib sqlite3 + json). No production data is touched.
set -eu

REPO=$(cd "$(dirname "$0")/.." && pwd)
ROOT=$(mktemp -d "${TMPDIR:-/tmp}/hwatch-media-duplicate-e2e.XXXXXX")
MEDIA="$ROOT/media"
DB="$ROOT/hwatch.db"
PORT="${HWWATCH_E2E_PORT:-18877}"
BASE="http://127.0.0.1:$PORT"
JAR="$ROOT/cookies.txt"
PID=""

cleanup() {
  if [ -n "$PID" ]; then kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; fi
  rm -rf "$ROOT"
}
trap cleanup EXIT

log() { printf '==> %s\n' "$*"; }
json() { python3 -c "import sys,json;print(json.load(sys.stdin)$1)"; }

start_server() {
  R34WATCH_ADDR="127.0.0.1:$PORT" \
  R34WATCH_DB="$DB" \
  R34WATCH_WEB="$REPO/web/dist" \
  R34WATCH_MEDIA_ROOT="$MEDIA" \
  R34WATCH_ARIA_RPC="http://127.0.0.1:6899/jsonrpc" \
    "$ROOT/r34watch" >"$ROOT/server.log" 2>&1 &
  PID=$!
  i=0
  while [ "$i" -lt 60 ]; do
    if curl -fsS "$BASE/api/auth/status" >/dev/null 2>&1; then return 0; fi
    i=$((i + 1)); sleep 0.25
  done
  echo "server failed to start"; cat "$ROOT/server.log"; exit 1
}
stop_server() {
  kill "$PID" 2>/dev/null || true
  wait "$PID" 2>/dev/null || true
  PID=""
}

log "building binary"
(cd "$REPO" && go build -o "$ROOT/r34watch" ./cmd/r34watch)

mkdir -p "$MEDIA/Test/Creator"
for name in A B C D; do printf 'payload-%s' "$name" > "$MEDIA/Test/Creator/$name.mp4"; done

log "first boot to create schema"
start_server
stop_server

log "seeding completed media"
python3 - "$DB" "$MEDIA" <<'PY'
import sqlite3, sys, datetime
db, media = sys.argv[1], sys.argv[2]
now = datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
con = sqlite3.connect(db)
con.execute("INSERT INTO subscriptions(slug,name,url,platform,subscription_type,source_id,created_at,enabled,baseline_complete) VALUES('test:sub','sub','https://example.invalid/sub','rule34video','creator','',?,1,1)", (now,))
def video(vid, dur, sha, fname):
    path = f"{media}/Test/Creator/{fname}.mp4"
    con.execute("INSERT INTO videos(id,subscription_slug,title,page_url,status,file_path,plex_path,duration_seconds,downloaded_at,updated_at,discovered_at,platform,source_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)",
                (vid, "test:sub", vid, "https://example.invalid/v/"+vid, "completed", path, path, dur, now, now, now, "rule34video", ""))
    con.execute("INSERT INTO media_fingerprints(video_id,file_size,quick_hash,sha256,hash_status,updated_at) VALUES(?,?,?,?, 'completed', ?)",
                (vid, 10, "q-"+vid, sha, now))
video("test:target", 100, "sha-target", "A")
video("test:candidate", 101, "sha-candidate", "B")
video("test:target2", 200, "sha-target2", "C")
video("test:candidate2", 201, "sha-candidate2", "D")
con.commit(); con.close()
PY

log "booting server for the flow"
start_server

log "admin setup and login"
curl -fsS -c "$JAR" -H 'Content-Type: application/json' \
  -d '{"username":"e2e-admin","password":"e2e-password-123"}' \
  "$BASE/api/auth/setup" >/dev/null
CSRF=$(curl -fsS -b "$JAR" "$BASE/api/auth/status" | json "['csrf_token']")

log "create node with media_duplicate permission"
NODE=$(curl -fsS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
  -d '{"name":"e2e-node","permissions":{"dedupe":true,"transcode":false,"media_duplicate":true}}' \
  "$BASE/api/nodes")
TOKEN=$(printf '%s' "$NODE" | json "['token']")

log "heartbeat"
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"capabilities":{"dedupe":true,"transcode":false,"media_duplicate":true},"version":"e2e","platform":"linux"}' \
  "$BASE/api/node/v1/heartbeat" >/dev/null

claim_job() {
  curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"types":["dedupe"]}' "$BASE/api/node/v1/jobs/claim"
}

log "create media-duplicate job for test:target"
curl -fsS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
  -d '{"video_id":"test:target","type":"dedupe"}' "$BASE/api/compute/jobs" >/dev/null

CLAIM=$(claim_job)
printf '%s' "$CLAIM" | python3 -c '
import sys, json
claim = json.load(sys.stdin)["job"]
opts = json.loads(claim["input_json"])
assert claim["subtype"] == "media_duplicate", claim
assert opts["mode"] == "media_duplicate", opts
ids = [c["video_id"] for c in opts["candidates"]]
assert ids == ["test:candidate"], ids
assert opts["candidates"][0]["media_relative_path"] == "Test/Creator/B.mp4", opts
print("candidate payload OK:", opts["candidates"])
'
JOB_ID=$(printf '%s' "$CLAIM" | json "['job']['id']")

log "progress renews lease"
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"status":"transcoding","progress":0.5}' \
  "$BASE/api/node/v1/jobs/$JOB_ID/progress" >/dev/null

log "complete whole-video"
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"result":{"matches":[{"video_id":"test:candidate","classification":"whole-video","backend":"media-fingerprint-v1","algorithm_version":"media-fingerprint-v1","confidence":0.98,"duration_delta":1,"metrics":{"sample_count":32,"matched_samples":32,"median_hamming_distance":0,"duration_ratio":0.99}}]}}' \
  "$BASE/api/node/v1/jobs/$JOB_ID/complete" >/dev/null

curl -fsS -b "$JAR" "$BASE/api/duplicates" | python3 -c '
import sys, json
items = [i for i in json.load(sys.stdin)["items"] if i["match_type"] == "media_duplicate"]
assert len(items) == 1, items
assert items[0]["classification"] == "whole-video", items
assert items[0]["algorithm_version"] == "media-fingerprint-v1", items
print("whole-video -> duplicate candidate OK")
'

log "different must not create a candidate"
curl -fsS -b "$JAR" -H "X-CSRF-Token: $CSRF" -H 'Content-Type: application/json' \
  -d '{"video_id":"test:target2","type":"dedupe"}' "$BASE/api/compute/jobs" >/dev/null
CLAIM2=$(claim_job)
JOB2_ID=$(printf '%s' "$CLAIM2" | json "['job']['id']")
CAND2=$(printf '%s' "$CLAIM2" | python3 -c 'import sys,json;print(json.loads(json.load(sys.stdin)["job"]["input_json"])["candidates"][0]["video_id"])')
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d "{\"result\":{\"matches\":[{\"video_id\":\"$CAND2\",\"classification\":\"different\",\"confidence\":0.1}]}}" \
  "$BASE/api/node/v1/jobs/$JOB2_ID/complete" >/dev/null

curl -fsS -b "$JAR" "$BASE/api/duplicates" | python3 -c '
import sys, json
items = [i for i in json.load(sys.stdin)["items"] if i["match_type"] == "media_duplicate"]
assert len(items) == 1, items
print("different -> no new candidate OK")
'

log "PASS: media-duplicate node flow (heartbeat, claim, lease, complete) over real HTTP"
