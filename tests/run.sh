#!/usr/bin/env bash
# Runs lpod's "--json" commands against stand-ins for podman, systemctl and
# journalctl (tests/stubs) that answer from the host in tests/host, and checks
# what lpod prints, that it keeps the keys of the contract's fixtures, and how
# often it asks the host. Needs bash and jq...

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LPOD="${ROOT}/bin/lpod"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

export PATH="${ROOT}/tests/stubs:${PATH}"
export LPOD_PODMAN_BINARY="${ROOT}/tests/stubs/podman"
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

echo "${PASSED} passed, ${FAILED} failed"

[ "$FAILED" -eq 0 ]
