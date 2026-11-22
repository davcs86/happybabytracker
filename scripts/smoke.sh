#!/usr/bin/env bash
# Runs both images together and asserts the contract end to end, through the UI only.
# Needs Linux, GNU coreutils, bash >= 4.
set -euo pipefail

API_IMAGE="${1:?usage: smoke.sh <api-image> <ui-image>}"
UI_IMAGE="${2:?usage: smoke.sh <api-image> <ui-image>}"

API_MEM_LIMIT_MIB=60
UI_MEM_LIMIT_MIB=140
UI_PORT="${SMOKE_UI_PORT:-9212}"
REPO="$(cd "$(dirname "$0")/.." && pwd)"

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
# Tracker notes and texts (also in SECRETS_FILE), and volume and growth figures, matched on digit boundaries.
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

today() { TZ="$KID_ZONE" date '+%Y-%m-%d'; }

# The two 4a bottles are 163 ml (ago 50) and 4.25 fl oz = 126 ml (ago 45); either may fall on yesterday.
bottles_on() { # kid-local date -> what that day's Bottles row shows for them
  local n=0 ml=0
  if [ "$(ago 50 | cut -c1-10)" = "$1" ]; then n=$((n + 1)) ml=$((ml + 163)); fi
  if [ "$(ago 45 | cut -c1-10)" = "$1" ]; then n=$((n + 1)) ml=$((ml + 126)); fi
  if [ "$n" = 0 ]; then echo 'None yet'; else echo "$n · $ml ml"; fi
}

today_value() { # body-name label -> the value the home's Today card shows beside label
  pieces "$1" '<dt' | grep -F ">$2</dt>" | head -n 1 | sed -E 's|</dd>.*||; s|<[^>]*>| |g' |
    sed -E "s|^[[:space:]]*$2[[:space:]]*||; s|[[:space:]]+\$||" || true
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
logged_value() { # log-file -> succeeds when a remembered volume or growth figure is in it
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
# 126 is the 4.25 fl oz bottle in ml, 551 and 713 the ml bottle and pumping in hundredths of an ounce.
for v in 163 4.25 425 211 126 551 713; do remember_value "$v"; done
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
# to= is the day after NOW, so entries stamped after a midnight the run crosses still fall in the range.
TIMELINE="$KID_PATH/timeline?from=$(ago 240 | cut -c1-10)&to=$(ago -1440 | cut -c1-10)"
code="$(req timeline-a "$ADDR_A" "$SESSION_A" "$UI_URL$TIMELINE")"
[ "$code" = 200 ] || fail "4c timeline" "status $code"
for t in "$N_BOTTLE_ML" "$N_BOTTLE_OZ" "$N_NURSING" "$N_PUMPING" "$N_DIAPER" "$N_MEAL" "$N_SLEEP" \
  "$N_TIMER" "Solids: $MEAL_WHAT · $MEAL_REACTION" 'Bottle 163 ml, breast milk + cow' \
  'Bottle 126 ml, formula' 'Nursing L 8 min · R 7 min' 'Pumping 12 min, 211 ml (both)' \
  'Diaper wet' 'aria-label="Totals"' ">Bottles $(bottles_on "$(ago 50 | cut -c1-10)")<" \
  ">Bottles $(bottles_on "$(ago 45 | cut -c1-10)")<"; do
  body_has timeline-a "$t" || fail "4c timeline" "missing '$(printf '%s' "$t" | scrub)'"
done
code="$(req home-today "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
shown="$(today_value home-today Bottles)"
want="$(bottles_on "$(today)")"
{ [ "$code" = 200 ] && body_has home-today 'id="today-heading"' && [ "$shown" = "$want" ] &&
  body_has home-today 'Diapers'; } || fail "4c today card" "status $code, Bottles '$shown', want '$want'"
pass "4c the timeline shows every note, both bottle contents, fl oz in ml and the day totals; Today shows Bottles $shown"

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

# 4h. Growth through the real engine in the read-only image: scores follow the kid's source.
for f in who/LICENSE who/NOTICE cdc/NOTICE; do
  docker cp "$API:/usr/share/doc/happybabytracker-api/$f" "$WORK/notice" >/dev/null 2>&1 ||
    fail "4h notices" "the API image lacks $f"
  cmp -s "$WORK/notice" "$REPO/api/internal/growth/data/$f" || fail "4h notices" "$f differs from the repository"
  rm -f "$WORK/notice"
done
WHO_ATTRIBUTION='Geneva: World Health Organization; 2006. Licence: CC BY-NC-SA 3.0 IGO.'
CDC_ATTRIBUTION='Source: CDC, National Center for Health Statistics, 2000 CDC Growth Charts'
N_GROWTH="$(note "growth-note-${SUFFIX}")"
N_GROWTH_B="$(note "growth-caregiver-note-${SUFFIX}")"
N_GROWTH_EDIT="$(note "growth-edited-note-${SUFFIX}")"
# 3200 g = 11288 and 4100 g = 14462 hundredths of an ounce; 500 mm = 1969 and 340 mm = 1339 hundredths of an inch.
for v in 3.2 3200 11288 50 500 1969 4.1 4100 14462 34 340 1339; do remember_value "$v"; done
GROWTH="$KID_PATH/growth"
select_options() { # body-name select-name -> the option values of that select, one per line
  pieces "$1" '<select' | grep -F "name=\"$2\"" | sed 's|</select>.*||' |
    grep -oE '<option[^>]*value="[^"]*"' | sed -E 's/.*value="([^"]*)"/\1/' || true
}
growth_source() { # name source -> as smoke-a, saves the kid's settings with that growth source
  local code
  code="$(form "$1" "$ADDR_A" "$SESSION_A" "$KID_PATH/settings?/save" "name=$KID_NAME" \
    'birth_date=2026-01-15' 'sex=female' "time_zone=$KID_ZONE" "growth_source=$2")"
  { [ "$code" = 200 ] && body_has "$1" 'Saved'; } || fail "4h source $2" "status $code"
}
weight_line() { # body-name -> the weight line of the row that shows the growth note
  pieces "$1" '<li' | grep -F -- "$N_GROWTH" | grep -oE 'Weight 3\.2 kg: [^<]*' | head -n 1 || true
}
WEIGHT_SCORE_RE='^Weight 3\.2 kg: z -?[0-9]+\.[0-9]{2}, percentile [0-9]+\.[0-9]$'
originals() { # body-name -> name=value of every hidden *_original input, one per line
  grep -oE '<input type="hidden" name="[a-z_]+_original" value="[^"]*"' "$WORK/$1.body" |
    sed -E 's/.*name="([a-z_]+)" value="([^"]*)"/\1=\2/' || true
}
code="$(req settings-a "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH/settings")"
sources="$(select_options settings-a growth_source | tr '\n' ' ')"
{ [ "$code" = 200 ] && [ "$sources" = 'who cdc ' ]; } || fail "4h settings sources" "status $code, '$sources'"
code="$(form growth-new-a "$ADDR_A" "$SESSION_A" "$GROWTH/new?/save" measured_on=2026-01-15 \
  weight_unit=kg weight_kg=3.2 length_unit=cm length=50 head_unit=cm head=34 "note=$N_GROWTH")"
[ "$code $(location growth-new-a)" = "303 $GROWTH?measure=weight" ] ||
  fail "4h add measurement" "status $code, Location $(location growth-new-a)"
code="$(req growth-who "$ADDR_A" "$SESSION_A" "$UI_URL$GROWTH")"
line_who="$(weight_line growth-who)"
{ [ "$code" = 200 ] && [[ "$line_who" =~ $WEIGHT_SCORE_RE ]] &&
  body_has growth-who 'Length 50 cm: z ' && body_has growth-who 'Head circumference 34 cm: z ' &&
  body_has growth-who "$WHO_ATTRIBUTION"; } ||
  fail "4h growth page (who)" "status $code, weight line '$line_who'"
growth_source source-cdc cdc
code="$(req growth-cdc "$ADDR_A" "$SESSION_A" "$UI_URL$GROWTH")"
line_cdc="$(weight_line growth-cdc)"
{ [ "$code" = 200 ] && [ "${line_cdc##*, percentile }" != "${line_who##*, percentile }" ] &&
  [[ "$line_cdc" =~ $WEIGHT_SCORE_RE ]] && body_has growth-cdc "$CDC_ATTRIBUTION"; } ||
  fail "4h growth page (cdc)" "status $code, weight line '$line_cdc', with who '$line_who'"
growth_source source-who who
code="$(req growth-who-2 "$ADDR_A" "$SESSION_A" "$UI_URL$GROWTH")"
line_back="$(weight_line growth-who-2)"
{ [ "$code" = 200 ] && [ "$line_back" = "$line_who" ]; } ||
  fail "4h growth page (who again)" "status $code, weight line '$line_back', first '$line_who'"
code="$(form growth-new-b "$ADDR_B" "$SESSION_B" "$GROWTH/new?/save" measured_on=2026-02-01 \
  weight_unit=kg weight_kg=4.1 length_unit=cm length= head_unit=cm head= "note=$N_GROWTH_B")"
code_g="$(req growth-b "$ADDR_B" "$SESSION_B" "$UI_URL$GROWTH")"
{ [ "$code $(location growth-new-b)" = "303 $GROWTH?measure=weight" ] && [ "$code_g" = 200 ] &&
  body_has growth-b "$N_GROWTH_B" && body_has growth-b 'Weight 4.1 kg: z '; } ||
  fail "4h caregiver measurement" "status $code, Location $(location growth-new-b), page $code_g"
GROWTH_ENTRY="$(pieces growth-b '<li' | grep -F -- "$N_GROWTH" | grep -oE '/growth/[0-9a-f-]+"' |
  head -n 1 | tr -d '"' || true)"
[ -n "$GROWTH_ENTRY" ] || fail "4h measurement row" "no edit link"
GROWTH_ENTRY="$KID_PATH$GROWTH_ENTRY"
for p in growth:"$GROWTH" growth-new:"$GROWTH/new" growth-entry:"$GROWTH_ENTRY"; do
  code="$(req "nogrant-${p%%:*}" "$ADDR_C" "$SESSION_C" "$UI_URL${p#*:}")"
  { [ "$code" = 404 ] && body_has "nogrant-${p%%:*}" 'Not found' &&
    ! grep -qF -f "$NOTES_FILE" "$WORK/nogrant-${p%%:*}.body" &&
    ! body_has "nogrant-${p%%:*}" "$KID_NAME"; } || fail "4h no grant ${p%%:*}" "status $code"
  code="$(req "admin-${p%%:*}" "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL${p#*:}")"
  [ "$code $(location "admin-${p%%:*}")" = '303 /admin/parents' ] ||
    fail "4h admin GET ${p%%:*}" "status $code, Location $(location "admin-${p%%:*}")"
done
# The edit resends every shown text beside its original, clearing the head circumference.
code="$(req growth-edit-page "$ADDR_A" "$SESSION_A" "$UI_URL$GROWTH_ENTRY")"
mapfile -t kept < <(originals growth-edit-page)
{ [ "$code" = 200 ] && [ "${#kept[@]}" = 9 ]; } || fail "4h edit page" "status $code, ${#kept[@]} originals"
edit_fields=()
for kv in "${kept[@]}"; do
  shown="${kv%%=*}"
  shown="${shown%_original}"
  if [ "$shown" = head ]; then edit_fields+=("head="); else edit_fields+=("$shown=${kv#*=}"); fi
  edit_fields+=("$kv")
done
code="$(form growth-edit "$ADDR_A" "$SESSION_A" "$GROWTH_ENTRY?/save" "${edit_fields[@]}" "note=$N_GROWTH_EDIT")"
[ "$code $(location growth-edit)" = "303 $GROWTH?measure=weight" ] ||
  fail "4h edit measurement" "status $code, Location $(location growth-edit)"
code="$(req growth-edited "$ADDR_A" "$SESSION_A" "$UI_URL$GROWTH")"
{ [ "$code" = 200 ] && body_has growth-edited "$N_GROWTH_EDIT" && ! body_has growth-edited "$N_GROWTH" &&
  ! body_has growth-edited 'Head circumference 34 cm' && body_has growth-edited 'Length 50 cm: z '; } ||
  fail "4h after edit" "status $code"
GROWTH_ENTRY_B="$(pieces growth-b '<li' | grep -F -- "$N_GROWTH_B" | grep -oE '/growth/[0-9a-f-]+"' |
  head -n 1 | tr -d '"' || true)"
[ -n "$GROWTH_ENTRY_B" ] || fail "4h caregiver row" "no edit link"
code="$(form growth-delete "$ADDR_B" "$SESSION_B" "$KID_PATH$GROWTH_ENTRY_B?/delete")"
[ "$code $(location growth-delete)" = "303 $GROWTH?measure=weight" ] ||
  fail "4h delete measurement" "status $code, Location $(location growth-delete)"
code="$(req growth-deleted "$ADDR_B" "$SESSION_B" "$UI_URL$GROWTH")"
{ [ "$code" = 200 ] && ! body_has growth-deleted "$N_GROWTH_B" && ! body_has growth-deleted 'Weight 4.1 kg' &&
  body_has growth-deleted "$N_GROWTH_EDIT"; } || fail "4h after delete" "status $code"
pass "4h the API image ships the WHO and CDC notices; settings offer who and cdc; a measurement shows its z and percentile ('${line_who#*: }' with who, '${line_cdc#*: }' with cdc, back with who); $USER_B adds one; $USER_C gets 404 and the admin a 303 on the growth pages; $USER_A edits a measurement, clearing its head circumference; $USER_B deletes theirs"

# 4i. Vaccination through the real schedules: a 6-in-1 fills its lines once; a switch keeps the doses.
VACCINES="$KID_PATH/vaccines"
US_CAPTION='Schedule: Centers for Disease Control and Prevention (CDC), CDC Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025)'
US_ATTRIBUTION='Source: CDC, Recommended Child and Adolescent Immunization Schedule for Ages 18 Years or Younger, United States, 2025 (revised 07/02/2025). CDC does not endorse this application or its use of the material.'
UK_CAPTION='Schedule: UK Health Security Agency (UKHSA), Complete routine immunisation schedule from 1 September 2026'
HEXAVALENT=(antigens=diphtheria antigens=tetanus antigens=pertussis antigens=polio antigens=hib antigens=hepatitis_b)
HEXAVALENT_LINES='diphtheria-1 tetanus-1 pertussis-1 polio-1 hib-1 hepatitis_b-1'
DOSE_ON=2026-03-16
remember "$DOSE_ON"
V_LOT="$(note "lot-${SUFFIX}")"
V_LOT_EDIT="$(note "lot-edited-${SUFFIX}")"
V_LOT_B="$(note "lot-caregiver-${SUFFIX}")"
V_PRODUCT="$(note "product-${SUFFIX}")"
N_VACCINE="$(note "vaccine-note-${SUFFIX}")"
N_VACCINE_B="$(note "vaccine-caregiver-note-${SUFFIX}")"
vaccine_schedule() { # name schedule -> as smoke-a, saves the kid's settings with that schedule
  local code
  code="$(form "$1" "$ADDR_A" "$SESSION_A" "$KID_PATH/settings?/save" "name=$KID_NAME" \
    'birth_date=2026-01-15' 'sex=female' "time_zone=$KID_ZONE" growth_source=who "vaccine_schema=$2")"
  { [ "$code" = 200 ] && body_has "$1" 'Saved'; } || fail "4i schedule $2" "status $code"
}
line_status() { # body-name line-id -> the status badge of that schedule line
  grep -oE "id=\"vline-$2-status\"[^>]*>[^<]*" "$WORK/$1.body" | head -n 1 | sed 's/.*>//' || true
}
group_status() { # body-name from-date -> the status badge of that age group
  tr -d '\n' <"$WORK/$1.body" | grep -oE "id=\"vgroup-$2\"[^>]*>[^<]*<span[^>]*>[^<]*" | head -n 1 |
    sed 's/.*>//' || true
}
active_groups() { # body-name -> the from dates of the groups shown before Completed, in page order
  tr -d '\n' <"$WORK/$1.body" | sed 's|Completed (.*||' | grep -oE 'id="vgroup-[0-9-]{10}"' |
    sed -E 's/id="vgroup-(.*)"/\1/' || true
}
given_lines() { # body-name date -> how many lines and Other entries show a dose given that day
  echo $(($(count "$1" ">Given $2<") + $(count "$1" ", given $2<")))
}
administration() { # body-name line-id -> the edit path of the dose that fills that line
  local tail
  tail="$(pieces "$1" '<li' | grep -F "id=\"vline-$2-name\"" | grep -oE '/vaccines/[0-9a-f-]+"' |
    head -n 1 | tr -d '"' || true)"
  if [ -n "$tail" ]; then echo "$KID_PATH$tail"; fi
}
code="$(req settings-v "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH/settings")"
schedules="$(select_options settings-v vaccine_schema | tr '\n' ' ')"
{ [ "$code" = 200 ] && [ "$schedules" = ' us uk mx ' ] && body_has settings-v '>None yet</option>'; } ||
  fail "4i settings schedules" "status $code, '$schedules'"
vaccine_schedule schedule-us us
code="$(req vaccines-us "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
mapfile -t groups < <(active_groups vaccines-us)
{ [ "$code" = 200 ] && body_has vaccines-us "$US_CAPTION" && body_has vaccines-us "$US_ATTRIBUTION" &&
  body_has vaccines-us '>What is not included</summary>' &&
  body_has vaccines-us 'Tdap: 1 dose during each pregnancy' && [ "${#groups[@]}" -ge 2 ] &&
  printf '%s\n' "${groups[@]}" | sort -c && [ "${groups[0]}" = 2026-01-15 ] &&
  [ "$(group_status vaccines-us 2026-01-15)" = Overdue ] &&
  [ "$(line_status vaccines-us hepatitis_b-1)" = Overdue ] &&
  body_has vaccines-us 'Due from 2026-01-15 · overdue from 2026-02-15' &&
  [ "$(count vaccines-us 'action="?/dismiss"')" -gt 0 ]; } ||
  fail "4i vaccines page (us)" "status $code, groups '${groups[*]}', birth $(group_status vaccines-us 2026-01-15)"
code="$(req home-due "$ADDR_A" "$SESSION_A" "$UI_URL$KID_PATH")"
due_href="$(grep -oE '<a href="[^"]*"[^>]*>Birth · Overdue \(1\)</a>' "$WORK/home-due.body" |
  sed -E 's/<a href="([^"]*)".*/\1/' || true)"
{ [ "$code" = 200 ] && body_has home-due 'id="due-heading"' && [[ "$due_href" == *"$VACCINES" ]]; } ||
  fail "4i home Due" "status $code, link '$due_href'"
code="$(form vaccine-new-a "$ADDR_A" "$SESSION_A" "$VACCINES/new?/save" "given_on=$DOSE_ON" \
  product_shortcut=hexavalent "${HEXAVALENT[@]}" "lot=$V_LOT" "product=$V_PRODUCT" "note=$N_VACCINE")"
[ "$code $(location vaccine-new-a)" = "303 $VACCINES" ] ||
  fail "4i record a 6-in-1" "status $code, Location $(location vaccine-new-a)"
code="$(req vaccines-given "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
[ "$code" = 200 ] || fail "4i after the 6-in-1" "status $code"
for l in $HEXAVALENT_LINES; do
  [ "$(line_status vaccines-given "$l")" = Given ] || fail "4i 6-in-1 line $l" "$(line_status vaccines-given "$l")"
done
{ [ "$(given_lines vaccines-given "$DOSE_ON")" = 6 ] && [ "$(count vaccines-given ", given $DOSE_ON<")" = 0 ] &&
  [ "$(line_status vaccines-given hepatitis_b-2)" = Overdue ]; } ||
  fail "4i 6-in-1 listed once" "$(given_lines vaccines-given "$DOSE_ON") entries for the dose"
DOSE_A="$(administration vaccines-given diphtheria-1)"
[ -n "$DOSE_A" ] || fail "4i dose link" "no edit link on diphtheria-1"
code="$(req vaccine-edit-page "$ADDR_A" "$SESSION_A" "$UI_URL$DOSE_A")"
mapfile -t kept < <(originals vaccine-edit-page)
{ [ "$code" = 200 ] && [ "${#kept[@]}" = 5 ] && body_has vaccine-edit-page "value=\"$V_LOT\""; } ||
  fail "4i edit page" "status $code, ${#kept[@]} originals"
edit_fields=()
for kv in "${kept[@]}"; do
  shown="${kv%%=*}"
  shown="${shown%_original}"
  case "$shown" in
    antigens)
      IFS=, read -ra ticked <<<"${kv#*=}"
      for a in "${ticked[@]}"; do edit_fields+=("antigens=$a"); done
      ;;
    lot) edit_fields+=("lot=$V_LOT_EDIT") ;;
    *) edit_fields+=("$shown=${kv#*=}") ;;
  esac
  edit_fields+=("$kv")
done
code="$(form vaccine-edit "$ADDR_A" "$SESSION_A" "$DOSE_A?/save" "${edit_fields[@]}")"
code_e="$(req vaccine-edited "$ADDR_A" "$SESSION_A" "$UI_URL$DOSE_A")"
{ [ "$code $(location vaccine-edit)" = "303 $VACCINES" ] && [ "$code_e" = 200 ] &&
  body_has vaccine-edited "value=\"$V_LOT_EDIT\"" && body_has vaccine-edited "value=\"$V_PRODUCT\"" &&
  body_has vaccine-edited "value=\"$N_VACCINE\"" && [ "$(count vaccine-edited 'checked')" = 6 ]; } ||
  fail "4i edit lot" "status $code, Location $(location vaccine-edit), page $code_e"
code="$(form vaccine-delete "$ADDR_A" "$SESSION_A" "$DOSE_A?/delete")"
code_d="$(req vaccine-deleted "$ADDR_A" "$SESSION_A" "$UI_URL$DOSE_A")"
code_v="$(req vaccines-deleted "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
{ [ "$code $(location vaccine-delete)" = "303 $VACCINES" ] && [ "$code_d" = 404 ] && [ "$code_v" = 200 ] &&
  [ "$(given_lines vaccines-deleted "$DOSE_ON")" = 0 ] &&
  [ "$(line_status vaccines-deleted hepatitis_b-1)" = Overdue ]; } ||
  fail "4i delete dose" "status $code, Location $(location vaccine-delete), dose $code_d, page $code_v"
code="$(form vaccine-new-b "$ADDR_B" "$SESSION_B" "$VACCINES/new?/save" "given_on=$DOSE_ON" \
  product_shortcut=hexavalent "${HEXAVALENT[@]}" "lot=$V_LOT_B" product= "note=$N_VACCINE_B")"
code_v="$(req vaccines-b "$ADDR_B" "$SESSION_B" "$UI_URL$VACCINES")"
{ [ "$code $(location vaccine-new-b)" = "303 $VACCINES" ] && [ "$code_v" = 200 ] &&
  [ "$(given_lines vaccines-b "$DOSE_ON")" = 6 ] && [ "$(count vaccines-b 'action="?/dismiss"')" = 0 ]; } ||
  fail "4i caregiver records" "status $code, Location $(location vaccine-new-b), page $code_v"
DOSE_B="$(administration vaccines-b diphtheria-1)"
[ -n "$DOSE_B" ] || fail "4i caregiver dose link" "no edit link on diphtheria-1"
code="$(form dismiss-b "$ADDR_B" "$SESSION_B" "$VACCINES?/dismiss" line=rotavirus-1 reason=not_needed)"
{ [ "$code" = 403 ] && body_has dismiss-b 'Only a guardian can dismiss vaccines'; } ||
  fail "4i caregiver dismiss" "status $code"
code="$(form dismiss-a "$ADDR_A" "$SESSION_A" "$VACCINES?/dismiss" line=rotavirus-1 reason=not_needed)"
code_v="$(req vaccines-dismissed "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
{ [ "$code $(location dismiss-a)" = "303 $VACCINES" ] && [ "$code_v" = 200 ] &&
  [ "$(line_status vaccines-dismissed rotavirus-1)" = Dismissed ] &&
  body_has vaccines-dismissed 'Dismissed: not needed'; } || fail "4i dismiss" "status $code, page $code_v"
code="$(form undo-a "$ADDR_A" "$SESSION_A" "$VACCINES?/undoDismiss" line=rotavirus-1)"
code_v="$(req vaccines-undone "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
{ [ "$code $(location undo-a)" = "303 $VACCINES" ] && [ "$code_v" = 200 ] &&
  [ "$(line_status vaccines-undone rotavirus-1)" = Overdue ] &&
  ! body_has vaccines-undone 'Dismissed: '; } || fail "4i undo dismissal" "status $code, page $code_v"
code="$(form dismiss-before "$ADDR_A" "$SESSION_A" "$VACCINES?/dismissBefore" before=2026-03-01 \
  reason=given_unknown)"
code_v="$(req vaccines-before "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
{ [ "$code" = 200 ] && body_has dismiss-before 'Dismissed 1 line.' && [ "$code_v" = 200 ] &&
  [ "$(line_status vaccines-before hepatitis_b-2)" = Dismissed ] &&
  body_has vaccines-before 'Dismissed: given, date unknown'; } ||
  fail "4i dismiss before" "status $code, page $code_v, hepatitis_b-2 $(line_status vaccines-before hepatitis_b-2)"
vaccine_schedule schedule-uk uk
code="$(req vaccines-uk "$ADDR_A" "$SESSION_A" "$UI_URL$VACCINES")"
{ [ "$code" = 200 ] && body_has vaccines-uk "$UK_CAPTION" && ! body_has vaccines-uk "$US_CAPTION" &&
  [ "$(given_lines vaccines-uk "$DOSE_ON")" = 6 ] && ! body_has vaccines-uk 'Dismissed: ' &&
  [ "$(count vaccines-uk 'Dismissed<')" = 0 ] && [ -n "$(administration vaccines-uk diphtheria-1)" ]; } ||
  fail "4i switch to uk" "status $code, $(given_lines vaccines-uk "$DOSE_ON") entries for the dose"
for p in vaccines:"$VACCINES" vaccines-new:"$VACCINES/new" vaccine-dose:"$DOSE_B"; do
  code="$(req "nogrant-${p%%:*}" "$ADDR_C" "$SESSION_C" "$UI_URL${p#*:}")"
  { [ "$code" = 404 ] && body_has "nogrant-${p%%:*}" 'Not found' &&
    ! grep -qF -f "$NOTES_FILE" "$WORK/nogrant-${p%%:*}.body" &&
    ! body_has "nogrant-${p%%:*}" "$KID_NAME"; } || fail "4i no grant ${p%%:*}" "status $code"
  code="$(req "admin-${p%%:*}" "$ADDR_ADMIN" "$ADMIN_SESSION" "$UI_URL${p#*:}")"
  [ "$code $(location "admin-${p%%:*}")" = '303 /admin/parents' ] ||
    fail "4i admin GET ${p%%:*}" "status $code, Location $(location "admin-${p%%:*}")"
done
pass "4i settings offer us, uk and mx; the us page shows the CDC caption, attribution and omissions, groups in age order, birth overdue; the home shows Due; a 6-in-1 fills its 6 lines once, its lot is edited and it is deleted; $USER_B records one and may not dismiss; a dismissal is undone; dismiss-before dismisses 1 line; with uk the dose stays and the us dismissal is gone; $USER_C gets 404 and the admin a 303 on the vaccine pages"

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
code="$(form admin-log-bottle "$ADDR_ADMIN" "$ADMIN_SESSION" "$KID_PATH/log/bottles?/save" \
  contents=formula volume=90 volume_unit=ml note=)"
[ "$code $(location admin-log-bottle)" = '303 /admin/parents' ] ||
  fail "6 admin POST log/bottles" "status $code, Location $(location admin-log-bottle)"
code="$(req timeline-after-admin "$ADDR_B" "$SESSION_B" "$UI_URL$TIMELINE")"
{ [ "$code" = 200 ] && body_has timeline-after-admin "$N_BOTTLE_ML" &&
  ! body_has timeline-after-admin 'Bottle 90 ml'; } || fail "6 admin POST created an entry" "status $code"
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
pass "6 the admin's GETs and its bottle POST are 303s; none of its $n captured bodies and headers contain the kid's name or a note (controls match)"

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
printf '%s\n' '{"x":126}' >"$WORK/control-hit.log"
printf '%s\n' '{"duration_ms":126,"status":200}' >"$WORK/control-miss.log"
{ logged_value "$WORK/control-hit.log" && ! logged_value "$WORK/control-miss.log"; } ||
  fail "9 control" "logged_value misses a figure or matches a measured one"
docker logs "$API" >"$WORK/api-2.log" 2>&1
docker logs "$UI" >"$WORK/ui.log" 2>&1
for f in api-1.log api-2.log ui.log; do
  [ -s "$WORK/$f" ] || fail "9 logs" "$f is empty"
  if grep -qF -f "$SECRETS_FILE" "$WORK/$f"; then fail "9 logs" "$f carries a secret or identifier"; fi
  if logged_value "$WORK/$f"; then fail "9 logs" "$f carries a volume or measurement figure"; fi
done
pass "9 the logs of both containers carry no username, kid name, password, token, note, meal text, vaccine lot, product or dose date, volume or measurement"

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
