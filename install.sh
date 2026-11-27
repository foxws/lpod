#!/usr/bin/env bash

# Installs or upgrades lpod. Run it again, or "lpod
# self-update", to upgrade:
#
#   curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
#
# Uninstall with:
#
#   curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash -s -- --uninstall
#
# LPOD_VERSION pins a release (e.g. v2.2.0), LPOD_BRANCH installs a branch
# instead (e.g. main, unverified, reporting its version as dev-BRANCH), and
# LPOD_INSTALL_DIR changes where lpod goes (default ~/.local/bin, or
# /usr/local/bin as root).
#
# On amd64 and arm64, a release installs the Go binary as lpod and the bash
# script next to it as lpod.sh, which the binary hands the commands it
# doesn't implement yet to. Elsewhere, for a branch, and for a release
# without binaries, the bash script is lpod itself...
#
# Everything runs from "main" on the last line, so a download cut off
# halfway through "curl | bash" never runs a partial script...

set -euo pipefail

REPOSITORY="https://github.com/foxws/lpod"
VERSION="${LPOD_VERSION:-latest}"
BRANCH="${LPOD_BRANCH:-}"

if [ -n "$BRANCH" ]; then
    VERSION="$BRANCH"
fi
FILES=(lpod)

if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="${LPOD_INSTALL_DIR:-/usr/local/bin}"
    SYSTEMCTL_CMD=(systemctl)
    SYSTEMD_UNIT_PATH="/etc/systemd/system"
else
    INSTALL_DIR="${LPOD_INSTALL_DIR:-$HOME/.local/bin}"
    SYSTEMCTL_CMD=(systemctl --user)
    SYSTEMD_UNIT_PATH="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
fi

function info {
    echo "  $*"
}

function warn {
    echo "  ! $*" >&2
}

function fail {
    echo "  x $*" >&2

    exit 1
}

# Whether the installer can ask a question, even when piped into bash...
function has_tty {
    ( : < /dev/tty ) 2>/dev/null
}

function has_systemd_session {
    "${SYSTEMCTL_CMD[@]}" show-environment >/dev/null 2>&1
}

# The name of the release's binary for this machine, or nothing when there's
# none for it...
function binary_name {
    case "$(uname -m)" in
        x86_64|amd64) echo "lpod-linux-amd64" ;;
        aarch64|arm64) echo "lpod-linux-arm64" ;;
    esac
}

function download_url {
    local NAME="$1"

    if [ -n "$BRANCH" ]; then
        echo "https://raw.githubusercontent.com/foxws/lpod/${BRANCH}/bin/${NAME}"
    elif [ "$VERSION" == "latest" ]; then
        echo "${REPOSITORY}/releases/latest/download/${NAME}"
    else
        echo "${REPOSITORY}/releases/download/v${VERSION#v}/${NAME}"
    fi
}

# Downloads a release asset to the given path, returning non-zero when it
# doesn't exist...
function fetch {
    local NAME="$1"
    local TARGET="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "$TARGET" "$(download_url "$NAME")"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$TARGET" "$(download_url "$NAME")"
    else
        fail "Neither curl nor wget is installed."
    fi
}

# Checks a download against the release's SHA256SUMS. A release is never
# installed unverified, so a missing or failed download of it stops here...
function verify {
    local NAME="$1"
    local FILE="$2"
    local SUMS="$3"
    local EXPECTED ACTUAL

    if [ -n "$BRANCH" ]; then
        warn "A branch has no SHA256SUMS, so ${NAME} from ${BRANCH} isn't verified."

        return 0
    fi

    if [ ! -s "$SUMS" ]; then
        fail "Could not download SHA256SUMS of ${VERSION}, so ${NAME} can't be verified. Nothing was installed. Releases before v2.2.0 have none."
    fi

    EXPECTED="$(awk -v name="$NAME" '$2 == name || $2 == "*" name { print $1 }' "$SUMS")"
    ACTUAL="$(sha256sum "$FILE" | cut -d ' ' -f 1)"

    if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
        fail "The checksum of ${NAME} doesn't match the release's SHA256SUMS. Nothing was installed."
    fi
}

# Warns about what commonly keeps rootless Quadlet services from working...
function check_podman {
    if ! command -v podman >/dev/null 2>&1; then
        warn "Podman isn't installed yet. lpod needs Podman 6.0 or later."

        return 0
    fi

    if ! podman quadlet --help >/dev/null 2>&1; then
        warn "\"podman quadlet\" isn't available. lpod needs Podman 6.0 or later."
    fi

    if [ "$(id -u)" -ne 0 ]; then
        local USER_NAME
        USER_NAME="$(id -un)"

        if ! grep -q "^${USER_NAME}:" /etc/subuid 2>/dev/null || ! grep -q "^${USER_NAME}:" /etc/subgid 2>/dev/null; then
            warn "${USER_NAME} has no /etc/subuid or /etc/subgid range, which rootless Podman needs. Add one with: sudo usermod --add-subuids 100000-165535 --add-subgids 100000-165535 ${USER_NAME}"
        fi
    fi
}

function do_install {
    local STAGING PREVIOUS CURRENT NAME
    STAGING="$(mktemp -d)"

    # shellcheck disable=SC2064
    trap "rm -rf '${STAGING}'" EXIT

    PREVIOUS=""

    if [ -x "${INSTALL_DIR}/lpod" ]; then
        PREVIOUS="$("${INSTALL_DIR}/lpod" --version 2>/dev/null || true)"
    fi

    echo "Installing lpod (${VERSION}) to ${INSTALL_DIR}"

    check_podman

    # Download and verify everything before replacing anything, so a failed
    # download or checksum leaves a working install as it was...
    if [ -z "$BRANCH" ]; then
        fetch SHA256SUMS "${STAGING}/SHA256SUMS" 2>/dev/null || rm -f "${STAGING}/SHA256SUMS"
    fi

    for NAME in "${FILES[@]}"; do
        fetch "$NAME" "${STAGING}/${NAME}" || fail "Could not download ${NAME} (${VERSION})."
        verify "$NAME" "${STAGING}/${NAME}" "${STAGING}/SHA256SUMS"
    done

    # The Go binary, which a branch, a release before v4.3.0 and other
    # architectures don't have. Without it, the bash script is lpod...
    local BINARY=""

    if [ -z "$BRANCH" ]; then
        BINARY="$(binary_name)"
    fi

    if [ -n "$BINARY" ]; then
        if fetch "$BINARY" "${STAGING}/${BINARY}" 2>/dev/null; then
            verify "$BINARY" "${STAGING}/${BINARY}" "${STAGING}/SHA256SUMS"
        else
            info "${VERSION} has no ${BINARY}, so lpod is the bash script."
            BINARY=""
        fi
    fi

    # A release has its version baked in; a branch says which branch it is...
    if [ -n "$BRANCH" ]; then
        sed -i "s/^LPOD_VERSION=\"dev\"\$/LPOD_VERSION=\"dev-${BRANCH//\//-}\"/" "${STAGING}/lpod"
    fi

    mkdir -p "$INSTALL_DIR"
    chmod +x "${STAGING}/lpod"

    if [ -n "$BINARY" ]; then
        chmod +x "${STAGING}/${BINARY}"

        # The script goes first, so the new binary finds it as soon as it
        # replaces the old lpod...
        mv "${STAGING}/lpod" "${INSTALL_DIR}/lpod.sh"
        mv "${STAGING}/${BINARY}" "${INSTALL_DIR}/lpod"
    else
        mv "${STAGING}/lpod" "${INSTALL_DIR}/lpod"
        rm -f "${INSTALL_DIR}/lpod.sh"
    fi

    # lpod-setup rendered laravel-podman presets, and was dropped in v4...
    rm -f "${INSTALL_DIR}/lpod-setup"

    CURRENT="$("${INSTALL_DIR}/lpod" --version)"

    if [ -n "$PREVIOUS" ] && [ "$PREVIOUS" != "$CURRENT" ]; then
        info "Upgraded ${PREVIOUS} to ${CURRENT}"
    else
        info "Installed ${CURRENT}"
    fi

    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *) warn "${INSTALL_DIR} isn't on your PATH. Add it to your shell profile to run lpod." ;;
    esac

    # The "lpod-idle@" timers of releases before v3 run "lpod idle", which
    # v4 no longer has. "lpod APP ondemand enable" writes an app's idle check
    # now...
    local UNIT

    for UNIT in "${SYSTEMD_UNIT_PATH}"/timers.target.wants/lpod-idle@*.timer; do
        [ -e "$UNIT" ] || [ -L "$UNIT" ] || continue

        UNIT="$(basename "$UNIT")"
        warn "${UNIT} runs the idle check of lpod before v3, which is gone. Turn it off: ${SYSTEMCTL_CMD[*]} disable --now ${UNIT}"
    done

    # Rootless services only start at boot, and keep running after logout, with linger...
    if [ "$(id -u)" -ne 0 ] && [ "$(loginctl show-user "$(id -un)" --property=Linger --value 2>/dev/null)" != "yes" ]; then
        if has_tty; then
            local ANSWER
            read -r -p "  Enable linger, so your services start at boot without logging in? [Y/n] " ANSWER < /dev/tty

            if [ -z "$ANSWER" ] || [[ "$ANSWER" =~ ^[Yy] ]]; then
                loginctl enable-linger "$(id -un)" && info "Enabled linger for $(id -un)"
            fi
        else
            warn "Linger is off. On a server, run once: loginctl enable-linger $(id -un)"
        fi
    fi
}

# Removes lpod, the lpod-setup of releases before v4 and the idle templates
# of releases before v3, after disabling every app's old idle check. It leaves the apps' own units, secrets and volumes alone...
function do_uninstall {
    local UNIT NAME

    echo "Uninstalling lpod from ${INSTALL_DIR}"

    if has_systemd_session; then
        for UNIT in "${SYSTEMD_UNIT_PATH}"/timers.target.wants/lpod-idle@*.timer; do
            [ -e "$UNIT" ] || [ -L "$UNIT" ] || continue

            "${SYSTEMCTL_CMD[@]}" disable --now "$(basename "$UNIT")" && info "Disabled $(basename "$UNIT")"
        done
    fi

    rm -f "${SYSTEMD_UNIT_PATH}/lpod-idle@.timer" "${SYSTEMD_UNIT_PATH}/lpod-idle@.service"

    if has_systemd_session; then
        "${SYSTEMCTL_CMD[@]}" daemon-reload
    fi

    for NAME in "${FILES[@]}" lpod.sh lpod-setup; do
        rm -f "${INSTALL_DIR:?}/${NAME}"
    done

    info "Removed lpod, lpod-setup and the lpod-idle@ templates. Your services, secrets and volumes are untouched."
}

function main {
    if [ "$(uname -s)" != "Linux" ]; then
        fail "Unsupported operating system [$(uname -s)]. lpod requires Linux with systemd; macOS and Windows, including WSL, are not supported."
    fi

    command -v systemctl >/dev/null 2>&1 || fail "systemd is required."

    case "${1:-}" in
        --uninstall)
            do_uninstall
            ;;

        "")
            do_install
            ;;

        *)
            fail "Unknown option: $1. Use --uninstall, or no option to install."
            ;;
    esac
}

main "$@"
