#!/usr/bin/env bash

# Installs or upgrades lpod and lpod-setup, then writes the systemd templates
# for "lpod idle". Run it again, or "lpod self-update", to upgrade:
#
#   curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
#
# Uninstall with:
#
#   curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash -s -- --uninstall
#
# LPOD_VERSION pins a release (e.g. v2.2.0), LPOD_INSTALL_DIR changes where
# the scripts go (default ~/.local/bin, or /usr/local/bin as root).
#
# Everything runs from "main" on the last line, so a download cut off
# halfway through "curl | bash" never runs a partial script...

set -euo pipefail

REPOSITORY="https://github.com/foxws/lpod"
VERSION="${LPOD_VERSION:-latest}"
FILES=(lpod lpod-setup)

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

function download_url {
    local NAME="$1"

    if [ "$VERSION" == "latest" ]; then
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

# Checks a download against the release's SHA256SUMS. Releases before
# SHA256SUMS was published can't be verified, so those only get a warning...
function verify {
    local NAME="$1"
    local FILE="$2"
    local SUMS="$3"
    local EXPECTED ACTUAL

    if [ ! -s "$SUMS" ]; then
        warn "This release has no SHA256SUMS, so ${NAME} can't be verified."

        return 0
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
        warn "Podman isn't installed yet. lpod needs Podman 5.3 or later."

        return 0
    fi

    if ! podman quadlet --help >/dev/null 2>&1; then
        warn "\"podman quadlet\" isn't available. lpod needs Podman 5.3 or later."
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
    fetch SHA256SUMS "${STAGING}/SHA256SUMS" 2>/dev/null || rm -f "${STAGING}/SHA256SUMS"

    for NAME in "${FILES[@]}"; do
        fetch "$NAME" "${STAGING}/${NAME}" || fail "Could not download ${NAME} (${VERSION})."
        verify "$NAME" "${STAGING}/${NAME}" "${STAGING}/SHA256SUMS"
    done

    mkdir -p "$INSTALL_DIR"

    for NAME in "${FILES[@]}"; do
        chmod +x "${STAGING}/${NAME}"
        mv "${STAGING}/${NAME}" "${INSTALL_DIR}/${NAME}"
    done

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

    if ! has_systemd_session; then
        warn "No systemd user session. Log in directly (not with su) and run: lpod idle setup"

        return 0
    fi

    "${INSTALL_DIR}/lpod" idle setup
    info "Wrote the lpod-idle@ systemd templates. Enable the idle check per app with: lpod idle enable APPLICATION"

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

# Removes lpod, lpod-setup and the idle templates, after disabling every
# app's idle check. It leaves the apps' own units, secrets and volumes alone...
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

    for NAME in "${FILES[@]}"; do
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
