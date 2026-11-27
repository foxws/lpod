#!/usr/bin/env bash
# Runs lpod's commands against stand-ins for podman, systemctl, journalctl,
# git and the others it runs (tests/stubs), which answer from the host in
# tests/host, and checks what lpod prints, that it keeps the keys of the
# contract's fixtures, and how often it asks the host. Needs bash and jq...

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# The lpod to test: the bash script, or LPOD_UNDER_TEST, such as a Go build
# with lpod.sh next to it. The tests run it from another directory, so a
# relative path is made absolute...
LPOD="$(realpath -s "${LPOD_UNDER_TEST:-${ROOT}/bin/lpod}")"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

export PATH="${ROOT}/tests/stubs:${PATH}"
export LPOD_PODMAN_BINARY="${ROOT}/tests/stubs/podman"
export LPOD_QUADLET_BINARY="${ROOT}/tests/stubs/quadlet"
export HOME="${WORK}/home"
export XDG_CACHE_HOME="${WORK}/cache"
export XDG_CONFIG_HOME="${WORK}/config"
export LPOD_TEST_HOST="${WORK}/host"
export LPOD_TEST_LOG="${WORK}/calls.log"

OLD_VALKEY="a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
OUTPUT=""
TEST=""
PASSED=0
FAILED=0

# Starts the test NAME on a fresh copy of the host...
function test_case {
    TEST="$1"
    rm -rf "$LPOD_TEST_HOST" "$XDG_CACHE_HOME" "$XDG_CONFIG_HOME"
    mkdir -p "$HOME" "${XDG_CONFIG_HOME}/systemd/user"
    cp -r "${ROOT}/tests/host" "$LPOD_TEST_HOST"
    cp "${LPOD_TEST_HOST}/systemd/"* "${XDG_CONFIG_HOME}/systemd/user/"
    : > "$LPOD_TEST_LOG"
}

function pass {
    PASSED=$((PASSED + 1))
}

function fail {
    FAILED=$((FAILED + 1))
    echo "FAIL ${TEST}: $1" >&2
}

# Runs lpod with ARGS from an empty directory and keeps what it prints in
# OUTPUT...
function lpod {
    local STATUS=0
    OUTPUT="$(cd "$HOME" && "$LPOD" "$@" 2> "${WORK}/stderr")" || STATUS=$?

    if [ "$STATUS" -ne 0 ]; then
        fail "lpod $* exited with ${STATUS}: $(cat "${WORK}/stderr")"

        return 1
    fi
}

# Runs lpod like lpod does, but keeps its exit code in STATUS instead of
# failing the test, for commands that fail on purpose...
function lpod_status {
    STATUS=0
    OUTPUT="$(cd "$HOME" && "$LPOD" "$@" 2> "${WORK}/stderr")" || STATUS=$?
}

# Checks that the last lpod_status exited with CODE...
function exited_with {
    if [ "$STATUS" -eq "$1" ]; then
        pass
    else
        fail "expected exit code ${1}, got ${STATUS}: $(cat "${WORK}/stderr")"
    fi
}

# Checks that the jq FILTER is true for OUTPUT...
function check {
    if jq -e "$1" <<< "$OUTPUT" > /dev/null 2>&1; then
        pass
    else
        fail "expected ${1}, got: $(jq -c . <<< "$OUTPUT" 2>/dev/null || echo "$OUTPUT")"
    fi
}

# Checks that the stubs got the CALL (a whole line of the log) COUNT times
# since the test started...
function calls {
    local COUNT
    COUNT="$(grep -cxF -- "$1" "$LPOD_TEST_LOG")"

    if [ "$COUNT" -eq "$2" ]; then
        pass
    else
        fail "expected ${2} of \"${1}\", got ${COUNT}"
    fi
}

# Checks that COUNT calls since the test started match the extended regular
# expression PATTERN, such as none that pass an option where a name goes...
function calls_like {
    local COUNT
    COUNT="$(grep -cE -- "$1" "$LPOD_TEST_LOG")"

    if [ "$COUNT" -eq "$2" ]; then
        pass
    else
        fail "expected ${2} calls like \"${1}\", got ${COUNT}: $(grep -E -- "$1" "$LPOD_TEST_LOG" | head -n 3)"
    fi
}

# Checks that the command after MESSAGE succeeds, and fails the test with
# MESSAGE when it doesn't...
function holds {
    local MESSAGE="$1"
    shift

    if "$@"; then
        pass
    else
        fail "$MESSAGE"
    fi
}

# Checks that the last lpod_status printed TEXT to stderr...
function said {
    if grep -qF -- "$1" "${WORK}/stderr"; then
        pass
    else
        fail "expected \"${1}\" on stderr, got: $(cat "${WORK}/stderr")"
    fi
}

# Installs the Quadlet NAME with CONTENT on the test host, as part of the
# application APP ("" for a lone Quadlet), and generating UNIT, which is
# NAME's own unit unless it's given...
function add_quadlet {
    local NAME="$1" APP="$2" CONTENT="$3" UNIT="${4:-${1%.*}.service}"

    printf '%s\n' "$CONTENT" > "${LPOD_TEST_HOST}/quadlets/${NAME}"
    jq --arg name "$NAME" --arg app "$APP" --arg unit "$UNIT" \
        '. + [{Name: $name, UnitName: $unit, Path: "@QUADLETS@/\($name)", Status: "inactive/dead", App: $app}]' \
        "${LPOD_TEST_HOST}/quadlet-list.json" > "${WORK}/quadlet-list.json"
    mv "${WORK}/quadlet-list.json" "${LPOD_TEST_HOST}/quadlet-list.json"
}

# Forgets the calls so far, to count those of the next command only...
function forget_calls {
    : > "$LPOD_TEST_LOG"
}

# Checks that the first item of the jq PATH in OUTPUT has every key of the
# same item in the contract's FIXTURE...
function keeps_contract {
    local FIXTURE="$1" PATH_FILTER="$2"

    if jq -e -n \
        --slurpfile output <(printf '%s' "$OUTPUT") \
        --slurpfile fixture "${ROOT}/tests/fixtures/contract-1/${FIXTURE}.json" \
        "(\$fixture[0] | ${PATH_FILTER} | keys) - (\$output[0] | ${PATH_FILTER} | keys) == []" > /dev/null; then
        pass
    else
        fail "lacks keys of ${FIXTURE}.json at ${PATH_FILTER}"
    fi
}

PODMAN_STATS="podman stats --no-stream --format {{json .}}"
PODMAN_PS="podman ps --all --format json"
QUADLET_LIST="podman quadlet list --format json"

test_case "auto-update --dry-run keeps the image each pending container runs"
lpod auto-update --dry-run --json && check '.ok'
OUTPUT="$(cat "${XDG_CACHE_HOME}/lpod/auto-update.json")"
check "first(.[] | select(.ContainerName == \"stry-valkey\")).ImageID == \"${OLD_VALKEY}\""
check 'first(.[] | select(.ContainerName == "gone")).ImageID == null'
check 'first(.[] | select(.ContainerName == "stry-app")).ImageID == null'

test_case "images shows an update until the image is pulled"
lpod auto-update --dry-run --json
lpod images --json
check 'first(.images[] | select(.repository == "docker.io/library/valkey")) | .update == "9" and .used_by == ["stry"] and .containers == 2'
check 'first(.images[] | select(.repository == "ghcr.io/francoism90/stry")).update == null'
keeps_contract images '.images[0]'
lpod image pull docker.io/library/valkey:9 --json
lpod images --json
check '[.images[] | select(.update != null)] == []'
check "first(.images[] | select(.id == \"${OLD_VALKEY:0:12}\")).dangling"

test_case "images ignores the updates of an earlier lpod, which have no image ID"
mkdir -p "${XDG_CACHE_HOME}/lpod"
jq 'map(del(.ImageID))' "${LPOD_TEST_HOST}/auto-update.json" > "${XDG_CACHE_HOME}/lpod/auto-update.json"
lpod images --json
check '[.images[] | select(.update != null)] == []'

test_case "images reads the host once, without stats"
lpod images --json
calls "$QUADLET_LIST" 1
calls "$PODMAN_PS" 1
calls "podman ps --all --quiet" 0
calls "$PODMAN_STATS" 0

test_case "volumes"
lpod volumes --json
check 'first(.volumes[] | select(.name == "stry-data")) | .quadlet == "stry-data.volume" and .applications == ["stry"] and .containers == ["stry-app"]'
check 'first(.volumes[] | select(.name == "f00dcafe")) | .type == "anonymous" and .applications == ["stry"]'
check 'first(.volumes[] | select(.name == "/srv/media")) | .type == "bind" and .applications == ["stry"]'
keeps_contract volumes '.volumes[0]'
calls "$QUADLET_LIST" 1
calls "$PODMAN_PS" 1
calls "$PODMAN_STATS" 0

test_case "networks"
lpod networks --json
check 'first(.networks[] | select(.name == "stry")) | .quadlet == "stry.network" and .applications == ["stry"] and .containers == 2'
check 'first(.networks[] | select(.name == "podman")) | .default and .applications == ["glance"] and .containers == 2'
keeps_contract networks '.networks[0]'
calls "$QUADLET_LIST" 1
calls "$PODMAN_PS" 1
calls "$PODMAN_STATS" 0

test_case "containers"
lpod containers --json
check 'first(.containers[] | select(.name == "stry-app")) | .application == "stry" and .cpu == 2.5 and .memory == 300000000 and .health == "healthy"'
check 'first(.containers[] | select(.name == "systemd-glance")) | .application == "glance" and .cpu == null and .exit_code == 1'
check 'first(.containers[] | select(.name == "scratch")) | .application == null and .unit == null'
keeps_contract containers '.containers[0]'
calls "$PODMAN_STATS" 1

test_case "container NAME inspect shows stats, and its actions don't"
lpod container stry-app inspect --json
check '.container | .cpu == 2.5 and .quadlet == "stry-app.container" and .restarts == 0 and .auto_update == "registry"'
calls "$QUADLET_LIST" 1
calls "$PODMAN_STATS" 1
forget_calls
lpod container stry-valkey restart --json
calls "$PODMAN_STATS" 0

test_case "apps"
lpod apps --json
check 'first(.applications[] | select(.name == "stry")) | .status == "running" and .kind == "container" and .memory == 320000000 and .containers == 2 and .ports == ["0.0.0.0:8000->8080/tcp"]'
check 'first(.applications[] | select(.name == "glance")) | .status == "failed" and .error == "Error: config: glance.yml: no such file or directory"'
keeps_contract apps '.applications[0]'
calls "$QUADLET_LIST" 1
calls "$PODMAN_PS" 1
calls "$PODMAN_STATS" 1

test_case "APP status"
lpod stry status --json
check '.application | .name == "stry" and (.items | length) == 2 and first(.items[] | select(.name == "stry-valkey")).cpu == 0.5 and .volumes == [{"name": "stry-data", "size": null}]'
keeps_contract app-status '.application'
calls "$QUADLET_LIST" 1
calls "$PODMAN_PS" 1
lpod glance status --json
check '.application | .status == "failed" and .error == "Error: config: glance.yml: no such file or directory"'

test_case "times and ports of the listings"
lpod images --json
check 'first(.images[] | select(.repository == "ghcr.io/francoism90/stry")).created_at == "2026-11-18T11:06:40+00:00"'
lpod containers --json
check 'first(.containers[] | select(.name == "stry-app")).started_at == "2026-11-18T11:06:40+00:00"'
check 'first(.containers[] | select(.name == "systemd-glance")).started_at == null'
check 'first(.containers[] | select(.name == "scratch")) | .started_at == null and .pod == "scratch-pod"'
check '[.containers[].name] | index("5555-infra") == null'
lpod container stry-app inspect --json
check '.container.created_at == "2026-11-26T10:00:00Z"'
lpod container scratch inspect --json
check '.container.created_at == null'
lpod apps --json
check 'first(.applications[] | select(.name == "stry")).since == "2026-11-18T11:06:40+00:00"'
lpod pods --json
check '.pods == [{"id": "555555555555", "name": "scratch-pod", "status": "degraded", "application": null, "quadlet": null, "unit": null, "containers": 1, "running": 0, "ports": ["127.0.0.1:9000->80/tcp"], "networks": ["podman"], "created_at": "2026-11-17T10:00:00Z"}]'

test_case "journal and events"
lpod journal --json
check '.entries[0] == {"time": "2026-11-18T11:06:40+00:00", "unit": "glance", "priority": 3, "message": "Error: config: glance.yml: no such file or directory"}'
check '.entries[1] | .time == "2026-11-18T12:06:40+00:00" and .message == "Started"'
lpod events --json
check '[.events[] | {time, action, application}] == [
    {"time": "2026-11-18T11:06:40+00:00", "action": "died", "application": "glance"},
    {"time": "2026-11-18T08:20:00+00:00", "action": "pull", "application": null}
]'
check '.events[0].exit_code == 1'

test_case "systemd timers"
lpod settings --json
check '.settings.auto_update == {"enabled": true, "schedule": "*-*-* 03:00:00", "rollback": true, "next_run": "2026-11-18T12:06:40+00:00", "last_run": null}'
lpod stry timers --json
check '.timers[0] | .name == "backup" and .enabled and .running and .next_run == "2026-11-18T12:06:40+00:00" and .last_run == "2026-11-18T08:20:00+00:00" and .last_result == null'
lpod host --json
check '.auto_update == {"enabled": true, "next_run": "2026-11-18T12:06:40+00:00"}'
check '.system.booted_at | test("^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\\+00:00$")'
check '.units == {"active": 2, "failed": 1}'

# The cgroup checks read the real /sys of the machine the tests run on, so
# only their presence is checked...
mkdir -p "${WORK}/runtime"
export XDG_RUNTIME_DIR="${WORK}/runtime"

test_case "doctor"
lpod_status doctor --json
exited_with 0
check '["podman", "cgroups", "storage", "jq", "git", "systemd", "failed-services"] - [.checks[].id] == []'
check 'first(.checks[] | select(.id == "podman")) == {"id": "podman", "status": "ok", "message": "Podman 6.2.0 with Quadlet", "fix": ""}'
check 'first(.checks[] | select(.id == "storage")) | .status == "ok" and .message == "Podman stores images with the overlay driver"'
check 'first(.checks[] | select(.id == "failed-services")) | .status == "ok" and .message == "No applications'"'"' services have failed (1 other unit(s) failed: app-nvidia\\x2dsettings\\x2dload@autostart.service)"'
check '[.checks[] | select(.id == "proxy-certificate")] == []'

if [ "$(id -u)" -ne 0 ]; then
    check 'first(.checks[] | select(.id == "linger")).status == "ok"'
    check "first(.checks[] | select(.id == \"runtime-dir\")) | .status == \"ok\" and .message == \"XDG_RUNTIME_DIR is ${XDG_RUNTIME_DIR}\""
    check 'first(.checks[] | select(.id == "user-bus")) | .status == "warn" and (.fix | contains("dbus-user-session"))'
fi

lpod_status doctor
exited_with 0
if grep -q "✓ Podman 6.2.0 with Quadlet" <<< "$OUTPUT"; then pass; else fail "doctor's text output lacks the Podman check: ${OUTPUT}"; fi

test_case "doctor finds problems"
printf 'glance.service\nstry-ondemand.socket\napp-nvidia\\x2dsettings\\x2dload@autostart.service\n' > "${LPOD_TEST_HOST}/failed-units"
echo vfs > "${LPOD_TEST_HOST}/storage-driver"
echo no > "${LPOD_TEST_HOST}/linger"
lpod_status doctor --json
check 'first(.checks[] | select(.id == "failed-services")) | .status == "warn" and .message == "These applications'"'"' services have failed: glance.service stry-ondemand.socket (1 other unit(s) failed: app-nvidia\\x2dsettings\\x2dload@autostart.service)"'
check 'first(.checks[] | select(.id == "storage")) | .status == "warn" and (.fix | contains("podman system reset"))'

if [ "$(id -u)" -ne 0 ]; then
    check 'first(.checks[] | select(.id == "linger")) | .status == "warn" and (.fix | startswith("loginctl enable-linger"))'
    XDG_RUNTIME_DIR="" lpod_status doctor --json
    exited_with 1
    check 'first(.checks[] | select(.id == "runtime-dir")).status == "fail" and ([.checks[] | select(.id == "user-bus")] == []) and .failures >= 1'
fi

test_case "image pull pulls the reference, and a failed pull fails with podman's error"
lpod image pull docker.io/library/valkey:9 --json && check '.ok'
calls "podman pull --quiet docker.io/library/valkey:9" 1
lpod_status image pull docker.io/library/missing:1 --json
exited_with 1
said "manifest unknown"
holds "a failed pull printed JSON" test -z "$OUTPUT"

test_case "image rm removes an image by its ID, and fails while a container uses it"
UNUSED="e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5"
jq --arg id "$UNUSED" '. + [{Id: $id, Names: ["docker.io/library/alpine:3"], Containers: 0, Size: 8000000}]' \
    "${LPOD_TEST_HOST}/images.json" > "${WORK}/images.json" && mv "${WORK}/images.json" "${LPOD_TEST_HOST}/images.json"
lpod image rm "${UNUSED:0:12}" --json && check '.ok'
calls "podman image rm ${UNUSED:0:12}" 1
lpod_status image rm "${OLD_VALKEY:0:12}" --json
exited_with 1
said "image used by container"

test_case "images prune reports the short IDs it removed and the bytes of the unused images"
jq --arg id "$UNUSED" '. + [
    {Id: $id, Names: ["docker.io/library/alpine:3"], Containers: 0, Size: 8000000},
    {Id: ("f6" * 32), Names: ["docker.io/library/busybox:1"], Containers: 0, Size: 4000000},
    {Id: ("f7" * 32), Names: ["docker.io/library/debian:13"], Containers: 0, Size: 50000000, ReadOnly: true}
]' "${LPOD_TEST_HOST}/images.json" > "${WORK}/images.json" && mv "${WORK}/images.json" "${LPOD_TEST_HOST}/images.json"
printf '%s\n' "$UNUSED" "$(printf 'f6%.0s' {1..32})" > "${LPOD_TEST_HOST}/pruned-image"
lpod images prune --json
check '.removed == ["e5e5e5e5e5e5", "f6f6f6f6f6f6"] and .reclaimed == 12000000'
keeps_contract image-prune '.'
calls "podman image prune --all --force" 1

test_case "containers, volumes and networks prune report what they removed"
printf '%s\n' "$(printf 'ab%.0s' {1..32})" > "${LPOD_TEST_HOST}/pruned-container"
printf '%s\n' stry-cache "$(printf '9c%.0s' {1..32})" > "${LPOD_TEST_HOST}/pruned-volume"
printf 'iot\n' > "${LPOD_TEST_HOST}/pruned-network"
lpod containers prune --json
check '. == {"contract": 1, "removed": ["abababababab"], "reclaimed": null}'
keeps_contract container-prune '.'
lpod volumes prune --json
check '.removed == ["stry-cache", "9c9c9c9c9c9c"] and .reclaimed == null'
keeps_contract volume-prune '.'
lpod networks prune --json
check '. == {"contract": 1, "removed": ["iot"], "reclaimed": null}'
keeps_contract network-prune '.'
calls_like "^podman (container|volume|network) prune --force$" 3

test_case "a prune with nothing to remove reports an empty list"
lpod images prune --json
check '.removed == [] and .reclaimed == 0'
lpod networks prune --json
check '.removed == []'

test_case "a prune fails when podman does"
touch "${LPOD_TEST_HOST}/prune-fails"

for KIND in images containers volumes networks; do
    lpod_status "$KIND" prune --json
    exited_with 1
    said "database is locked"
    holds "a failed ${KIND} prune printed JSON" test -z "$OUTPUT"
done

test_case "container NAME start|stop|restart goes through the container's unit, without waiting"

for ACTION in start stop restart; do
    forget_calls
    lpod container stry-app "$ACTION" --json && check '.ok'
    calls "systemctl --user --no-block ${ACTION} stry-app.service" 1
    calls_like "^podman (start|stop|restart)" 0
done

test_case "container NAME start|stop|restart runs podman for a container without a unit, by name or ID"
lpod container scratch start --json && check '.ok'
calls "podman start scratch" 1
lpod container 444444444444 stop --json && check '.ok'
calls "podman stop 444444444444" 1
calls_like "^systemctl .*(start|stop|restart)" 0

test_case "container NAME refuses an unknown container or action"
lpod_status container ghost start --json
exited_with 1
said "No container named ghost."
lpod_status container stry-app pause --json
exited_with 1
said "Usage: lpod container NAME"
calls_like "^(podman|systemctl .*) (start|stop|restart|pause)" 0

test_case "container NAME exec reports the command's exit code and output, and fails only on podman's own errors"
lpod container stry-app exec --json -- sh -c 'echo out; echo err >&2; exit 3'
check '. == {"contract": 1, "exit_code": 3, "stdout": "out\n", "stderr": "err\n"}'
keeps_contract container-exec '.'
lpod container stry-app exec --user www-data --workdir /app --json -- true && check '.exit_code == 0'
calls "podman exec --user www-data --workdir /app stry-app true" 1
lpod container stry-app exec --json -- sh -c 'head -c 70000 /dev/zero | tr "\0" a'
check '(.stdout | length) == 65536'
lpod_status container stry-app exec --user ghost --json -- true
exited_with 1
said "unable to find user ghost"
holds "a failed exec printed JSON" test -z "$OUTPUT"

test_case "container NAME exec only passes stdin on with --stdin"
lpod container stry-app exec --stdin --json -- cat < <(printf 'p4ss')
check '.stdout == "p4ss"'
calls "podman exec --interactive stry-app cat" 1
calls_like "p4ss" 0
lpod container stry-app exec --json -- cat < <(printf 'p4ss')
check '.stdout == ""'
calls "podman exec: stdin is open" 0

test_case "container NAME exec refuses a stopped container, a missing command and unknown options"
lpod_status container scratch exec --json -- true
exited_with 1
said "scratch isn't running"
lpod_status container stry-app exec --json
exited_with 1
said "Usage: lpod container NAME exec"
lpod_status container stry-app exec --tty --json -- true
exited_with 1
said "Unknown option: --tty"
calls_like "^podman exec" 0

test_case "APP up and restart act on the main unit, and down stops every service"
lpod stry up --json && check '.ok'
calls "systemctl --user start stry-app.service" 1
lpod stry restart --no-block --json && check '.ok'
calls "systemctl --user --no-block restart stry-app.service" 1
lpod stry down --json && check '.ok'
calls "systemctl --user stop stry-app.service stry-valkey.service" 1
lpod stry down --no-block --json && check '.ok'
calls "systemctl --user --no-block stop stry-app.service stry-valkey.service" 1

test_case "APP down --force kills every service, and clears its failed state"
lpod stry down --force --json && check '.ok'
calls "systemctl --user --no-block stop stry-app.service stry-valkey.service" 1
calls "systemctl --user kill --signal=SIGKILL stry-app.service stry-valkey.service" 1
calls "systemctl --user --quiet is-active stry-app.service stry-valkey.service" 1
calls "systemctl --user reset-failed stry-app.service stry-valkey.service" 1
lpod_status stry up --force --json
exited_with 1
said "--force only works with down."

test_case "APP up, down and redeploy refuse an application that isn't installed"

for ACTION in up down restart redeploy; do
    lpod_status ghost "$ACTION" --json
    exited_with 1
    said "No application or Quadlet named ghost is installed."
done

calls_like "^(systemctl .*(start|stop|restart)|podman pull)" 0

test_case "APP up and restart wake an on-demand application that sleeps"
printf '# lpod-ondemand: {"port": 8000}\n[Socket]\nListenStream=8000\n' > "${XDG_CONFIG_HOME}/systemd/user/stry-ondemand.socket"
lpod stry up --json && check '.ok'
calls "systemctl --user start stry-ondemand.service" 1
lpod stry restart --json && check '.ok'
calls "systemctl --user restart stry-app.service" 1
lpod stry down --json
forget_calls
lpod stry restart --json && check '.ok'
calls "systemctl --user start stry-ondemand.service" 1
calls_like "^systemctl .*(start|restart) stry-app.service" 0

test_case "APP redeploy pulls every image from a registry, then restarts the main unit"
lpod stry redeploy --json && check '.ok'
calls "podman pull --quiet -- ghcr.io/francoism90/stry:4" 1
calls "podman pull --quiet -- docker.io/library/valkey:9" 1
holds "redeploy didn't restart the main unit last" test "$(tail -n 1 "$LPOD_TEST_LOG")" == "systemctl --user restart stry-app.service"

test_case "APP redeploy skips images built on the host, and stops at a failed pull"
jq 'map(if .Names[0] == "stry-valkey" then .Image = "localhost/valkey:dev" else . end)' "${LPOD_TEST_HOST}/ps.json" > "${WORK}/ps.json"
mv "${WORK}/ps.json" "${LPOD_TEST_HOST}/ps.json"
lpod stry redeploy --json && check '.ok'
calls_like "^podman pull" 1
calls_like "^podman pull .*localhost" 0
jq 'map(if .Names[0] == "stry-valkey" then .Image = "docker.io/library/missing:1" else . end)' "${LPOD_TEST_HOST}/ps.json" > "${WORK}/ps.json"
mv "${WORK}/ps.json" "${LPOD_TEST_HOST}/ps.json"
forget_calls
lpod_status stry redeploy --json
exited_with 1
said "manifest unknown"
calls_like "^systemctl .*restart" 0

test_case "APP redeploy --no-block hands the work to a transient unit"
lpod stry redeploy --no-block --json && check '.ok'
calls_like "^systemd-run --quiet --collect --no-block --user --unit lpod-redeploy-stry /.+ stry redeploy --json$" 1
calls_like "^(podman pull|systemctl .*restart)" 0

HELLO='{"files": [
    {"name": "hello.container", "content": "[Container]\nImage=docker.io/library/hello-world\nVolume=hello-data.volume:/data\n"},
    {"name": "hello-data.volume", "content": "[Volume]\n"}
]}'
INSTALLED="${XDG_CONFIG_HOME}/containers/systemd"

test_case "quadlet check lists the units the files generate, next to the installed Quadlets"
mkdir -p "$INSTALLED"
printf '[Container]\nImagee=docker.io/library/broken\n' > "${INSTALLED}/broken.container"
printf '[Container]\nImage=docker.io/library/valkey:9\n' > "${INSTALLED}/stry-valkey.container"
lpod quadlet check --stdin --json <<< "$HELLO"
check '. == {"contract": 1, "valid": true, "errors": [], "units": ["hello.service", "hello-data-volume.service"]}'
keeps_contract quadlet-check '.'
calls "quadlet -dryrun -user" 1
holds "quadlet check left its files with the installed Quadlets" test ! -e "${INSTALLED}/hello.container"

test_case "quadlet check finds the installed Quadlets the files refer to"
mkdir -p "$INSTALLED"
printf '[Network]\n' > "${INSTALLED}/stry.network"
lpod quadlet check --stdin --json <<< '{"files": [{"name": "hello.container", "content": "[Container]\nImage=docker.io/library/hello-world\nNetwork=stry.network\n"}]}'
check '.valid and .units == ["hello.service"]'
lpod quadlet check --stdin --json <<< '{"files": [{"name": "hello.container", "content": "[Container]\nImage=docker.io/library/hello-world\nNetwork=gone.network\n"}]}'
check '.valid == false and .errors == ["hello.container: requested Quadlet gone.network was not found"]'

test_case "quadlet check reports the generator's errors about the files, without the lines that load them"
lpod quadlet check --stdin --json <<< '{"files": [{"name": "hello.container", "content": "[Container]\nImagee=docker.io/library/hello-world\n"}]}'
check '. == {"contract": 1, "valid": false, "errors": ["hello.container: unsupported key '"'"'Imagee'"'"' in group '"'"'Container'"'"'"], "units": []}'
keeps_contract quadlet-check-invalid '.'

test_case "quadlet check and install refuse a file name that isn't a plain Quadlet's, before writing anything"

for NAME in ../hello.container hello.container.d/lpod-ondemand.conf .hidden.container hello.conf hello; do
    for COMMAND in "check" "install hello"; do
        # shellcheck disable=SC2086
        lpod_status quadlet $COMMAND --stdin --json < <(jq -n --arg name "$NAME" '{files: [{name: $name, content: "[Container]\n"}]}')
        exited_with 1
        said "Invalid Quadlet file name: ${NAME}."
    done
done

lpod_status quadlet check --stdin --json <<< '{"files": []}'
exited_with 1
said 'Pass {"files": [{"name": ..., "content": ...}]} on stdin.'
calls_like "^(quadlet|podman quadlet install)" 0

test_case "quadlet install installs the files as an application, byte for byte"
lpod quadlet install hello --stdin --json < <(jq -n '{files: [
    {name: "hello.container", content: "[Container]\nImage=docker.io/library/hello-world\nExec=sh -c '"'"'echo \"$HOME\" $(id)'"'"'"},
    {name: "hello.env", content: "GREETING=hi\n"}
]}') && check '.ok'
calls_like "^podman quadlet install --application=hello --replace=false /.+/hello$" 1
holds "hello.container changed on its way" cmp -s <(printf '%s' "[Container]
Image=docker.io/library/hello-world
Exec=sh -c 'echo \"\$HOME\" \$(id)'") "${LPOD_TEST_HOST}/quadlets/hello.container"
holds "hello.env wasn't installed" test -f "${LPOD_TEST_HOST}/quadlets/hello.env"
calls_like "^systemctl .*start" 0

test_case "quadlet install --start starts the application's units, without waiting"
lpod quadlet install hello --start --stdin --json < <(jq '.files += [
    {name: "hello.network", content: "[Network]\n"},
    {name: "hello.image", content: "[Image]\nImage=docker.io/library/hello-world\n"}
]' <<< "$HELLO") && check '.ok'
STARTED="$(sed -n 's/^systemctl --user --no-block start //p' "$LPOD_TEST_LOG" | tr ' ' '\n' | LC_ALL=C sort | tr '\n' ' ')"
holds "started ${STARTED:-nothing}" test "$STARTED" == "hello-data-volume.service hello-network.service hello.service "

test_case "quadlet install only replaces an application's files with --replace"
lpod quadlet install hello --stdin --json <<< "$HELLO"
lpod_status quadlet install hello --stdin --json <<< "$HELLO"
exited_with 1
said "a Quadlet with name hello"
lpod quadlet install hello --replace --stdin --json <<< "$HELLO" && check '.ok'
calls_like "^podman quadlet install --application=hello --replace=true " 1

test_case "quadlet install refuses an application name that's an option, a path or a Quadlet's, and files not on stdin"

for APPLICATION in -rf ../hello hello/world hello.container ""; do
    lpod_status quadlet install "$APPLICATION" --stdin --json <<< "$HELLO"
    exited_with 1
    said "Invalid application name: ${APPLICATION}."
done

lpod_status quadlet install hello --json <<< "$HELLO"
exited_with 1
said "--stdin"
lpod_status quadlet install hello --stdin --force --json <<< "$HELLO"
exited_with 1
said "Unknown option: --force"
calls_like "^podman quadlet install" 0

test_case "quadlet fetch returns a directory's files and the nearest pods.yml"
REPOSITORY="${LPOD_TEST_HOST}/repositories/catalog"
mkdir -p "${REPOSITORY}/nextcloud/config" "${REPOSITORY}/testing/nextcloud/next" "${REPOSITORY}/many" "${REPOSITORY}/big"
printf 'name: catalog\n' > "${REPOSITORY}/pods.yml"
printf 'name: nextcloud\n' > "${REPOSITORY}/nextcloud/pods.yml"
printf '[Pod]\n' > "${REPOSITORY}/nextcloud/nextcloud.pod"
printf '[Container]\nPod=nextcloud.pod\n' > "${REPOSITORY}/nextcloud/nextcloud-db.container"
printf '[Container]\nPod=nextcloud.pod\n' > "${REPOSITORY}/nextcloud/nextcloud-app.container"
printf 'SECRET=x\n' > "${REPOSITORY}/nextcloud/.env"
printf 'not a file of this directory\n' > "${REPOSITORY}/nextcloud/config/app.yml"
printf 'name: testing\n' > "${REPOSITORY}/testing/pods.yml"
printf '[Container]\n' > "${REPOSITORY}/testing/nextcloud/next/nextcloud.container"
head -c 70000 /dev/zero | tr '\0' '#' > "${REPOSITORY}/big/pods.yml"
head -c 300000 /dev/zero | tr '\0' '#' > "${REPOSITORY}/big/big.container"
printf '[Container]\n' > "${REPOSITORY}/big/small.container"

for I in $(seq -w 1 60); do
    printf '[Container]\n' > "${REPOSITORY}/many/app-${I}.container"
done

lpod quadlet fetch https://example.test/catalog.git --ref v2 --path nextcloud/ --json
check '.manifest == "name: nextcloud\n" and .manifest_path == "nextcloud"'
check '[.files[].name] == ["nextcloud-app.container", "nextcloud-db.container", "nextcloud.pod", "pods.yml"]'
check '.files[0].content == "[Container]\nPod=nextcloud.pod\n"'
keeps_contract quadlet-fetch '.'
keeps_contract quadlet-fetch '.files[0]'
calls_like "^git .*clone --quiet --depth 1 --branch v2 -- https://example.test/catalog.git " 1
lpod quadlet fetch https://example.test/catalog.git --ref v2 --path testing/nextcloud/next --json
check '.manifest == "name: testing\n" and .manifest_path == "testing" and [.files[].name] == ["nextcloud.container"]'
lpod quadlet fetch https://example.test/catalog.git --ref v2 --path many --json
check '.manifest == "name: catalog\n" and .manifest_path == "" and (.files | length) == 50 and .files[-1].name == "app-50.container"'
keeps_contract quadlet-fetch-manifest '.'
lpod quadlet fetch https://example.test/catalog.git --ref v2 --path big --json
# A pods.yml over 64 KiB isn't the manifest, but it's still one of the files...
check '.manifest == "name: catalog\n" and [.files[].name] == ["pods.yml", "small.container"]'

test_case "quadlet fetch without a pods.yml, and of a repository, branch or directory that isn't there"
mkdir -p "${LPOD_TEST_HOST}/repositories/bare/hello"
printf '[Container]\n' > "${LPOD_TEST_HOST}/repositories/bare/hello/hello.container"
lpod quadlet fetch https://example.test/bare.git --ref main --path hello --json
check '.manifest == null and .manifest_path == null and [.files[].name] == ["hello.container"]'
lpod quadlet fetch https://example.test/bare.git --ref main --json
check '.manifest == null and .files == []'
lpod_status quadlet fetch https://example.test/missing.git --ref main --json
exited_with 1
said "not found"
lpod_status quadlet fetch https://example.test/bare.git --ref main --path goodbye --json
exited_with 1
said "goodbye is not a directory in the repository."
lpod_status quadlet fetch https://example.test/bare.git --ref main --depth 1 --json
exited_with 1
said "Unknown option: --depth"

# Untrusted input (#53, #55): names, Quadlet values, repositories and .env
# files come from users and Git repositories...

test_case "names that could be options or paths are refused before podman runs"

for ARGS in "container -rf restart" "container ../stry-app inspect" "volume rm --all" "volume rm ../stry-data" \
    "volume export --all" "network rm -f" "secret rm --all" "secret create ../db --stdin" \
    "image pull --all-tags" "image pull ../x" "image rm --force" "image rm stry"; do
    forget_calls
    # shellcheck disable=SC2086
    lpod_status $ARGS --json < /dev/null
    exited_with 1
    calls_like "" 0
done

test_case "a secret's value only comes from stdin"
lpod_status secret create db-password --json
exited_with 1
said "--stdin"
calls_like "^podman secret" 0
lpod secret create db-password --stdin --json < <(printf 's3cret') && check '.ok'
calls "podman secret create --driver file db-password -" 1
calls_like "s3cret" 0
holds "podman didn't get the secret's value on stdin" grep -qxF s3cret "${LPOD_TEST_HOST}/secret-value"

test_case "a Secret= name in a Quadlet is refused before it becomes an argument"
add_quadlet evil.container "" $'[Container]\nImage=docker.io/library/alpine:3\nSecret=--all,type=env,target=TOKEN'
lpod_status evil secrets < /dev/null
exited_with 1
said "Invalid name: --all."
calls_like "^podman secret" 0

test_case "a ServiceName= that starts with - never reaches systemctl as an option"
add_quadlet evil.container "" $'[Container]\nImage=docker.io/library/alpine:3\n[Service]\nServiceName=--host=evil' "--host=evil.service"
lpod apps --json
check 'first(.applications[] | select(.name == "evil")).unit == "evil.service"'
lpod evil restart --json && check '.ok'
calls "systemctl --user restart evil.service" 1
calls_like "--host" 0

test_case "APP remove passes names after --, and only removes the plain ones no one else uses"
add_quadlet evil-app.container evil "$(printf '%s\n' '[Container]' \
    'Image=--all' 'Image=docker.io/library/valkey:9' 'Image=docker.io/library/alpine:3' \
    'Volume=-rf:/srv' 'Volume=evil-data:/data' 'Volume=stry-data.volume:/shared' \
    'Secret=--all,type=env' 'Secret=evil-key,type=env')"
lpod evil remove --volumes --images --secrets --json && check '.ok'
calls "podman quadlet rm --force --recursive -- evil" 1
calls "podman volume rm -- evil-data" 1
calls "podman image rm -- docker.io/library/alpine:3" 1
calls "podman secret rm -- evil-key" 1
calls_like "^podman (volume|image|secret) rm" 3
calls_like "^podman (volume|image|secret) rm -- -" 0

test_case "quadlet fetch refuses a URL, branch or directory that git would read as something else, before cloning"

for ARGS in "file:///etc --ref main" "/etc --ref main" "ext::sh --ref main" "--upload-pack=touch --ref main" \
    "https://example.test/apps.git --ref --upload-pack=touch" "https://example.test/apps.git --ref main..x" \
    "https://example.test/apps.git --ref main --path ../etc" "https://example.test/apps.git --ref main --path stry/../.." \
    "https://example.test/apps.git --ref main --path /etc"; do
    forget_calls
    # shellcheck disable=SC2086
    lpod_status quadlet fetch $ARGS --json
    exited_with 1
    calls_like "^git" 0
done

test_case "quadlet fetch never reads a file outside the repository, through a symlink or a name with a newline"
OUTSIDE="${WORK}/outside"
REPOSITORY="${LPOD_TEST_HOST}/repositories/apps"
rm -rf "$OUTSIDE"
mkdir -p "${OUTSIDE}/apps" "${REPOSITORY}/stry" "${REPOSITORY}/nested"
echo "the host's own file" > "${OUTSIDE}/secret"
echo "the host's own file" > "${OUTSIDE}/apps/secret.container"
printf 'name: root\n' > "${REPOSITORY}/pods.yml"
printf '[Container]\nImage=ghcr.io/francoism90/stry:4\n' > "${REPOSITORY}/stry/stry.container"
echo hidden > "${REPOSITORY}/stry/.env"
# Read a line at a time, this name would also be name.container in the
# working directory, the home directory over SSH...
echo newline > "${REPOSITORY}/stry/"$'evil\nname.container'
echo "the host's own file" > "${HOME}/name.container"
ln -s "${OUTSIDE}/secret" "${REPOSITORY}/stry/pods.yml"
ln -s "${OUTSIDE}/secret" "${REPOSITORY}/stry/linked.container"
ln -s "$OUTSIDE" "${REPOSITORY}/linked"
ln -s ../stry "${REPOSITORY}/nested/stry"

lpod quadlet fetch https://example.test/apps.git --ref main --path stry --json
check '.commit == "abc1234" and ([.files[].name] == ["stry.container"])'
check '.manifest == "name: root\n" and .manifest_path == ""'
calls_like "^git .*clone --quiet --depth 1 --branch main -- https://example.test/apps.git " 1

for DIRECTORY in linked linked/apps nested/stry; do
    lpod_status quadlet fetch https://example.test/apps.git --ref main --path "$DIRECTORY" --json
    exited_with 1
    said "${DIRECTORY} is not a directory in the repository."
done

test_case ".env is read as text, and only for the settings lpod uses"
cat > "${HOME}/.env" <<'EOF'
$(touch pwned)
APP_USER=$(touch pwned)
LPOD_PODMAN_BINARY=/bin/false
export APP_URL="https://stry.test" # the site
APP_PORT=8443 # the proxy's port
EOF
lpod stry open
calls "xdg-open https://stry.test:8443" 1
lpod stry export-cert "${HOME}/stry.crt"
calls "podman cp systemd-stry:/data/caddy/pki/authorities/local/root.crt ${HOME}/stry.crt" 1
holds ".env ran a command" test ! -e "${HOME}/pwned"
printf 'APP_URL=file:///etc/passwd\n' > "${HOME}/.env"
lpod_status stry open
exited_with 1
said "Set APP_URL to an http:// or https:// address."
calls_like "^xdg-open" 1
rm -f "${HOME}/.env"

test_case "the proxy only trusts forwarded headers behind another proxy"
CADDYFILE="${HOME}/.config/pods-proxy/Caddyfile"
lpod proxy settings set --stdin --json < <(echo '{"mode": "letsencrypt", "email": "admin@example.test"}') && check '.ok'
holds "the letsencrypt mode trusts X-Forwarded-* headers" test "$(grep -c trusted_proxies "$CADDYFILE")" -eq 0
lpod proxy settings set --stdin --json < <(echo '{"mode": "http"}') && check '.ok'
holds "the http mode doesn't trust the proxy in front of it" grep -q "trusted_proxies static private_ranges" "$CADDYFILE"

echo "${PASSED} passed, ${FAILED} failed"

[ "$FAILED" -eq 0 ]
