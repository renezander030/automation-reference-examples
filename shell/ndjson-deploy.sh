#!/usr/bin/env bash
# Bash 4+ and Python 3. stdout is NDJSON; logs go to stderr.
# Source this file; services in deploy_summary are a whitespace-separated list.
log() { printf '%s\n' "$*" >&2; }
_clock() { python3 -c 'import time;print(time.monotonic_ns()//1000000)'; }
_emit() { python3 -c 'import json,sys;print(json.dumps(json.loads(sys.argv[1]),separators=(",",":")))' "$1"; }
PHASES_JSON=()
CURRENT_PHASE=''
PHASE_START_MS=0
DEPLOY_START_MS=$(_clock)
phase_start() {
 CURRENT_PHASE="$1"; PHASE_START_MS=$(_clock)
 python3 -c 'import json,sys,time;print(json.dumps({"event":"phase.start","phase":sys.argv[1],"ts":time.time_ns()//1000000}))' "$1"
 log "$1"
}
phase_end() {
 local name="$1" status="$2" error="${3:-}" now dur summary
 if [ "$name" != "$CURRENT_PHASE" ]; then log 'phase name does not match active phase'; return 2; fi
 now=$(_clock); dur=$((now-PHASE_START_MS))
 python3 -c 'import json,sys;print(json.dumps({"event":"phase.end","phase":sys.argv[1],"status":sys.argv[2],"duration_ms":int(sys.argv[3]),"error":sys.argv[4]}))' "$name" "$status" "$dur" "$error"
 summary=$(python3 -c 'import json,sys;print(json.dumps({"name":sys.argv[1],"status":sys.argv[2],"duration_ms":int(sys.argv[3])}))' "$name" "$status" "$dur")
 PHASES_JSON+=("$summary");CURRENT_PHASE=''
}
deploy_summary() {
 local now dur
 now=$(_clock);dur=$((now-DEPLOY_START_MS))
 python3 -c 'import json,sys;print(json.dumps({"event":"deploy.complete","version":sys.argv[1],"services":sys.argv[2].split(),"outcome":sys.argv[3],"duration_ms":int(sys.argv[4]),"phases":[json.loads(x) for x in sys.argv[5:]]}))' "$1" "$2" "$3" "$dur" "${PHASES_JSON[@]}"
}
