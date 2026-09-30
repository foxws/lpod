#!/usr/bin/env bash

# Installs or upgrades lpod and lpod-setup, then writes the systemd templates
# for "lpod idle". Run it again to upgrade:
#
#   curl -fsSL https://github.com/foxws/lpod/releases/latest/download/install.sh | bash
#
# LPOD_VERSION pins a release (e.g. v2.2.0), LPOD_INSTALL_DIR changes where
# the scripts go (default ~/.local/bin, or /usr/local/bin as root)...

set -euo pipefail

REPOSITORY="https://github.com/foxws/lpod"
VERSION="${LPOD_VERSION:-latest}"

if [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="${LPOD_INSTALL_DIR:-/usr/local/bin}"
    SYSTEMCTL_CMD=(systemctl)
else
    INSTALL_DIR="${LPOD_INSTALL_DIR:-$HOME/.local/bin}"
    SYSTEMCTL_CMD=(systemctl --user)
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

function download_url {
    local NAME="$1"

    if [ "$VERSION" == "latest" ]; then
        echo "${REPOSITORY}/releases/latest/download/${NAME}"
    else
        echo "${REPOSITORY}/releases/download/v${VERSION#v}/${NAME}"
    fi
}

# Downloads to a temporary file first, so an interrupted download never
# replaces a working script...
function download {
    local NAME="$1"
    local TARGET="${INSTALL_DIR}/${NAME}"
    local TEMPORARY="${TARGET}.download"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 -o "$TEMPORARY" "$(download_url "$NAME")" || { rm -f "$TEMPORARY"; fail "Could not download ${NAME}."; }
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$TEMPORARY" "$(download_url "$NAME")" || { rm -f "$TEMPORARY"; fail "Could not download ${NAME}."; }
    else
        fail "Neither curl nor wget is installed."
    fi

    chmod +x "$TEMPORARY"
    mv "$TEMPORARY" "$TARGET"
}

if [ "$(uname -s)" != "Linux" ]; then
    fail "Unsupported operating system [$(uname -s)]. lpod requires Linux with systemd; macOS and Windows, including WSL, are not supported."
fi

command -v systemctl >/dev/null 2>&1 || fail "systemd is required."
command -v podman >/dev/null 2>&1 || warn "Podman isn't installed yet. lpod needs it with Quadlet support."

PREVIOUS=""

if [ -x "${INSTALL_DIR}/lpod" ]; then
    PREVIOUS="$("${INSTALL_DIR}/lpod" --version 2>/dev/null || true)"
fi

echo "Installing lpod (${VERSION}) to ${INSTALL_DIR}"

mkdir -p "$INSTALL_DIR"
download lpod
download lpod-setup

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

if ! "${SYSTEMCTL_CMD[@]}" show-environment >/dev/null 2>&1; then
    warn "No systemd user session. Log in directly (not with su) and run: lpod idle setup"

    exit 0
fi

"${INSTALL_DIR}/lpod" idle setup
info "Wrote the lpod-idle@ systemd templates. Enable the idle check per app with: lpod idle enable APPLICATION"

# Rootless services only start at boot, and keep running after logout, with linger...
if [ "$(id -u)" -ne 0 ] && [ "$(loginctl show-user "$USER" --property=Linger --value 2>/dev/null)" != "yes" ]; then
    if has_tty; then
        read -r -p "  Enable linger, so your services start at boot without logging in? [Y/n] " ANSWER < /dev/tty

        if [ -z "$ANSWER" ] || [[ "$ANSWER" =~ ^[Yy] ]]; then
            loginctl enable-linger "$USER" && info "Enabled linger for ${USER}"
        fi
    else
        warn "Linger is off. On a server, run once: loginctl enable-linger ${USER}"
    fi
fi
