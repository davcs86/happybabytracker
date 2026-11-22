#!/usr/bin/env bash
# Runs both images together and asserts the contract end to end, through the UI only.
set -euo pipefail

API_IMAGE="${1:?usage: smoke.sh <api-image> <ui-image>}"
UI_IMAGE="${2:?usage: smoke.sh <api-image> <ui-image>}"

API_MEM_LIMIT_MIB=60
UI_MEM_LIMIT_MIB=140
UI_PORT="${SMOKE_UI_PORT:-9212}"

rand() { head -c "$1" /dev/urandom | od -An -tx1 | tr -d ' \n'; }

SUFFIX="$(rand 6)"
API="hbt-smoke-api-${SUFFIX}"
UI="hbt-smoke-ui-${SUFFIX}"
NETWORK="hbt-smoke-${SUFFIX}"
VOLUME="hbt-smoke-${SUFFIX}"
WORK="$(mktemp -d)"

INTERNAL_API_TOKEN="$(rand 24)"
ADMIN_USERNAME="adm-${SUFFIX}"
ADMIN_PASSWORD="$(rand 24)"
ADMIN_PASSWORD_2="$(rand 24)"
PASSWORD_A="$(rand 24)"
PASSWORD_B="$(rand 24)"
PASSWORD_C="$(rand 24)"
USER_A="smoke-a"
USER_B="smoke-b"
USER_C="smoke-c"
KID_NAME="Kid${SUFFIX}"
KID_ZONE=Europe/Madrid

UI_URL="http://127.0.0.1:${UI_PORT}"
ORIGIN="https://localhost:${UI_PORT}"
FORWARDED_HOST="localhost:${UI_PORT}"

# Every secret and identifier the logs must never carry; tokens are appended as they appear.
SECRETS_FILE="$WORK/secrets"
printf '%s\n' "$INTERNAL_API_TOKEN" "$ADMIN_USERNAME" "$ADMIN_PASSWORD" "$ADMIN_PASSWORD_2" \
  "$PASSWORD_A" "$PASSWORD_B" "$PASSWORD_C" "$USER_A" "$USER_B" "$USER_C" "$KID_NAME" >"$SECRETS_FILE"
remember() { printf '%s\n' "$1" >>"$SECRETS_FILE"; }
# Tracker notes and texts (also in SECRETS_FILE), and volume figures, matched on digit boundaries.
NOTES_FILE="$WORK/notes"
VALUES_FILE="$WORK/values"
: >"$NOTES_FILE"
: >"$VALUES_FILE"
note() { remember "$1"; printf '%s\n' "$1" >>"$NOTES_FILE"; echo "$1"; }
remember_value() { printf '%s\n' "$1" >>"$VALUES_FILE"; }

cleanup() {
  docker rm -f "$UI" "$API" >/dev/null 2>&1 || true
  docker network rm "$NETWORK" >/dev/null 2>&1 || true
  docker volume rm "$VOLUME" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

pass() { echo "PASS: $1"; }

scrub() { # stdin -> stdout with every remembered secret and identifier redacted
  local text p
  text="$(cat)"
  while IFS= read -r p; do text="${text//"$p"/[redacted]}"; done <"$SECRETS_FILE"
  printf '%s\n' "$text"
}

oom_killed() { # container -> succeeds when the kernel OOM-killed it
  [ "$(docker inspect --format '{{.State.OOMKilled}}' "$1" 2>/dev/null)" = true ]
}

fail() {
  local c
  for c in "$API" "$UI"; do
    if oom_killed "$c"; then echo "FAIL: $c was OOM-killed" >&2; fi
  done
  echo "FAIL: $1${2:+: $2}" >&2
  if [ -s "$WORK/api-1.log" ]; then scrub <"$WORK/api-1.log" >&2; fi
  docker logs "$API" 2>&1 | scrub >&2 || true
  docker logs "$UI" 2>&1 | scrub >&2 || true
  exit 1
}

fail_if_oom() { # assertion
  local c
  for c in "$API" "$UI"; do
    if oom_killed "$c"; then fail "$1" "$c was OOM-killed"; fi
  done
}

port_busy() {
  (exec 3<>"/dev/tcp/127.0.0.1/$1") 2>/dev/null
}

wait_ok() { # url
  local _
  for _ in $(seq 1 30); do
    if curl -fsS --max-time 10 "$1" >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  return 1
}

start_api() { # admin-password
  docker run -d --name "$API" --network "$NETWORK" \
    --read-only --cap-drop ALL --security-opt no-new-privileges --memory 96m --memory-swap 96m \
    -v "$VOLUME":/data \
    -e ADMIN_USERNAME="$ADMIN_USERNAME" -e ADMIN_PASSWORD="$1" \
    -e INTERNAL_API_TOKEN="$INTERNAL_API_TOKEN" -e PUBLIC_ORIGIN="$ORIGIN" \
    "$API_IMAGE" >/dev/null
}

# Sent as the proxy sends it; one client address per actor keeps the login limiter apart.
req() { # name client-addr cookie|- curl-args... -> status; body and headers in $WORK/<name>.*
  local name="$1" addr="$2" cookie="$3"
  shift 3
  local args=(-sS --max-time 10 -o "$WORK/$name.body" -D "$WORK/$name.headers" -w '%{http_code}'
    -H 'X-Forwarded-Proto: https' -H "X-Forwarded-Host: ${FORWARDED_HOST}"
    -H "X-Forwarded-For: ${addr}" -H 'Accept: text/html')
  if [ "$cookie" != "-" ]; then args+=(-H "Cookie: hbt_session=${cookie}"); fi
  curl "${args[@]}" "$@" || echo 000
}

# A form POST as a browser without JavaScript sends it (SvelteKit checks the Origin).
form() { # name addr cookie path field=value...
  local name="$1" addr="$2" cookie="$3" path="$4"
  shift 4
  local data=() kv
  for kv in "$@"; do data+=(--data-urlencode "$kv"); done
  req "$name" "$addr" "$cookie" -X POST "$UI_URL$path" -H "Origin: ${ORIGIN}" \
    -H 'Content-Type: application/x-www-form-urlencoded' "${data[@]}"
}

location() { # name
  grep -i '^location:' "$WORK/$1.headers" | sed -E 's/^[^:]*: *//' | tr -d '\r' || true
}

set_cookie() { # name -> the hbt_session value the response sets ('' when none)
  { grep -i '^set-cookie: hbt_session=' "$WORK/$1.headers" |
    sed -E 's/^[^=]*=([^;]*).*/\1/' | tr -d '\r'; } || true
}

body_has() { grep -qF -- "$2" "$WORK/$1.body"; }

login() { # name addr username password -> session cookie
  local code session
  code="$(form "$1" "$2" - /login "username=$3" "password=$4")"
  session="$(set_cookie "$1")"
  { [ "$code" = 303 ] && [ -n "$session" ]; } || fail "login $1" "status $code, no session cookie"
  remember "$session"
  echo "$session"
}

parent_id() { # body-name username -> the parent's id from the admin list
  tr -d '\n' <"$WORK/$1.body" |
    grep -oE "name=\"parent_id\" value=\"[^\"]+\"/> *<input type=\"hidden\" name=\"username\" value=\"$2\"" |
    head -n 1 | sed -E 's/.*name="parent_id" value="([^"]+)".*/\1/' || true
}

invite_url() { # body-name -> the link shown to the admin
  grep -oE 'data-testid="parent-link">[^<]*' "$WORK/$1.body" | sed -E 's/.*>[[:space:]]*//' |
    head -n 1 || true
}

redeem() { # name addr url password -> session cookie
  local path="${3#"$ORIGIN"}" code session
  [ "$path" != "$3" ] || fail "invite link $1" "link is not under $ORIGIN"
  remember "${path##*/}"
  code="$(form "$1" "$2" - "$path" "password=$4" "confirm=$4")"
  session="$(set_cookie "$1")"
  { [ "$code" = 303 ] && [ -n "$session" ]; } || fail "redeem $1" "status $code, no session cookie"
  remember "$session"
  echo "$session"
}

# The container's memory high-water mark: cgroup v1 max_usage_in_bytes or v2 memory.peak.
peak_file() { # container -> the readable high-water file, or nothing
  local pid path f
  pid="$(docker inspect --format '{{.State.Pid}}' "$1")"
  path="$(sed -n 's/^[0-9]*:memory:\(.*\)$/\1/p' "/proc/$pid/cgroup" 2>/dev/null || true)"
  f="/sys/fs/cgroup/memory${path}/memory.max_usage_in_bytes"
  if [ -n "$path" ] && [ -r "$f" ]; then echo "$f"; return 0; fi
  path="$(sed -n 's/^0::\(.*\)$/\1/p' "/proc/$pid/cgroup" 2>/dev/null || true)"
  f="/sys/fs/cgroup${path}/memory.peak"
  if [ -n "$path" ] && [ -r "$f" ]; then echo "$f"; fi
}

peak_mib() { # container -> high-water MiB rounded up, or nothing when no file is readable
  local f
  f="$(peak_file "$1")"
  [ -n "$f" ] || return 0
  awk '{ v = $1 / 1048576; printf "%d\n", (v == int(v)) ? v : int(v) + 1 }' "$f"
}

mem_mib() { # container -> resident MiB (integer, rounded up; 99999 when unreadable)
  { docker stats --no-stream --format '{{.MemUsage}}' "$1" || echo '? ?'; } | awk '{
    v = $1; u = v; sub(/[0-9.]+/, "", u); sub(/[A-Za-z]+$/, "", v)
    if (u == "B") v /= 1048576; else if (u == "KiB") v /= 1024
    else if (u == "GiB") v *= 1024; else if (u != "MiB") v = 99999
    printf "%d\n", (v == int(v)) ? v : int(v) + 1 }'
}

NOW="$(date +%s)"
ago() { # minutes -> that instant as datetime-local text in the kid's zone (GNU date)
  TZ="$KID_ZONE" date -d "@$((NOW - $1 * 60))" '+%Y-%m-%dT%H:%M'
}

pieces() { # name marker -> the body on one line, split before every marker
  tr -d '\n' <"$WORK/$1.body" | sed "s|$2|\n&|g"
}

row_path() { # body-name text -> the edit path of the timeline row that shows text
  local tail
  tail="$(pieces "$1" '<li' | grep -F -- "$2" | grep -oE 'log/[a-z]+/[0-9a-f-]+"' | head -n 1 | tr -d '"' || true)"
  if [ -n "$tail" ]; then echo "$KID_PATH/$tail"; fi
}

timer_id() { # body-name action -> the entry id the home's form for that action posts
  pieces "$1" '<form' | grep -F "action=\"?/$2\"" | grep -oE 'name="id" value="[^"]+"' |
    head -n 1 | sed -E 's/.*value="([^"]+)"/\1/' || true
}

modules_on() { # body-name -> the module fields of every ticked box on the modules page
  tr -d '\n' <"$WORK/$1.body" | grep -oE '<input[^>]*name="module"[^>]*>' | grep -E ' checked' |
    sed -E 's/.* value="([^"]+)".*/module=\1/' || true
}

count() { { grep -oF -- "$2" "$WORK/$1.body" || true; } | wc -l | tr -d ' '; }

# A figure may sit beside digits only in timestamps, durations, statuses and ids, so those go first.
logged_value() { # log-file -> succeeds when a remembered volume figure is in it
  local v plain="$1.plain"
  sed -E 's/"(time|request_id)":"[^"]*"//g; s/"(status|duration_ms|timeout_ms)":[0-9]+//g
    s/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}//g' "$1" >"$plain"
  while IFS= read -r v; do
    if grep -qE "(^|[^0-9.])${v//./\\.}([^0-9]|$)" "$plain"; then return 0; fi
  done <"$VALUES_FILE"
  return 1
}

ADDR_ADMIN=203.0.113.10
ADDR_A=203.0.113.11
ADDR_B=203.0.113.12
ADDR_C=203.0.113.13

if port_busy "$UI_PORT"; then
  echo "FAIL: port $UI_PORT is busy; set SMOKE_UI_PORT" >&2
  exit 1
fi

docker network create "$NETWORK" >/dev/null
docker volume create "$VOLUME" >/dev/null
start_api "$ADMIN_PASSWORD"
docker run -d --name "$UI" --network "$NETWORK" \
  --read-only --cap-drop ALL --security-opt no-new-privileges --memory 192m --memory-swap 192m \
  -p "127.0.0.1:${UI_PORT}:9212" \
  -e API_BASE_URL="http://${API}:9210" -e ORIGIN="$ORIGIN" \
  -e INTERNAL_API_TOKEN="$INTERNAL_API_TOKEN" \
  "$UI_IMAGE" >/dev/null

# Memory is the cgroup high-water mark when the host exposes it, else sampled docker stats.
MEM_FILE="$(peak_file "$API")"
MEM_FILE="${MEM_FILE##*/}"
mem() { if [ -n "$MEM_FILE" ]; then peak_mib "$1"; else mem_mib "$1"; fi; }

# 1. Health through the UI, and the container contract.
wait_ok "$UI_URL/healthz" || fail "1 ui /healthz" "no 200 within 30 s"
wait_ok "$UI_URL/healthz/api" || fail "1 ui /healthz/api" "no 200 within 30 s"
[ -z "$(docker port "$API")" ] || fail "1 api port" "the API publishes a port"
for c in "$API" "$UI"; do
  [ "$(docker inspect --format '{{.HostConfig.ReadonlyRootfs}}' "$c")" = true ] ||
    fail "1 read-only rootfs" "$c"
  [ "$(docker inspect --format '{{.Config.User}}' "$c")" = 10001:10001 ] || fail "1 uid" "$c"
done
pass "1 /healthz and /healthz/api are 200; the API port is not published; read-only, uid 10001"

# 2. The origin check.
code="$(curl -sS --max-time 10 -o "$WORK/bad-host.body" -w '%{http_code}' -H 'X-Forwarded-Proto: https' \
  -H 'X-Forwarded-Host: evil.example' -H "X-Forwarded-For: ${ADDR_ADMIN}" "$UI_URL/login" || true)"
{ [ "$code" = 400 ] && body_has bad-host 'Bad origin'; } ||
  fail "2 wrong Host" "expected the app's 400 Bad origin, got $code"
pass "2 a wrong Host gets the app's 400 Bad origin"

# 3. Admin creates smoke-a; smoke-a redeems the invite and creates a kid.
ADMIN_SESSION="$(login admin-login "$ADDR_ADMIN" "$ADMIN_USERNAME" "$ADMIN_PASSWORD")"
code="$(form admin-create-a "$ADDR_ADMIN" "$ADMIN_SESSION" '/admin/parents?/create' \
  "username=$USER_A" 'display_name=Smoke A')"
[ "$code" = 200 ] || fail "3 create $USER_A" "status $code"
SESSION_A="$(redeem redeem-a "$ADDR_A" "$(invite_url admin-create-a)" "$PASSWORD_A")"
code="$(form kid-new "$ADDR_A" "$SESSION_A" /kids/new "name=$KID_NAME" 'birth_date=2026-01-15' \
  'sex=female' "time_zone=$KID_ZONE")"
KID_PATH="$(location kid-new)"
case "$code $KID_PATH" in
  "303 /kids/"*) ;;
  *) fail "3 create kid" "status $code, Location $KID_PATH" ;;
esac
code="$(req kid-a "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
{ [ "$code" = 200 ] && body_has kid-a "$KID_NAME"; } || fail "3 kid page" "status $code"
pass "3 admin creates $USER_A, the invite is redeemed, $USER_A creates a kid"

# 4. smoke-b becomes a caregiver: sees the kid, gets a 404 page for its access screen.
code="$(form admin-create-b "$ADDR_ADMIN" "$ADMIN_SESSION" '/admin/parents?/create' \
  "username=$USER_B" 'display_name=Smoke B')"
[ "$code" = 200 ] || fail "4 create $USER_B" "status $code"
SESSION_B="$(redeem redeem-b "$ADDR_B" "$(invite_url admin-create-b)" "$PASSWORD_B")"
code="$(form grant-b "$ADDR_A" "$SESSION_A" "$KID_PATH/access?/add" "username=$USER_B" \
  'role=caregiver')"
[ "$code" = 200 ] || fail "4 grant caregiver" "status $code"
code="$(req home-b "$ADDR_B" "$SESSION_B" "$UI_URL/")"
[ "$code $(location home-b)" = "303 $KID_PATH" ] ||
  fail "4 $USER_B home" "status $code, Location $(location home-b)"
code="$(req kid-home-b "$ADDR_B" "$SESSION_B" "$UI_URL$KID_PATH")"
{ [ "$code" = 200 ] && body_has kid-home-b "$KID_NAME"; } || fail "4 $USER_B sees the kid" "status $code"
code="$(req access-b "$ADDR_B" "$SESSION_B" "$UI_URL$KID_PATH/access")"
{ [ "$code" = 404 ] && body_has access-b 'Not found' && ! body_has access-b "$KID_NAME"; } ||
  fail "4 caregiver access screen" "status $code"
pass "4 $USER_B, a caregiver, sees the kid and gets a 404 page on its access screen"

# 4a. smoke-a logs one entry of each tracker through the forms; each lands on the entry's timeline day.
N_BOTTLE_ML="$(note "bottle-ml-note-${SUFFIX}")"
N_BOTTLE_OZ="$(note "bottle-oz-note-${SUFFIX}")"
N_NURSING="$(note "nursing-note-${SUFFIX}")"
N_PUMPING="$(note "pumping-note-${SUFFIX}")"
N_DIAPER="$(note "diaper-note-${SUFFIX}")"
N_MEAL="$(note "meal-note-${SUFFIX}")"
N_SLEEP="$(note "sleep-note-${SUFFIX}")"
N_TIMER="$(note "nursing-timer-note-${SUFFIX}")"
N_DIAPER_EDIT="$(note "diaper-edited-note-${SUFFIX}")"
MEAL_WHAT="$(note "pear-${SUFFIX}")"
MEAL_REACTION="$(note "liked-${SUFFIX}")"
for v in 163 4.25 425 211; do remember_value "$v"; done
log_entry() { # name tracker field=value... -> as smoke-a, expects the 303 to the timeline
  local name="$1" tracker="$2" code
  shift 2
  code="$(form "$name" "$ADDR_A" "$SESSION_A" "$KID_PATH/log/$tracker?/save" "$@")"
  case "$code $(location "$name")" in
    "303 $KID_PATH/timeline?from="*) ;;
    *) fail "4a log $tracker" "status $code, Location $(location "$name")" ;;
  esac
}
log_entry log-bottle-ml bottles "at=$(ago 50)" contents=breast_milk contents=cow_milk volume=163 \
  volume_unit=ml "note=$N_BOTTLE_ML"
log_entry log-bottle-oz bottles "at=$(ago 45)" contents=formula volume=4.25 volume_unit=floz \
  "note=$N_BOTTLE_OZ"
log_entry log-nursing nursing "started_at=$(ago 120)" "ended_at=$(ago 100)" left_minutes=8 \
  right_minutes=7 "note=$N_NURSING"
log_entry log-pumping pumping "started_at=$(ago 90)" duration_minutes=12 volume=211 volume_unit=ml \
  side=both "note=$N_PUMPING"
log_entry log-diaper diapers at= kind=wet "note=$N_DIAPER"
log_entry log-meal meals "at=$(ago 25)" "what=$MEAL_WHAT" "reaction=$MEAL_REACTION" "note=$N_MEAL"
log_entry log-sleep sleep "started_at=$(ago 240)" "ended_at=$(ago 180)" "note=$N_SLEEP"
pass "4a $USER_A logs a bottle (two contents, ml), a bottle in fl oz, nursing, pumping, a diaper, a meal and a sleep"

# 4b. Timers from the home: one sleep at a time; a nursing timer switches side and ends.
code="$(form start-sleep "$ADDR_A" "$SESSION_A" "$KID_PATH?/startSleep")"
{ [ "$code" = 200 ] && body_has start-sleep 'Asleep since'; } || fail "4b start sleep" "status $code"
code="$(form sleep-again "$ADDR_A" "$SESSION_A" "$KID_PATH/log/sleep?/save" started_at= ended_at= note=)"
{ [ "$code" = 409 ] && body_has sleep-again 'A sleep timer is already running'; } ||
  fail "4b second sleep timer" "status $code"
code="$(form start-sleep-2 "$ADDR_A" "$SESSION_A" "$KID_PATH?/startSleep")"
{ [ "$code" = 200 ] && [ "$(count start-sleep-2 'Asleep since')" = 1 ]; } ||
  fail "4b second sleep tap" "status $code, $(count start-sleep-2 'Asleep since') running sleeps"
SLEEP_ID="$(timer_id start-sleep-2 endSleep)"
[ -n "$SLEEP_ID" ] || fail "4b sleep timer" "no End form"
code="$(form end-sleep "$ADDR_A" "$SESSION_A" "$KID_PATH?/endSleep" "id=$SLEEP_ID")"
{ [ "$code" = 200 ] && body_has end-sleep ">$KID_NAME</h1>" && ! body_has end-sleep 'Asleep since'; } ||
  fail "4b end sleep" "status $code"
log_entry log-nursing-timer nursing started_at= side=left ended_at= left_minutes= right_minutes= \
  "note=$N_TIMER"
code="$(req home-nursing "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
NURSING_ID="$(timer_id home-nursing switchSide)"
{ [ "$code" = 200 ] && body_has home-nursing 'Nursing, left since' && [ -n "$NURSING_ID" ]; } ||
  fail "4b nursing timer" "status $code"
code="$(form switch-side "$ADDR_A" "$SESSION_A" "$KID_PATH?/switchSide" "id=$NURSING_ID" side=right)"
{ [ "$code" = 200 ] && body_has switch-side 'Nursing, right since'; } || fail "4b switch side" "status $code"
code="$(form end-nursing "$ADDR_A" "$SESSION_A" "$KID_PATH?/endNursing" "id=$NURSING_ID")"
{ [ "$code" = 200 ] && body_has end-nursing ">$KID_NAME</h1>" &&
  ! body_has end-nursing 'Nursing, right since' &&
  ! body_has end-nursing 'Running now'; } ||
  fail "4b end nursing" "status $code"
pass "4b a sleep timer starts and ends from the home, a second one is refused (409), a nursing timer switches side and ends"

# 4c. The timeline of the kid's days shows every entry; the home shows today's card.
TIMELINE="$KID_PATH/timeline?from=$(ago 240 | cut -c1-10)&to=$(ago 0 | cut -c1-10)"
code="$(req timeline-a "$ADDR_A" "$SESSION_A" "$UI_URL$TIMELINE")"
[ "$code" = 200 ] || fail "4c timeline" "status $code"
for t in "$N_BOTTLE_ML" "$N_BOTTLE_OZ" "$N_NURSING" "$N_PUMPING" "$N_DIAPER" "$N_MEAL" "$N_SLEEP" \
  "$N_TIMER" "Solids: $MEAL_WHAT · $MEAL_REACTION" 'Bottle 163 ml, breast milk + cow' \
  'Bottle 126 ml, formula' 'Nursing L 8 min · R 7 min' 'Pumping 12 min, 211 ml (both)' \
  'Diaper wet' 'aria-label="Totals"'; do
  body_has timeline-a "$t" || fail "4c timeline" "missing '$(printf '%s' "$t" | scrub)'"
done
code="$(req home-today "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
{ [ "$code" = 200 ] && body_has home-today 'id="today-heading"' && body_has home-today 'Bottles' &&
  body_has home-today 'Diapers'; } || fail "4c today card" "status $code"
pass "4c the timeline shows every note, both bottle contents, fl oz in ml and the day totals; the home shows Today"

# 4d. The caregiver edits smoke-a's diaper and deletes the meal.
code="$(req timeline-b "$ADDR_B" "$SESSION_B" "$UI_URL$TIMELINE")"
DIAPER_PATH="$(row_path timeline-b "$N_DIAPER")"
MEAL_PATH="$(row_path timeline-b "$N_MEAL")"
{ [ "$code" = 200 ] && [ -n "$DIAPER_PATH" ] && [ -n "$MEAL_PATH" ]; } ||
  fail "4d caregiver timeline" "status $code, rows '$DIAPER_PATH' '$MEAL_PATH'"
code="$(form edit-diaper "$ADDR_B" "$SESSION_B" "$DIAPER_PATH?/save" at= kind=dirty "note=$N_DIAPER_EDIT")"
case "$code $(location edit-diaper)" in
  "303 $KID_PATH/timeline?from="*) ;;
  *) fail "4d caregiver edit" "status $code, Location $(location edit-diaper)" ;;
esac
code="$(form delete-meal "$ADDR_B" "$SESSION_B" "$MEAL_PATH?/delete")"
case "$code $(location delete-meal)" in
  "303 $KID_PATH/timeline?from="*) ;;
  *) fail "4d caregiver delete" "status $code, Location $(location delete-meal)" ;;
esac
code="$(req timeline-b2 "$ADDR_B" "$SESSION_B" "$UI_URL$TIMELINE")"
{ [ "$code" = 200 ] && body_has timeline-b2 "$N_DIAPER_EDIT" && body_has timeline-b2 'Diaper dirty' &&
  ! body_has timeline-b2 "$N_DIAPER" && ! body_has timeline-b2 "$N_MEAL" &&
  ! body_has timeline-b2 "$MEAL_WHAT" && body_has timeline-b2 "$N_BOTTLE_ML"; } ||
  fail "4d after edit and delete" "status $code"
pass "4d $USER_B, a caregiver, edits $USER_A's diaper and deletes the meal; the timeline shows both"

# 4e. A parent with no grant gets the 404 page on every new route; a caregiver on the modules page.
code="$(form admin-create-c "$ADDR_ADMIN" "$ADMIN_SESSION" '/admin/parents?/create' \
  "username=$USER_C" 'display_name=Smoke C')"
[ "$code" = 200 ] || fail "4e create $USER_C" "status $code"
SESSION_C="$(redeem redeem-c "$ADDR_C" "$(invite_url admin-create-c)" "$PASSWORD_C")"
for p in timeline:"$TIMELINE" log:"$KID_PATH/log/bottles" modules:"$KID_PATH/modules" \
  entry:"$DIAPER_PATH"; do
  code="$(req "nogrant-${p%%:*}" "$ADDR_C" "$SESSION_C" "$UI_URL${p#*:}")"
  { [ "$code" = 404 ] && body_has "nogrant-${p%%:*}" 'Not found' &&
    ! grep -qF -f "$NOTES_FILE" "$WORK/nogrant-${p%%:*}.body" &&
    ! body_has "nogrant-${p%%:*}" "$KID_NAME"; } || fail "4e no grant ${p%%:*}" "status $code"
done
code="$(req modules-b "$ADDR_B" "$SESSION_B" "$UI_URL$KID_PATH/modules")"
{ [ "$code" = 404 ] && body_has modules-b 'Not found' && ! body_has modules-b "$KID_NAME"; } ||
  fail "4e caregiver modules" "status $code"
pass "4e $USER_C, with no grant, gets the 404 page on the timeline, log, modules and entry pages; $USER_B on modules"

# 4f. Pumping switched off leaves the home and the default timeline, not the data; then back on.
code="$(req modules-a "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH/modules")"
mapfile -t MODULES_ON < <(modules_on modules-a)
{ [ "$code" = 200 ] && printf '%s\n' "${MODULES_ON[@]}" | grep -qx 'module=pumping' &&
  body_has home-today "$KID_PATH/log/pumping\""; } || fail "4f modules page" "status $code"
mapfile -t WITHOUT_PUMPING < <(printf '%s\n' "${MODULES_ON[@]}" | grep -vx 'module=pumping')
code="$(form modules-off "$ADDR_A" "$SESSION_A" "$KID_PATH/modules?/save" "${WITHOUT_PUMPING[@]}")"
{ [ "$code" = 200 ] && body_has modules-off 'Saved'; } || fail "4f pumping off" "status $code"
code="$(req home-off "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
code_t="$(req timeline-off "$ADDR_A" "$SESSION_A" "$UI_URL$TIMELINE")"
code_p="$(req timeline-pumping "$ADDR_A" "$SESSION_A" "$UI_URL$TIMELINE&types=pumping")"
{ [ "$code $code_t $code_p" = '200 200 200' ] && ! body_has home-off "/log/pumping\"" &&
  ! body_has timeline-off "$N_PUMPING" && body_has timeline-off "$N_BOTTLE_ML" &&
  body_has timeline-pumping "$N_PUMPING" && ! body_has timeline-pumping "$N_BOTTLE_ML"; } ||
  fail "4f pumping hidden" "home $code, timeline $code_t, types=pumping $code_p"
code="$(form modules-on "$ADDR_A" "$SESSION_A" "$KID_PATH/modules?/save" "${MODULES_ON[@]}")"
code_h="$(req home-on "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
{ [ "$code $code_h" = '200 200' ] && body_has home-on "$KID_PATH/log/pumping\""; } ||
  fail "4f pumping on" "save $code, home $code_h"
pass "4f pumping off: gone from the home and the default timeline, shown with types=pumping; on again"

# 4g. The install files are served without a session, before the app's hooks.
code="$(req manifest "$ADDR_C" - "$UI_URL/manifest.webmanifest")"
code_i="$(req icon "$ADDR_C" - "$UI_URL/icons/icon-192.png")"
{ [ "$code $code_i" = '200 200' ] &&
  grep -qi '^content-type: application/manifest+json' "$WORK/manifest.headers" &&
  grep -qi '^content-type: image/png' "$WORK/icon.headers"; } ||
  fail "4g install files" "manifest $code, icon $code_i"
pass "4g /manifest.webmanifest and /icons/icon-192.png answer 200 with their types and no cookie"

# 5. The last active guardian cannot be disabled until another guardian exists.
SOLE_GUARDIAN='This parent is the only guardian of at least one child. Another guardian must be added first.'
code="$(req admin-parents "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL/admin/parents")"
[ "$code" = 200 ] || fail "5 parents page" "status $code"
ID_A="$(parent_id admin-parents "$USER_A")"
[ -n "$ID_A" ] || fail "5 parents page" "no id for $USER_A"
code="$(form admin-disable-1 "$ADDR_ADMIN" "$ADMIN_SESSION" '/admin/parents?/disable' "parent_id=$ID_A")"
{ [ "$code" = 409 ] && body_has admin-disable-1 "$SOLE_GUARDIAN"; } ||
  fail "5 disable sole guardian" "status $code or the sole_guardian message missing"
code="$(form role-b "$ADDR_A" "$SESSION_A" "$KID_PATH/access?/role" "username=$USER_B" \
  'role=guardian')"
[ "$code" = 200 ] || fail "5 make $USER_B guardian" "status $code"
code="$(form admin-disable-2 "$ADDR_ADMIN" "$ADMIN_SESSION" '/admin/parents?/disable' "parent_id=$ID_A")"
[ "$code" = 200 ] || fail "5 disable after a second guardian" "status $code"
code="$(req admin-parents-after "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL/admin/parents")"
code_a="$(req home-a "$ADDR_A" "$SESSION_A" "$UI_URL/")"
{ [ "$code" = 200 ] && body_has admin-parents-after 'Disabled' && [ "$code_a" = 303 ] &&
  [[ "$(location home-a)" == /login* ]]; } ||
  fail "5 $USER_A disabled" "admin page $code, $USER_A home $code_a, Location $(location home-a)"
pass "5 disabling the sole guardian is refused; after a second guardian it succeeds"

# 6. The admin never sees a kid: pages, action answers, page data, and the kid's own URLs.
code="$(req admin-parents-data "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL/admin/parents/__data.json")"
[ "$code" = 200 ] || fail "6 admin page data" "status $code"
for p in home:/ kid:"$KID_PATH" kid-access:"$KID_PATH/access" kid-timeline:"$TIMELINE" \
  kid-log:"$KID_PATH/log/bottles" kid-modules:"$KID_PATH/modules" kid-entry:"$DIAPER_PATH"; do
  code="$(req "admin-${p%%:*}" "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL${p#*:}")"
  [ "$code $(location "admin-${p%%:*}")" = '303 /admin/parents' ] ||
    fail "6 admin GET ${p#*:}" "status $code, Location $(location "admin-${p%%:*}")"
done
code="$(req admin-kid-data "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL$KID_PATH/__data.json")"
{ [ "$code" = 200 ] && body_has admin-kid-data '"type":"redirect"' &&
  body_has admin-kid-data '/admin/parents'; } || fail "6 admin kid page data" "status $code"
leaks() { grep -qF -- "$KID_NAME" "$1" || grep -qF -f "$NOTES_FILE" "$1"; }
leaks "$WORK/kid-a.body" || fail "6 control" "the kid's name is not found in its own page"
grep -qF -- "$N_BOTTLE_ML" "$WORK/timeline-a.body" || fail "6 control" "a note is not found in the timeline"
n=0
for f in "$WORK"/admin-*.body "$WORK"/admin-*.headers; do
  n=$((n + 1))
  if leaks "$f"; then fail "6 admin sees no kid" "$(basename "$f")"; fi
done
pass "6 none of the admin's $n captured bodies and headers contain the kid's name or a note (controls match)"

API_MEM_1="$(mem "$API")"
UI_MEM_1="$(mem "$UI")"

# 7. Data and sessions live on the volume.
docker restart "$API" >/dev/null
wait_ok "$UI_URL/healthz/api" || fail "7 api restart" "no 200 within 30 s"
code="$(req kid-b "$ADDR_B" "$SESSION_B" "$UI_URL$KID_PATH")"
{ [ "$code" = 200 ] && body_has kid-b "$KID_NAME"; } || fail "7 session after restart" "status $code"
code="$(req admin-after-restart "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL/admin/parents")"
[ "$code" = 200 ] || fail "7 admin session after restart" "status $code"
pass "7 after an API restart the sessions of $USER_B and the admin still work"

# 8. New admin credentials end the old admin session (API no_session: cookie cleared).
API_MEM_1B="$(mem "$API")"
docker logs "$API" >"$WORK/api-1.log" 2>&1
docker rm -f "$API" >/dev/null
start_api "$ADMIN_PASSWORD_2"
wait_ok "$UI_URL/healthz/api" || fail "8 api with new password" "no 200 within 30 s"
code="$(req admin-old "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL/admin/parents")"
cleared="$(grep -ci '^set-cookie: hbt_session=;.*Max-Age=0' "$WORK/admin-old.headers" || true)"
case "$code $(location admin-old) $cleared" in
  "303 /login"*" 1") ;;
  *) fail "8 old admin cookie" "status $code, Location $(location admin-old), cleared $cleared" ;;
esac
NEW_ADMIN="$(login admin-login-2 "$ADDR_ADMIN" "$ADMIN_USERNAME" "$ADMIN_PASSWORD_2")"
[ -n "$NEW_ADMIN" ] || fail "8 new admin login"
pass "8 after the admin password changes the old admin cookie is refused and cleared"

# A parent login runs argon2 in the fresh API process before memory is read.
login login-b "$ADDR_B" "$USER_B" "$PASSWORD_B" >/dev/null

# Assertion 10's burst: each login has its own limiter keys, so every one reaches an argon2 verify.
BURST=24
for i in $(seq 1 "$BURST"); do
  pw="$(rand 12)"
  remember "burst-${SUFFIX}-$i"
  remember "$pw"
  form "burst-$i" "198.51.100.$i" - /login "username=burst-${SUFFIX}-$i" "password=$pw" \
    >"$WORK/burst-$i.code" &
done
api_burst=0
ui_burst=0
while [ -z "$MEM_FILE" ] && [ -n "$(jobs -r)" ]; do
  m="$(mem_mib "$API")"
  api_burst=$((m > api_burst ? m : api_burst))
  m="$(mem_mib "$UI")"
  ui_burst=$((m > ui_burst ? m : ui_burst))
done
wait
fail_if_oom "10 login burst"
if [ -n "$MEM_FILE" ]; then
  api_burst="$(mem "$API")"
  ui_burst="$(mem "$UI")"
fi
for i in $(seq 1 "$BURST"); do
  [ "$(cat "$WORK/burst-$i.code")" = 400 ] || fail "10 burst login $i" "status $(cat "$WORK/burst-$i.code")"
done

# 9. No identifier, password or token in either container's logs.
docker logs "$API" >"$WORK/api-2.log" 2>&1
docker logs "$UI" >"$WORK/ui.log" 2>&1
for f in api-1.log api-2.log ui.log; do
  [ -s "$WORK/$f" ] || fail "9 logs" "$f is empty"
  if grep -qF -f "$SECRETS_FILE" "$WORK/$f"; then fail "9 logs" "$f carries a secret or identifier"; fi
  if logged_value "$WORK/$f"; then fail "9 logs" "$f carries a volume figure"; fi
done
pass "9 the logs of both containers carry no username, kid name, password, token, note, meal text or volume"

# 10. Memory of each API process and of the UI, with the login burst in the last API process.
fail_if_oom "10 memory"
API_MEM_2="$(mem "$API")"
UI_MEM_2="$(mem "$UI")"
api_mem=0
for m in "$API_MEM_1" "$API_MEM_1B" "$api_burst" "$API_MEM_2"; do
  [ "$m" -gt 0 ] 2>/dev/null || fail "10 api memory" "unreadable reading '$m'"
  api_mem=$((m > api_mem ? m : api_mem))
done
ui_mem=0
for m in "$UI_MEM_1" "$ui_burst" "$UI_MEM_2"; do
  [ "$m" -gt 0 ] 2>/dev/null || fail "10 ui memory" "unreadable reading '$m'"
  ui_mem=$((m > ui_mem ? m : ui_mem))
done
if [ -n "$MEM_FILE" ]; then
  readings="$MEM_FILE: api ${API_MEM_1}/${API_MEM_1B}/${api_burst} MiB (first process before/after restart, last process after the ${BURST}-login burst), ui ${UI_MEM_2} MiB"
else
  readings="sampled docker stats: api ${API_MEM_1}/${API_MEM_1B}/${api_burst}/${API_MEM_2} MiB (first process before/after restart, during the ${BURST}-login burst, end), ui ${UI_MEM_1}/${ui_burst}/${UI_MEM_2} MiB"
fi
code="$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' "$UI_URL/healthz/api" || true)"
[ "$code" = 200 ] || fail "10 /healthz/api after the burst" "status $code; $readings"
[ "$api_mem" -le "$API_MEM_LIMIT_MIB" ] || fail "10 api memory over ${API_MEM_LIMIT_MIB} MiB" "$readings"
[ "$ui_mem" -le "$UI_MEM_LIMIT_MIB" ] || fail "10 ui memory over ${UI_MEM_LIMIT_MIB} MiB" "$readings"
pass "10 memory within ${API_MEM_LIMIT_MIB}/${UI_MEM_LIMIT_MIB} MiB, no OOM kill, /healthz/api 200: $readings"
